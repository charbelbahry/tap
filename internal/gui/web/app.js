// Owner: Role C.
// One WebSocket message = one protocol line. Lines starting with "EVT" are
// events; any other line answers the oldest command in `pending`.
// Server text is always shown with textContent, never innerHTML.
"use strict";

const $ = (id) => document.getElementById(id);
const pending = []; // [{cmd, quiet}]
let ws;
let me = "";
let chatScope = "GLOBAL";

// ---------- helpers ----------

// "item.pilgrims_candle" -> "Pilgrims Candle"
function label(id) {
  return String(id).split(".").pop().replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function json(line) {
  try { return JSON.parse(line.slice(3)); } catch { return null; }
}

function addLine(listId, text) {
  const li = document.createElement("li");
  li.textContent = text;
  $(listId).appendChild(li);
  $(listId).scrollTop = $(listId).scrollHeight;
}

function btn(text, cmd) {
  const b = document.createElement("button");
  b.className = "btn";
  b.textContent = text;
  b.onclick = () => send(cmd);
  return b;
}

function img(src, fallback) {
  const i = document.createElement("img");
  i.src = src;
  i.onerror = () => { i.onerror = null; i.src = fallback; };
  return i;
}

// Fill a list with one <li> per id: icon, name, then action buttons.
function fillList(listId, ids, folder, actions) {
  const ul = $(listId);
  ul.replaceChildren();
  for (const id of ids) {
    const li = document.createElement("li");
    if (folder) li.append(img(`assets/${folder}/${id}.png`, `assets/${folder}/_default.png`));
    li.append(" " + label(id) + " ");
    for (const [text, cmd] of actions) li.append(btn(text, `${cmd} ${id}`));
    ul.appendChild(li);
  }
}

function send(cmd, quiet = false) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  pending.push({ cmd, quiet });
  ws.send(cmd);
  addLine("raw-log", "> " + cmd);
}

// ---------- replies ----------

function onReply(cmd, line, quiet) {
  const verb = cmd.split(" ")[0].toUpperCase();
  const ok = line.startsWith("OK");

  if (!ok) {
    if (verb === "CONNECT") $("login-error").textContent = line;
    else addLine("game-log", line);
    return;
  }

  switch (verb) {
    case "CONNECT":
      me = cmd.split(" ")[1];
      $("login").hidden = true;
      $("game").hidden = false;
      ["LOOK", "STATUS", "INVENTORY", "QUESTS", "WHO"].forEach((c) => send(c, true));
      setInterval(() => send("LOOK", true), 5000);
      break;
    case "LOOK": showRoom(json(line)); break;
    case "STATUS": showStatus(json(line)); break;
    case "INVENTORY":
      fillList("inventory", json(line) || [], "items", [["Drop", "DROP"]]);
      break;
    case "QUESTS":
      $("quests").replaceChildren();
      for (const q of json(line) || []) {
        addLine("quests", `${label(q.quest_id)} · ${q.status}${q.progress ? " " + q.progress : ""}`);
      }
      break;
    case "WHO": {
      const d = json(line);
      $("count-online").textContent = d && d.server != null ? d.server : line.split("=")[1];
      break;
    }
    case "TALK": {
      const d = json(line);
      const npc = cmd.split(" ").slice(1).join(" ");
      showStory(label(npc), d && d.dialogue ? d.dialogue : line.slice(3));
      break;
    }
    case "QUEST": {
      const d = json(line);
      if (d) showStory(label(d.quest_id), `${d.description} (${d.status})`);
      else addLine("game-log", line);
      send("QUESTS", true);
      send("STATUS", true);
      break;
    }
    case "MOVE": case "FLEE": case "TAKE": case "DROP": case "ATTACK": case "DEFEND":
      addLine("game-log", `${cmd} → ${line.slice(3)}`);
      ["LOOK", "STATUS", "INVENTORY"].forEach((c) => send(c, true));
      break;
    case "GROUP":
      if (line.includes("group=")) $("group-status").textContent = line.slice(3);
      if (cmd.toUpperCase() === "GROUP LEAVE") $("group-status").textContent = "No group";
      $("group-invite").replaceChildren();
      break;
    case "QUIT":
      break; // the socket closes right after
    default:
      if (!quiet) addLine("game-log", line);
  }
}

