const output = document.getElementById("output");
const historyEl = document.getElementById("history");
const serverHistoryEl = document.getElementById("serverHistory");
const authStatus = document.getElementById("authStatus");

let authKey = null; // текущий authKey после login

// --- REPL execution ---
document.getElementById("runBtn").addEventListener("click", async () => {
  const code = document.getElementById("codeInput").value.trim();
  if (!code) return;

  output.textContent = "Running...";

  try {
    const response = await fetch("/api/expr", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(authKey ? { "Authorization": "Bearer " + authKey } : {})
      },
      body: JSON.stringify({ expr: code })
    });

    const data = await response.json();
    const resultText = data.message
      ? `Error: ${data.message}${data.loc ? ` (at ${data.loc.source}:${data.loc.line}:${data.loc.column})` : ""}`
      : `Result: ${data.result}`;

    output.textContent = resultText;
    addHistoryItem(code, resultText);

  } catch (err) {
    const msg = `Fetch error: ${err.message}`;
    output.textContent = msg;
    addHistoryItem(code, msg);
  }
});

// --- Local command history ---
function addHistoryItem(command, result) {
  const item = document.createElement("div");
  item.className = "history-item";

  const cmd = document.createElement("div");
  cmd.className = "history-command";
  cmd.textContent = command;

  const res = document.createElement("div");
  res.className = "history-result";
  res.textContent = ">> " + result;

  item.appendChild(cmd);
  item.appendChild(res);
  historyEl.appendChild(item);

  historyEl.scrollTop = historyEl.scrollHeight;
}

// --- Login ---
document.getElementById("loginBtn").addEventListener("click", async () => {
  try {
    const username = prompt("Enter username (leave empty for anonymous):") || "";
    const password = username ? prompt("Enter password:") || "" : "";

    const response = await fetch("/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password })
    });

    const data = await response.json();
    if (data.message) {
      authStatus.textContent = "Login failed: " + data.message;
      authKey = null;
      return;
    }

    authStatus.textContent = username ? `Logged in as ${username}` : "Logged in (anonymous)";
    authKey = response.headers.get("X-Auth-Key") || null;

  } catch (err) {
    authStatus.textContent = `Login error: ${err.message}`;
    authKey = null;
  }
});

// --- Logout ---
document.getElementById("logoutBtn").addEventListener("click", async () => {
  if (!authKey) {
    authStatus.textContent = "Not authenticated";
    return;
  }

  try {
    const response = await fetch("/api/logout", {
      method: "POST",
      headers: {
        "Authorization": "Bearer " + authKey
      }
    });

    const data = await response.json();
    authStatus.textContent = data.message || "Not authenticated";
    authKey = null;

  } catch (err) {
    authStatus.textContent = `Logout error: ${err.message}`;
  }
});

// --- Load server history ---
document.getElementById("loadHistoryBtn").addEventListener("click", async () => {
  if (!authKey) {
    serverHistoryEl.textContent = "Login first to load server history.";
    return;
  }

  serverHistoryEl.textContent = "Loading from server...";
  try {
    const response = await fetch("/api/history", {
      headers: {
        "Authorization": "Bearer " + authKey
      }
    });

    const data = await response.json();
    if (data.message) {
      serverHistoryEl.textContent = `Error: ${data.message}`;
      return;
    }

    if (data.items && data.items.length > 0) {
      serverHistoryEl.textContent = data.items
        .map(h => `[${new Date(h.at).toLocaleTimeString()}] ${h.expr} => ${h.result}`)
        .join("\n");
    } else {
      serverHistoryEl.textContent = "No server history found.";
    }

  } catch (err) {
    serverHistoryEl.textContent = `Fetch error: ${err.message}`;
  }
});
