const usernameEl = document.getElementById("username");
const passwordEl = document.getElementById("password");
const authStatus = document.getElementById("authStatus");
const registerBtn = document.getElementById("registerBtn");
const loginBtn = document.getElementById("loginBtn");
const logoutBtn = document.getElementById("logoutBtn");

const replSection = document.getElementById("repl");
const codeInput = document.getElementById("codeInput");
const runBtn = document.getElementById("runBtn");
const output = document.getElementById("output");

const historySection = document.getElementById("historySection");
const historyEl = document.getElementById("history");

let authKey = null;

// --- Utility to set auth cookie header ---
function authHeaders() {
    return authKey ? { "Authorization": "Bearer " + authKey } : {};
}

// --- Register ---
registerBtn.addEventListener("click", async () => {
    const res = await fetch("/api/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: usernameEl.value, password: passwordEl.value })
    });
    const data = await res.json();
    if (res.ok) {
        authStatus.textContent = "Registered successfully. Please login.";
    } else {
        authStatus.textContent = data.message;
    }
});

// --- Login ---
loginBtn.addEventListener("click", async () => {
    const res = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: usernameEl.value, password: passwordEl.value })
    });
    const data = await res.json();
    if (res.ok) {
        authKey = data.authKey;
        authStatus.textContent = "Logged in successfully!";
        logoutBtn.style.display = "inline-block";
        replSection.style.display = "block";
        historySection.style.display = "block";
        loadHistory();
    } else {
        authStatus.textContent = data.message;
    }
});

// --- Logout ---
logoutBtn.addEventListener("click", async () => {
    await fetch("/api/logout", { method: "POST", headers: authHeaders() });
    authKey = null;
    authStatus.textContent = "Logged out.";
    logoutBtn.style.display = "none";
    replSection.style.display = "none";
    historySection.style.display = "none";
    historyEl.innerHTML = "";
});

// --- Run REPL code ---
runBtn.addEventListener("click", async () => {
    const expr = codeInput.value.trim();
    if (!expr) return;
    const res = await fetch("/api/expr", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ expr })
    });
    const data = await res.json();
    if (res.ok) {
        output.textContent = data.result;
        appendHistory(expr, data.result);
    } else {
        output.textContent = data.message;
    }
});

// --- Load history ---
async function loadHistory() {
    const res = await fetch("/api/history", {
        method: "GET",
        headers: authHeaders()
    });
    if (!res.ok) return;
    const data = await res.json();
    historyEl.innerHTML = "";
    data.history.forEach(item => {
        const div = document.createElement("div");
        div.className = "history-item";
        div.textContent = `[${new Date(item.at).toLocaleString()}] ${item.expr} => ${item.result}`;
        historyEl.appendChild(div);
    });
}

// --- Append single item to history ---
function appendHistory(expr, result) {
    const div = document.createElement("div");
    div.className = "history-item";
    div.textContent = `[${new Date().toLocaleString()}] ${expr} => ${result}`;
    historyEl.prepend(div);
}