function showRoom(d) {
  if (!d || !d.room) return;
  $("room-name").textContent = d.room.name || label(d.room.id);
  $("room-desc").textContent = d.room.description || "";
  const roomImg = $("room-img");
  roomImg.onerror = () => { roomImg.onerror = null; roomImg.src = "assets/rooms/_default.png"; };
  roomImg.src = `assets/rooms/${d.room.id}.png`;

  fillList("items", d.items || [], "items", [["Take", "TAKE"]]);
  fillList("npcs", d.npcs || [], "npcs", [["Talk", "TALK"], ["Attack", "ATTACK"], ["Quest", "QUEST"]]);
  $("players").replaceChildren();
  for (const p of d.players || []) addLine("players", p);
  $("count-room").textContent = (d.players || []).length;

  $("exits").replaceChildren();
  for (const dir of Object.keys(d.room.exits || {})) $("exits").append(btn(dir, "MOVE " + dir));
}

function showStatus(d) {
  if (!d) return;
  const max = d.max_hp || 100;
  $("hp-fill").style.width = Math.max(0, Math.min(100, (d.hp / max) * 100)) + "%";
  $("hp-text").textContent = `${d.hp} / ${max}`;
  $("me").textContent = `${me} · ${d.status}`;
}

function showStory(title, text) {
  const h = document.createElement("h3");
  h.textContent = title;
  const p = document.createElement("p");
  p.textContent = text;
  $("story").replaceChildren(h, p);
}

// ---------- events ----------

function onEvent(line) {
  const w = line.split(" ");
  const [, scope, type] = w;

  if (type === "CHAT") {
    addLine("chat-log", `[${scope}] ${w[3]}: ${w.slice(4).join(" ")}`);
  } else if (scope === "STATS") {
    $("count-online").textContent = line.split("=")[1];
  } else if (scope === "GROUP" && type === "INVITE") {
    const leader = w[3];
    $("group-invite").replaceChildren(`Invite from ${leader} `, btn("Join", "GROUP JOIN " + leader));
  } else if (scope === "GROUP") {
    addLine("game-log", `${w[3]} ${type === "JOIN" ? "joined" : "left"} your group`);
  } else if (scope === "ROOM") {
    // PRESENCE, ITEM, COMBAT: log it and refresh the room
    addLine("game-log", w.slice(2).join(" "));
    send("LOOK", true);
  }
  // anything else is unknown: ignored (it is still in the raw log)
}

// ---------- connection ----------

function connect() {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(`${proto}://${location.host}/ws`);

  ws.onmessage = (e) => {
    const line = String(e.data).replace(/\r?\n$/, "");
    addLine("raw-log", "< " + line);
    if (line.startsWith("EVT")) return onEvent(line);
    const p = pending.shift();
    if (!p) return; // greeting "OK hello proto=1"
    onReply(p.cmd, line, p.quiet);
  };

  ws.onclose = () => {
    $("game").hidden = true;
    $("login").hidden = true;
    $("closed").hidden = false;
  };
}

// ---------- UI wiring ----------

$("game").hidden = true;
$("login").hidden = false;

$("login-form").onsubmit = (e) => {
  e.preventDefault();
  $("login-error").textContent = "";
  send("CONNECT " + $("login-name").value.trim());
};

document.querySelectorAll("#commands [data-cmd]").forEach((b) => {
  b.onclick = () => send(b.dataset.cmd);
});

document.querySelectorAll("#chat-tabs [data-scope]").forEach((b) => {
  b.onclick = () => { chatScope = b.dataset.scope; };
});

$("chat-form").onsubmit = (e) => {
  e.preventDefault();
  const msg = $("chat-input").value.trim();
  if (msg) send(`CHAT ${chatScope} ${msg}`);
  $("chat-input").value = "";
};

document.querySelectorAll("#log-tabs [data-log]").forEach((b) => {
  b.onclick = () => {
    $("game-log").hidden = b.dataset.log !== "game-log";
    $("raw-log").hidden = b.dataset.log !== "raw-log";
  };
});

$("raw-form").onsubmit = (e) => {
  e.preventDefault();
  const cmd = $("raw-input").value.trim();
  if (cmd) send(cmd);
  $("raw-input").value = "";
};

$("group-create").onclick = () => send("GROUP CREATE");
$("group-leave").onclick = () => send("GROUP LEAVE");
$("invite-form").onsubmit = (e) => {
  e.preventDefault();
  const name = $("invite-name").value.trim();
  if (name) send("GROUP INVITE " + name);
  $("invite-name").value = "";
};

$("reconnect").onclick = () => location.reload();

["chat-log", "game-log", "raw-log", "items", "npcs", "players", "inventory", "quests"]
  .forEach((id) => $(id).replaceChildren());

connect();
