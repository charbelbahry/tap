// Owner: Role B.

package world

// Validate checks that every reference in the world points to something real:
// the start and safe rooms, every exit, every item and NPC placed in a room,
// and every quest's giver, objective, prerequisite and reward item.
func (w *World) Validate() error {
	// TODO(B): return a clear error naming the first bad reference.
	return nil
}
