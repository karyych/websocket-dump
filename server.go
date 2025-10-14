package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// ------------------- Базовые типы и глобалы -------------------
// Клиент вебсокет-сервера
type Client struct {
	c  *websocket.Conn
	mu sync.Mutex // сериализуем записи
}

func (cl *Client) Write(ctx context.Context, typ websocket.MessageType, data []byte) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.c.Write(ctx, typ, data)
}

// Глобальный реестр активных клиентов
var (
	clients   = make(map[*Client]struct{})
	clientsMu sync.RWMutex
)

func addClient(cl *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[cl] = struct{}{}
}

func delClient(cl *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, cl)
}

func forEachClient(fn func(*Client)) int {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	n := 0
	for cl := range clients {
		n++
		fn(cl)
	}
	return n
}

// ------------------- Вебсокет сервер -------------------
// Проверка на вебсокет-запрос
func isWS(r *http.Request) bool {
	conn := strings.ToLower(strings.Join(r.Header["Connection"], ","))
	up := strings.ToLower(r.Header.Get("Upgrade"))
	return strings.Contains(conn, "upgrade") && up == "websocket"
}

// Основной источник вебсокет-подключений
func chat(w http.ResponseWriter, r *http.Request) {
	if !isWS(r) {
		w.Header().Set("Connection", "Upgrade")
		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "Upgrade Required", http.StatusUpgradeRequired)
		return
	}

	opts := &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	}
	c, err := websocket.Accept(w, r, opts)
	if err != nil {
		log.Println("Accept:", err)
		return
	}

	cl := &Client{c: c}
	addClient(cl)
	defer func() {
		delClient(cl)
		cl.c.Close(websocket.StatusNormalClosure, "done")
	}()

	log.Printf("Client connected: %s %s", r.RemoteAddr, r.URL.Path)

	// периодические пинги
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	go func() {
		for range t.C {
			_ = cl.c.Ping(r.Context())
		}
	}()

	for {
		typ, data, err := cl.c.Read(r.Context())
		if err != nil {
			log.Println("Read:", err)
			return
		}
		switch typ {
		case websocket.MessageText:
			log.Printf("Text: %s", string(data))
			err = cl.Write(r.Context(), websocket.MessageText, []byte("echo: "+string(data)))
		case websocket.MessageBinary:
			log.Printf("Binary: %d bytes", len(data))
			err = cl.Write(r.Context(), websocket.MessageBinary, data)
		}
		if err != nil {
			log.Println("Write:", err)
			return
		}
	}
}

// POST /api/sendText отправка текстового сообщения всем клиентам
func apiSendText(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	if msg == "" {
		http.Error(w, "msg required", 400)
		return
	}
	n := forEachClient(func(cl *Client) {
		_ = cl.Write(r.Context(), websocket.MessageText, []byte("srv: "+msg))
	})
	fmt.Fprintf(w, "sent to %d clients\n", n)
}

// POST /api/sendLong (130B, ветка len=126) отправка длинного WebSocket-сообщения
func apiSendLong(w http.ResponseWriter, r *http.Request) {
	payload := make([]byte, 130)
	for i := range payload {
		payload[i] = 'X'
	}
	n := forEachClient(func(cl *Client) {
		_ = cl.Write(r.Context(), websocket.MessageText, payload)
	})
	fmt.Fprintf(w, "sent 130B text to %d clients\n", n)
}

// POST /api/sendBin (по умолчанию 32 байта) отправка короткого WebSocket-сообщения
func apiSendBin(w http.ResponseWriter, r *http.Request) {
	nBytes := 32
	if s := r.URL.Query().Get("n"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 1<<20 {
			nBytes = v
		}
	}
	buf := make([]byte, nBytes)
	for i := range buf {
		buf[i] = byte(i)
	}
	n := forEachClient(func(cl *Client) {
		_ = cl.Write(r.Context(), websocket.MessageBinary, buf)
	})
	fmt.Fprintf(w, "sent %dB binary to %d clients\n", nBytes, n)
}

// POST /api/ping — серверный Ping
func apiPing(w http.ResponseWriter, r *http.Request) {
	n := forEachClient(func(cl *Client) {
		_ = cl.c.Ping(r.Context())
	})
	fmt.Fprintf(w, "pinged %d clients\n", n)
}

// POST /api/close завершение WebSocket-соединения
func apiClose(w http.ResponseWriter, r *http.Request) {
	code := websocket.StatusNormalClosure
	if s := r.URL.Query().Get("code"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			code = websocket.StatusCode(v)
		}
	}
	reason := r.URL.Query().Get("reason")
	n := forEachClient(func(cl *Client) {
		_ = cl.c.Close(code, reason)
	})
	fmt.Fprintf(w, "closed %d clients with %d %q\n", n, code, reason)
}

func main() {
	// статика сайта
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// вебсокет по /chat
	http.HandleFunc("/chat", chat)

	// HTTP API для действий с клиентами
	http.HandleFunc("/api/sendText", apiSendText)
	http.HandleFunc("/api/sendLong", apiSendLong)
	http.HandleFunc("/api/sendBin", apiSendBin)
	http.HandleFunc("/api/ping", apiPing)
	http.HandleFunc("/api/close", apiClose)

	addr := "0.0.0.0:8765"
	fmt.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
