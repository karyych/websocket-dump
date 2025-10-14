const $ = (s) => document.querySelector(s);
const log = (...a) => {
  const el = $("#log");
  el.textContent += a.join(" ") + "\n";
  el.scrollTop = el.scrollHeight;
};

let ws = null;
const setStatus = (s) => $("#status span").textContent = s;

// Подключение WebSocket
$("#btnConnect").onclick = () => {
  if (ws && ws.readyState === WebSocket.OPEN) return log("Уже подключен");

  const url = $("#endpoint").value.trim();
  log("Подключаюсь к", url);

  try {
    ws = new WebSocket(url);

    ws.onopen = () => {
      setStatus("connected");
      log("OPEN", url);
    };

    ws.onmessage = (e) => {
      const msg =
        typeof e.data === "string"
          ? e.data
          : `[binary] ${e.data.size}B`;
      log("RECV", msg);
    };

    ws.onclose = (e) => {
      setStatus("disconnected");
      log("CLOSE", e.code, e.reason || "");
    };

    ws.onerror = (e) => log("ERROR", e.message || e);
  } catch (err) {
    log("Connect error:", err);
    setStatus("error");
  }
};

$("#btnDisconnect").onclick = () => api("/api/close");

$("#btnClear").onclick = () => $("#log").textContent = "";

async function api(path) {
  try {
    const res = await fetch(path, { method: "POST" });
    const txt = await res.text();
    log("API", path, "→", txt.trim());
  } catch (err) {
    log("API error:", err);
  }
}

// Привязываем кнопки
$("#btnSendText").onclick = () => {
  const txt = $("#txt").value.trim() || "hello from URFU";
  api(`/api/sendText?msg=${encodeURIComponent(txt)}`);
};

$("#btnSendLong").onclick = () => api("/api/sendLong");

$("#btnSendBin").onclick = () => api("/api/sendBin?n=32");

$("#btnPing").onclick = () => api("/api/ping");
