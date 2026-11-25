const API_URL = "http://localhost:8080/api";
const USER_ID_PATTERN = /^[a-zA-Z0-9_-]{1,64}$/;

const userInput = document.getElementById("userId");
const loadBtn = document.getElementById("loadBtn");
const statusEl = document.getElementById("status");
const listEl = document.getElementById("list");

async function loadNotifications() {
  const userId = userInput.value.trim();
  listEl.innerHTML = "";

  if (!USER_ID_PATTERN.test(userId)) {
    statusEl.textContent =
      "ID korisnika sme da sadrži samo slova, cifre, crticu i donju crtu (do 64 znaka).";
    return;
  }

  statusEl.textContent = "Učitavanje...";

  try {
    const response = await fetch(
      `${API_URL}/notifications?userId=${encodeURIComponent(userId)}`
    );
    if (!response.ok) {
      throw new Error("Greška servera: " + response.status);
    }
    const notifications = await response.json();

    if (notifications.length === 0) {
      statusEl.textContent = "Nema notifikacija.";
      return;
    }
    statusEl.textContent = "";

    for (const n of notifications) {
      const li = document.createElement("li");

      const type = document.createElement("span");
      type.className = "type";
      type.textContent = n.type;

      const message = document.createElement("p");
      message.textContent = n.message;

      const time = document.createElement("small");
      time.textContent = new Date(n.createdAt).toLocaleString("sr-RS");

      li.append(type, message, time);
      listEl.appendChild(li);
    }
  } catch (err) {
    statusEl.textContent = "Ne mogu da učitam notifikacije. Da li radi gateway?";
  }
}

loadBtn.addEventListener("click", loadNotifications);
loadNotifications();