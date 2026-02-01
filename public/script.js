const history = [];
const historyEl = document.getElementById("history");
const output = document.getElementById("output");

document.getElementById("runBtn").addEventListener("click", async () => {
  const code = document.getElementById("codeInput").value.trim();
  if (!code) return;

  output.textContent = "Running...";

  try {
    const response = await fetch("/api/expr", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ expr: code })
    });

    if (!response.ok) throw new Error(`Server error: ${response.status}`);

    const data = await response.json();
    let resultText;

    if (data.error) {
      resultText = `Error: ${data.error}`;
      output.textContent = resultText;
    } else if (data.result !== undefined) {
      resultText = `Result: ${data.result}`;
      output.textContent = resultText;
    } else {
      resultText = `Unknown response: ${JSON.stringify(data)}`;
      output.textContent = resultText;
    }

    const item = document.createElement("div");
    item.className = "history-item";

    const cmd = document.createElement("div");
    cmd.className = "history-command";
    cmd.textContent = code;

    const res = document.createElement("div");
    res.className = "history-result";
    res.textContent = resultText;

    item.appendChild(cmd);
    item.appendChild(res);

    historyEl.appendChild(item);

    historyEl.scrollTop = historyEl.scrollHeight;

  } catch (err) {
    const errorText = `Fetch error: ${err.message}`;
    output.textContent = errorText;

    const item = document.createElement("div");
    item.className = "history-item";

    const cmd = document.createElement("div");
    cmd.className = "history-command";
    cmd.textContent = code;

    const res = document.createElement("div");
    res.className = "history-result";
    res.textContent = errorText;

    item.appendChild(cmd);
    item.appendChild(res);

    historyEl.appendChild(item);
    historyEl.scrollTop = historyEl.scrollHeight;
  }
});