const output = document.getElementById("output");
const historyEl = document.getElementById("history");

document.getElementById("runBtn").addEventListener("click", async () => {
  const code = document.getElementById("codeInput").value.trim();
  if (!code) return;

  output.textContent = "Running...";

  try {
    const response = await fetch("/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ code })
    });

    const data = await response.json();
    const resultText = data.error ? `Error: ${data.error}` : `Result: ${data.result}`;

    output.textContent = resultText;
    addHistoryItem(code, resultText);

  } catch (err) {
    const msg = `Fetch error: ${err.message}`;
    output.textContent = msg;
    addHistoryItem(code, msg);
  }
});

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
