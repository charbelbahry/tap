---
title: Game Rules & Formulas Reference
tags:
  - rules
  - combat
  - math
  - formulas
created: 2026-11-16
---

# ⚔️ Game Rules & Formulas Reference

Back to: [[00 - Dashboard]] | Plan: [[03 - Role B Plan & Milestones]] | World: [[05 - World Design & Lore]]

---

## 1. Combat System

Combat in TAP is strictly **turn-based**. Each command issued by a player represents one discrete combat turn. There are no background timers ticking during combat.

```
                   Player Issues Command (ATTACK / DEFEND / FLEE / MOVE)
                                            │
                                            ▼
                             1. Player Action Executes First
                                 (Deal damage / Set defense)
                                            │
                                            ▼
                           2. Target Status Evaluated
                                 ┌──────────┴──────────┐
                                 ▼                     ▼
                             NPC Dies             NPC Survives
                        • Award kill credit      • Enemy retaliates
                        • Start 45s respawn      • Check player HP (Death?)
```

### Turn Order & Initiative
1. **Player acts first**: The player's attack or defensive stance resolves immediately.
2. **Enemy counterattacks**: If the enemy survives, it counterattacks the player within the same turn.
3. **Simultaneous Attacks**: Ordered sequentially by the order the server receives the packets under `s.mu.Lock()`.
4. **Shared Fights**: Multiple players can attack the same enemy in the same room. The enemy retaliates against whichever player just struck it.

---

## 2. Damage Formulas

### Player Attack Damage
$$\text{Damage}_{\text{player}} = 8 + \text{rand}(0, 4) + \text{Bonuses}_{\text{items}} + \text{Bonuses}_{\text{quests}}$$

- **Base Hit**: 8 to 12 damage.
- **Modifiers**:
  - `Verger's Hammer`: $+6$ damage while carried.
  - Quest 1 reward: $+2$ permanent damage.
  - Quest 3 reward: $+5$ permanent damage.
  - Maximum prepared solo player hit: $8 + [0..4] + 6 + 2 + 5 = 21 \text{ to } 25 \text{ damage}$.

### Enemy Counterattack Damage
$$\text{Damage}_{\text{enemy}} = \text{rand}(\text{DamageMin}, \text{DamageMax})$$

If the player executed `DEFEND` on this turn:
$$\text{Damage}_{\text{received}} = \left\lfloor \frac{\text{Damage}_{\text{enemy}}}{2} \right\rfloor$$

---

## 3. Defensive & Evasion Mechanics

| Command | Action & Penalty | Reply Format |
|---|---|---|
| `DEFEND` | Stance: Incoming enemy counterattack on this turn deals **half damage**. | `OK {"hp":80,"damage_taken":5,"status":"combat"}` |
| `FLEE` | Escape: Chooses a random available exit from the room. Player has a **50% chance** of taking an unmitigated free hit from the enemy before escaping. | `OK room=loc.pilgrims_road` |
| `MOVE <dir>` | Retreat: Permitted even while in combat (so clients without a dedicated FLEE button never get stuck). The enemy deals a **guaranteed free counterattack hit** before the player leaves. | `OK room=<target-id>` |

---

## 4. Death & Respawn Mechanics

### Player Death
- **Trigger**: Player HP drops to $\le 0$.
- **Respawn Location**: Instantly transported to the safe room (`loc.votive_chapel`).
- **Health Penalty**: Respawns with **50 HP** (50% of base MaxHP).
- **Item Retention**: Carried items are **preserved** (items only drop to the floor on disconnect).
- **Combat Reset**: Combat engagement state is cleared.

### Safe Room Healing
- Whenever a living player enters `loc.votive_chapel` through normal movement (`MOVE`), their HP is automatically restored to full `MaxHP` (100 HP + bonuses).

---

## 5. Enemy Respawns & State

- Dead enemies respawn after **45 seconds**.
- **No Background Timers**: Respawn checks are evaluated lazily whenever a room is inspected or entered (`LOOK`, `MOVE`, `ATTACK`). If `time.Now() > RespawnAt`, the enemy is revived at full HP.

---

## 6. Boss Mechanics

### Baal (The Secret Boss)
- **Vulnerability Condition**: Baal cannot be harmed unless a player in the room is carrying **The Unspoken Question** (`item.unspoken_question`).
- **Hit Mitigation**: Attacks dealt without the item in the room deal **0 damage**, while Baal still counterattacks with full force ($15-25$ damage per hit).
- **Co-op Strategy**: A designated party member can hold the item and defend while teammates attack.
