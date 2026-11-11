// Owner: Role C.
//
// Plan (PLAN.md, Role C):
// 1. Open a WebSocket to /ws. Each message is one protocol line.
// 2. Lines starting with "EVT" are events. Any other line answers the
//    oldest command in the pending queue.
// 3. Render room, inventory, chat and log. Use textContent for server text.
// 4. Send LOOK after room events and every 5 seconds.
"use strict";

const log = document.getElementById("log");
log.textContent = "The bridge is running. WebSocket support comes in Role A step 12.";
