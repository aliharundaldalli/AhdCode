// Live check-ins for the dashboard.
//
// The page is complete without this script; it only adds check-ins and
// notices that arrive after the page loaded. Messages are written with
// textContent, never as HTML.
//
// Reconnection is explicit and visible. After a disconnect the status line
// counts down to the next attempt, backing off from 1 s to 30 s, and a
// "Reconnect now" button skips the wait. Nothing retries silently.
const statusLine = document.getElementById("live-status");
const list = document.getElementById("check-ins");
const reconnectButton = document.createElement("button");
reconnectButton.type = "button";
reconnectButton.textContent = "Reconnect now";
reconnectButton.hidden = true;
statusLine.after(reconnectButton);

let delay = 1000;
let countdown = null;
let failures = 0;

function show(text) {
  statusLine.textContent = text;
}

function addItem(text, className) {
  const item = document.createElement("li");
  item.textContent = text;
  if (className) {
    item.className = className;
  }
  list.prepend(item);
}

function connect() {
  clearInterval(countdown);
  reconnectButton.hidden = true;
  show("Connecting…");
  const scheme = location.protocol === "https:" ? "wss:" : "ws:";
  const socket = new WebSocket(`${scheme}//${location.host}/live`);

  socket.addEventListener("open", () => {
    delay = 1000;
    failures = 0;
    show("Live");
  });

  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    if (message.type === "check-in") {
      addItem(`${message.student} at ${message.at}`);
    } else if (message.type === "notice") {
      addItem(message.text, "notice");
    }
  });

  socket.addEventListener("close", (event) => {
    failures += 1;
    scheduleReconnect(event.code);
  });
}

function scheduleReconnect(code) {
  let remaining = Math.round(delay / 1000);
  const hint = failures >= 3 ? " If this continues, reload the page and sign in again." : "";
  const render = () => show(`Disconnected (code ${code}). Reconnecting in ${remaining} s.${hint}`);
  render();
  reconnectButton.hidden = false;
  countdown = setInterval(() => {
    remaining -= 1;
    if (remaining <= 0) {
      connect();
    } else {
      render();
    }
  }, 1000);
  delay = Math.min(delay * 2, 30000);
}

reconnectButton.addEventListener("click", connect);
connect();
