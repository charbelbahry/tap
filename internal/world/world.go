// Package world loads the static game data from data/world.yaml.
//
// Owner: Role B. These types match the format in PLAN.md section 7
// and are a starting point: change them as needed.
package world

import "errors"

// World is the whole content of the world file.
type World struct {
	Map    Map              `yaml:"world"`
	Items  map[string]Item  `yaml:"items"`
	NPCs   map[string]NPC   `yaml:"npcs"`
	Quests map[string]Quest `yaml:"quests"`
}

// Map holds the rooms and the special locations.
type Map struct {
	Start    string          `yaml:"start"`
	SafeRoom string          `yaml:"safe_room"`
	Rooms    map[string]Room `yaml:"rooms"`
}

type Room struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Exits       map[string]string `yaml:"exits"` // direction -> room ID
	Items       []string          `yaml:"items"`
	NPCs        []string          `yaml:"npcs"`
}

type Item struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Obtainable  bool   `yaml:"obtainable"`
	DamageBonus int    `yaml:"damage_bonus"`
	MaxHPBonus  int    `yaml:"max_hp_bonus"`
}

type NPC struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Role        string   `yaml:"role"` // dialogue, quest_giver or enemy
	Dialogue    []string `yaml:"dialogue"`
	Stats       Stats    `yaml:"stats"`
}

type Stats struct {
	HP        int `yaml:"hp"`
	DamageMin int `yaml:"damage_min"`
	DamageMax int `yaml:"damage_max"`
}

type Quest struct {
	Title       string    `yaml:"title"`
	Giver       string    `yaml:"giver"`
	Description string    `yaml:"description"`
	Objective   Objective `yaml:"objective"`
	Needs       string    `yaml:"needs"` // quest ID that must be completed first
	Reward      Reward    `yaml:"reward"`
}

type Objective struct {
	Kill    string `yaml:"kill"`    // NPC ID
	Deliver string `yaml:"deliver"` // item ID
}

type Reward struct {
	DamageBonus int    `yaml:"damage_bonus"`
	MaxHPBonus  int    `yaml:"max_hp_bonus"`
	Item        string `yaml:"item"`
	Title       string `yaml:"title"`
}

// Load reads and validates the world file.
func Load(path string) (*World, error) {
	// TODO(B): read the file, yaml.Unmarshal into World, then call Validate.
	// Add the dependency with: go get gopkg.in/yaml.v3
	return nil, errors.New("world.Load is not implemented yet (Role B)")
}
