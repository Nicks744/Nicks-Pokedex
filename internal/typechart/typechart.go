// Package typechart implementa a tabela de efetividade de tipos (Gen 6+)
// e o calculo defensivo usado na analise de fraquezas do time.
package typechart

import "sort"

// Types e a lista dos 18 tipos, em ordem canonica (usada na UI).
var Types = []string{
	"normal", "fire", "water", "electric", "grass", "ice",
	"fighting", "poison", "ground", "flying", "psychic", "bug",
	"rock", "ghost", "dragon", "dark", "steel", "fairy",
}

// chart[atacante][defensor] = multiplicador (apenas entradas != 1.0).
var chart = map[string]map[string]float64{
	"normal":   {"rock": 0.5, "ghost": 0, "steel": 0.5},
	"fire":     {"fire": 0.5, "water": 0.5, "grass": 2, "ice": 2, "bug": 2, "rock": 0.5, "dragon": 0.5, "steel": 2},
	"water":    {"fire": 2, "water": 0.5, "grass": 0.5, "ground": 2, "rock": 2, "dragon": 0.5},
	"electric": {"water": 2, "electric": 0.5, "grass": 0.5, "ground": 0, "flying": 2, "dragon": 0.5},
	"grass":    {"fire": 0.5, "water": 2, "grass": 0.5, "poison": 0.5, "ground": 2, "flying": 0.5, "bug": 0.5, "rock": 2, "dragon": 0.5, "steel": 0.5},
	"ice":      {"fire": 0.5, "water": 0.5, "grass": 2, "ice": 0.5, "ground": 2, "flying": 2, "dragon": 2, "steel": 0.5},
	"fighting": {"normal": 2, "ice": 2, "poison": 0.5, "flying": 0.5, "psychic": 0.5, "bug": 0.5, "rock": 2, "ghost": 0, "dark": 2, "steel": 2, "fairy": 0.5},
	"poison":   {"grass": 2, "poison": 0.5, "ground": 0.5, "rock": 0.5, "ghost": 0.5, "steel": 0, "fairy": 2},
	"ground":   {"fire": 2, "electric": 2, "grass": 0.5, "poison": 2, "flying": 0, "bug": 0.5, "rock": 2, "steel": 2},
	"flying":   {"electric": 0.5, "grass": 2, "fighting": 2, "bug": 2, "rock": 0.5, "steel": 0.5},
	"psychic":  {"fighting": 2, "poison": 2, "psychic": 0.5, "dark": 0, "steel": 0.5},
	"bug":      {"fire": 0.5, "grass": 2, "fighting": 0.5, "poison": 0.5, "flying": 0.5, "psychic": 2, "ghost": 0.5, "dark": 2, "steel": 0.5, "fairy": 0.5},
	"rock":     {"fire": 2, "ice": 2, "fighting": 0.5, "ground": 0.5, "flying": 2, "bug": 2, "steel": 0.5},
	"ghost":    {"normal": 0, "psychic": 2, "ghost": 2, "dark": 0.5},
	"dragon":   {"dragon": 2, "steel": 0.5, "fairy": 0},
	"dark":     {"fighting": 0.5, "psychic": 2, "ghost": 2, "dark": 0.5, "fairy": 0.5},
	"steel":    {"fire": 0.5, "water": 0.5, "electric": 0.5, "ice": 2, "rock": 2, "steel": 0.5, "fairy": 2},
	"fairy":    {"fire": 0.5, "fighting": 2, "poison": 0.5, "dragon": 2, "dark": 2, "steel": 0.5},
}

// Effectiveness retorna o multiplicador de um tipo atacante contra um defensor.
func Effectiveness(attacker, defender string) float64 {
	if m, ok := chart[attacker]; ok {
		if v, ok := m[defender]; ok {
			return v
		}
	}
	return 1
}

// DefensiveMultiplier retorna o dano que um Pokemon de `defTypes` recebe de um
// golpe do tipo `attacker` (produto dos tipos defensivos).
func DefensiveMultiplier(attacker string, defTypes []string) float64 {
	mult := 1.0
	for _, d := range defTypes {
		mult *= Effectiveness(attacker, d)
	}
	return mult
}

// Matchup e o resultado da analise defensiva para cada tipo atacante.
type Matchup struct {
	Type       string  `json:"type"`
	Multiplier float64 `json:"multiplier"`
}

// DefensiveProfile calcula, para um conjunto de tipos defensivos, como cada
// tipo atacante se comporta. Retorna listas ordenadas de fraquezas (>1),
// resistencias (<1) e imunidades (0).
func DefensiveProfile(defTypes []string) (weak, resist, immune []Matchup) {
	for _, atk := range Types {
		m := DefensiveMultiplier(atk, defTypes)
		mu := Matchup{Type: atk, Multiplier: m}
		switch {
		case m == 0:
			immune = append(immune, mu)
		case m > 1:
			weak = append(weak, mu)
		case m < 1:
			resist = append(resist, mu)
		}
	}
	// Mais perigoso / mais util primeiro.
	sort.SliceStable(weak, func(i, j int) bool { return weak[i].Multiplier > weak[j].Multiplier })
	sort.SliceStable(resist, func(i, j int) bool { return resist[i].Multiplier < resist[j].Multiplier })
	return weak, resist, immune
}
