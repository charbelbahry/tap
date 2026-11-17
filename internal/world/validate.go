// Owner: Role B.
// Package world validates that all static game definitions are consistent.
package world

import "fmt"

// checkRef validates that key exists in m.
// Cyclomatic Complexity: 2 (base 1 + 1 if)
func checkRef[T any](m map[string]T, key, entityType, refContext string) error {
	if _, ok := m[key]; !ok {
		return fmt.Errorf("%s %q referenced in %s does not exist", entityType, key, refContext)
	}
	return nil
}

// Validate checks that every reference in the world points to something real:
// the start and safe rooms, every exit, every item and NPC placed in a room,
// and every quest's giver, objective, prerequisite and reward item.
// Cyclomatic Complexity: 4 (base 1 + 3 ifs)
func (w *World) Validate() error {
	if len(w.Map.Rooms) == 0 {
		return fmt.Errorf("world has no rooms")
	}
	if err := w.validateSpecialRooms(); err != nil {
		return err
	}
	if err := w.validateRooms(); err != nil {
		return err
	}
	return w.validateQuests()
}

// validateSpecialRooms checks start and safe room references.
// Cyclomatic Complexity: 2 (base 1 + 1 if)
func (w *World) validateSpecialRooms() error {
	if err := checkRef(w.Map.Rooms, w.Map.Start, "start room", "world"); err != nil {
		return err
	}
	return checkRef(w.Map.Rooms, w.Map.SafeRoom, "safe room", "world")
}

// validateRooms checks exits, items, and NPCs in each room.
// Cyclomatic Complexity: 3 (base 1 + 1 for + 1 if)
func (w *World) validateRooms() error {
	for id, r := range w.Map.Rooms {
		if err := w.validateRoom(id, r); err != nil {
			return err
		}
	}
	return nil
}

// validateRoom validates a single room's exits, items, and NPCs.
// Cyclomatic Complexity: 7 (base 1 + 3 loops + 3 ifs)
func (w *World) validateRoom(roomID string, r Room) error {
	ctx := "room " + roomID
	for dir, target := range r.Exits {
		if err := checkRef(w.Map.Rooms, target, "exit "+dir+" -> target room", ctx); err != nil {
			return err
		}
	}
	for _, item := range r.Items {
		if err := checkRef(w.Items, item, "item", ctx); err != nil {
			return err
		}
	}
	for _, npc := range r.NPCs {
		if err := checkRef(w.NPCs, npc, "npc", ctx); err != nil {
			return err
		}
	}
	return nil
}

// validateQuests checks giver, objectives, prerequisites, and rewards.
// Cyclomatic Complexity: 3 (base 1 + 1 for + 1 if)
func (w *World) validateQuests() error {
	for id, q := range w.Quests {
		if err := w.validateQuest(id, q); err != nil {
			return err
		}
	}
	return nil
}

// validateQuest validates a single quest's references.
// Cyclomatic Complexity: 5 (base 1 + 4 ifs)
func (w *World) validateQuest(questID string, q Quest) error {
	ctx := "quest " + questID
	if err := checkRef(w.NPCs, q.Giver, "giver npc", ctx); err != nil {
		return err
	}
	if err := w.validateQuestObjective(ctx, q.Objective); err != nil {
		return err
	}
	if q.Needs != "" {
		if err := checkRef(w.Quests, q.Needs, "prerequisite quest", ctx); err != nil {
			return err
		}
	}
	return w.validateQuestReward(ctx, q.Reward)
}

// validateQuestObjective validates the kill or deliver objective.
// Cyclomatic Complexity: 5 (base 1 + 4 ifs)
func (w *World) validateQuestObjective(ctx string, obj Objective) error {
	if obj.Kill != "" {
		if err := checkRef(w.NPCs, obj.Kill, "kill target npc", ctx); err != nil {
			return err
		}
		if w.NPCs[obj.Kill].Role != "enemy" {
			return fmt.Errorf("kill target %q in %s must have role 'enemy'", obj.Kill, ctx)
		}
	}
	if obj.Deliver != "" {
		return checkRef(w.Items, obj.Deliver, "delivery item", ctx)
	}
	return nil
}

// validateQuestReward validates that any reward item exists.
// Cyclomatic Complexity: 2 (base 1 + 1 if)
func (w *World) validateQuestReward(ctx string, rew Reward) error {
	if rew.Item != "" {
		return checkRef(w.Items, rew.Item, "reward item", ctx)
	}
	return nil
}
