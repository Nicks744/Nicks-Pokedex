// Package model define as estruturas de dados compartilhadas entre o
// importador (que gera os JSON) e o servidor (que os consome).
package model

// Stats representa as base stats de um Pokemon.
type Stats struct {
	HP        int `json:"hp"`
	Attack    int `json:"attack"`
	Defence   int `json:"defence"`
	SpAttack  int `json:"sp_attack"`
	SpDefence int `json:"sp_defence"`
	Speed     int `json:"speed"`
}

// Total soma todas as base stats (BST).
func (s Stats) Total() int {
	return s.HP + s.Attack + s.Defence + s.SpAttack + s.SpDefence + s.Speed
}

// Ability e uma habilidade do Pokemon.
type Ability struct {
	Name   string `json:"name"`
	Hidden bool   `json:"hidden"`
}

// LevelMove e um golpe aprendido por level-up.
type LevelMove struct {
	Level int    `json:"level"`
	Move  string `json:"move"` // slug do move (ex.: "vinewhip")
}

// Evolution descreve uma evolucao possivel.
type Evolution struct {
	To        string `json:"to"`        // slug do resultado
	Condition string `json:"condition"` // texto legivel (ex.: "Level 16")
}

// Pokemon e a entrada consolidada da pokedex.
type Pokemon struct {
	Dex         int         `json:"dex"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Types       []string    `json:"types"`
	BaseStats   Stats       `json:"baseStats"`
	Abilities   []Ability   `json:"abilities"`
	EggGroups   []string    `json:"eggGroups"`
	Description string      `json:"description"`
	Generation  string      `json:"generation"`
	Labels      []string    `json:"labels"`
	LevelMoves  []LevelMove `json:"levelMoves"`
	TMMoves     []string    `json:"tmMoves"`
	EggMoves    []string    `json:"eggMoves"`
	TutorMoves  []string    `json:"tutorMoves"`
	Evolutions  []Evolution `json:"evolutions"`
	Implemented bool        `json:"implemented"`
}

// Move e o metadado de um golpe (vindo do Pokemon Showdown).
type Move struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Category string `json:"category"` // Physical | Special | Status
	Power    int    `json:"power"`    // 0 = variavel/nenhum
	Accuracy int    `json:"accuracy"` // 0 = nao erra (true)
	PP       int    `json:"pp"`
	Desc     string `json:"desc"`
}

// Team e o time pessoal salvo em disco.
type Team struct {
	Name    string   `json:"name"`
	Members []string `json:"members"` // slugs, ate 6
}

// HistoryEntry e um registro de consulta a um Pokemon.
type HistoryEntry struct {
	Slug string `json:"slug"`
	At   int64  `json:"at"` // unix seconds
}
