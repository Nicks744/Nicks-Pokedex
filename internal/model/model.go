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

// Pokemon e a entrada consolidada da pokedex. Cada forma (base, regional, mega,
// origin...) e uma entrada propria, com Slug unico e o mesmo Dex da especie.
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

	// Campos de forma (vazios na forma base sem variantes).
	Form       string `json:"form,omitempty"`       // rotulo curto: "Alola", "Galar", "Mega"... ("" ou "Base" na base)
	Aspect     string `json:"aspect,omitempty"`     // chave do aspecto Cobblemon (ex.: "alolan")
	BaseSlug   string `json:"baseSlug,omitempty"`   // slug da especie/forma-base (para agrupar as variantes)
	SpriteSlug string `json:"spriteSlug,omitempty"` // basename do sprite no Showdown (ex.: "vulpix-alola"); "" usa o sprite por Dex
	BattleOnly bool   `json:"battleOnly,omitempty"` // forma so de batalha (mega/primal)
}

// IsForm indica se esta entrada e uma variante (forma alternativa) da especie.
func (p Pokemon) IsForm() bool { return p.Aspect != "" || p.SpriteSlug != "" }

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
