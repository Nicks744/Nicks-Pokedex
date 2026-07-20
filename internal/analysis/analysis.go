// Package analysis sugere natureza e builds para um Pokemon a partir das suas
// base stats e do movepool. Heuristico (nao é competitivo oficial), mas util.
package analysis

import (
	"sort"

	"nickspokedex/internal/model"
)

// Nature descreve o efeito de uma natureza (+plus / -minus, em codigos de stat).
type Nature struct {
	Name  string
	Plus  string // atk, def, spa, spd, spe ("" = neutra)
	Minus string
}

var natureByName = map[string]Nature{
	"Adamant": {"Adamant", "atk", "spa"},
	"Jolly":   {"Jolly", "spe", "spa"},
	"Brave":   {"Brave", "atk", "spe"},
	"Modest":  {"Modest", "spa", "atk"},
	"Timid":   {"Timid", "spe", "atk"},
	"Quiet":   {"Quiet", "spa", "spe"},
	"Bold":    {"Bold", "def", "atk"},
	"Impish":  {"Impish", "def", "spa"},
	"Calm":    {"Calm", "spd", "atk"},
	"Careful": {"Careful", "spd", "spa"},
}

var statLabel = map[string]string{
	"atk": "Atk", "def": "Def", "spa": "SpA", "spd": "SpD", "spe": "Vel", "hp": "HP",
}

func natureEffect(name string) string {
	n := natureByName[name]
	if n.Plus == "" || n.Plus == n.Minus {
		return "neutra"
	}
	return "+" + statLabel[n.Plus] + " / −" + statLabel[n.Minus]
}

// Build e uma sugestao de conjunto (papel + natureza + EVs + golpes).
type Build struct {
	Role         string
	Nature       string
	NatureEffect string
	EVs          string
	Moves        []model.Move
	Reason       string
}

// Suggest gera as builds recomendadas. A primeira build carrega a natureza que
// "mais combina" com o Pokemon.
func Suggest(p model.Pokemon, moveOf func(string) (model.Move, bool)) []Build {
	pool := movepool(p, moveOf)
	if len(pool) == 0 {
		return nil
	}
	bs := p.BaseStats
	physical := bs.Attack >= bs.SpAttack
	fast := bs.Speed >= 85

	var builds []Build

	// 1) Build ofensiva principal (define a natureza recomendada).
	if physical {
		n := "Adamant"
		if fast {
			n = "Jolly"
		}
		b := offensiveBuild("Atacante Físico", n, "Physical", pool, p.Types)
		b.Reason = "Aposta no Ataque (maior que o Sp. Atk) e na Velocidade para bater primeiro."
		builds = append(builds, b)
	} else {
		n := "Modest"
		if fast {
			n = "Timid"
		}
		b := offensiveBuild("Atacante Especial", n, "Special", pool, p.Types)
		b.Reason = "Aposta no Sp. Atk (maior que o Ataque) e na Velocidade para bater primeiro."
		builds = append(builds, b)
	}

	// 2) Build secundaria: defensiva (se for resistente) ou variante ofensiva.
	bulky := bs.HP+bs.Defence+bs.SpDefence >= 260 || maxInt(bs.Defence, bs.SpDefence) >= 95
	if bulky {
		n := defensiveNature(bs)
		defLabel := "Def"
		other := "SpD"
		if natureByName[n].Plus == "spd" {
			defLabel, other = "SpD", "Def"
		}
		builds = append(builds, Build{
			Role:         "Muralha / Suporte",
			Nature:       n,
			NatureEffect: natureEffect(n),
			EVs:          "252 HP / 252 " + defLabel + " / 4 " + other,
			Moves:        pickDefensive(pool, p.Types),
			Reason:       "Usa a alta resistência para segurar o campo, recuperar HP e desgastar o oponente.",
		})
	} else {
		cat, st := "Physical", "Atk"
		nn := "Adamant"
		if !physical {
			cat, st, nn = "Special", "SpA", "Modest"
		}
		b := offensiveBuild("Ofensivo com bulk", nn, cat, pool, p.Types)
		b.EVs = "252 HP / 252 " + st + " / 4 Vel"
		b.Reason = "Mesma pegada ofensiva, mas com HP para aguentar um golpe antes de revidar."
		builds = append(builds, b)
	}

	return builds
}

func offensiveBuild(role, nature, cat string, pool []model.Move, ptypes []string) Build {
	moves := pickOffensive(pool, ptypes, cat)

	// Prepende um golpe de aumento de status, se o Pokemon aprender.
	setup := setupSpecial
	if cat == "Physical" {
		setup = setupPhysical
	}
	for _, m := range pool {
		if setup[m.Slug] {
			moves = append([]model.Move{m}, moves...)
			break
		}
	}
	if len(moves) > 4 {
		moves = moves[:4]
	}

	evStat := "Atk"
	if cat == "Special" {
		evStat = "SpA"
	}
	return Build{
		Role:         role,
		Nature:       nature,
		NatureEffect: natureEffect(nature),
		EVs:          "252 " + evStat + " / 252 Vel / 4 HP",
		Moves:        moves,
	}
}

var setupPhysical = map[string]bool{
	"swordsdance": true, "dragondance": true, "bulkup": true,
	"honeclaws": true, "bellydrum": true, "coil": true, "victorydance": true,
}
var setupSpecial = map[string]bool{
	"nastyplot": true, "calmmind": true, "quiverdance": true, "tailglow": true,
}

// pickOffensive escolhe ate 4 golpes ofensivos da categoria, priorizando STAB e
// maior poder, com variedade de tipos.
func pickOffensive(pool []model.Move, ptypes []string, cat string) []model.Move {
	isSTAB := func(t string) bool {
		for _, x := range ptypes {
			if x == t {
				return true
			}
		}
		return false
	}
	var cand []model.Move
	for _, m := range pool {
		if m.Category == cat && m.Power > 0 {
			cand = append(cand, m)
		}
	}
	sort.SliceStable(cand, func(i, j int) bool {
		si, sj := isSTAB(cand[i].Type), isSTAB(cand[j].Type)
		if si != sj {
			return si // STAB primeiro
		}
		return cand[i].Power > cand[j].Power
	})

	var picks []model.Move
	usedType := map[string]bool{}
	for _, m := range cand { // variedade de tipos
		if len(picks) >= 4 {
			break
		}
		if usedType[m.Type] {
			continue
		}
		usedType[m.Type] = true
		picks = append(picks, m)
	}
	for _, m := range cand { // completa se sobrou espaço
		if len(picks) >= 4 {
			break
		}
		if !containsMove(picks, m.Slug) {
			picks = append(picks, m)
		}
	}
	return picks
}

// pickDefensive monta recuperacao + STAB + status.
func pickDefensive(pool []model.Move, ptypes []string) []model.Move {
	recovery := map[string]bool{"recover": true, "roost": true, "synthesis": true, "moonlight": true, "morningsun": true, "softboiled": true, "slackoff": true, "rest": true, "milkdrink": true, "wish": true, "shoreup": true, "strengthsap": true}
	status := map[string]bool{"toxic": true, "willowisp": true, "thunderwave": true, "leechseed": true, "spore": true, "stunspore": true, "glare": true, "substitute": true, "protect": true, "defog": true, "stealthrock": true, "spikes": true}
	isSTAB := func(t string) bool {
		for _, x := range ptypes {
			if x == t {
				return true
			}
		}
		return false
	}

	var out []model.Move
	add := func(m model.Move) {
		if !containsMove(out, m.Slug) {
			out = append(out, m)
		}
	}
	for _, m := range pool { // recuperacao
		if recovery[m.Slug] {
			add(m)
			break
		}
	}
	var best model.Move // melhor STAB ofensivo
	bp := 0
	for _, m := range pool {
		if m.Power > bp && isSTAB(m.Type) {
			best, bp = m, m.Power
		}
	}
	if bp > 0 {
		add(best)
	}
	for _, m := range pool { // status
		if len(out) >= 4 {
			break
		}
		if status[m.Slug] {
			add(m)
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	return out
}

// defensiveNature aumenta a melhor defesa e reduz o ataque que o Pokemon menos usa.
func defensiveNature(bs model.Stats) string {
	physicalUser := bs.Attack >= bs.SpAttack
	if bs.Defence >= bs.SpDefence {
		if physicalUser {
			return "Impish" // +Def -SpA
		}
		return "Bold" // +Def -Atk
	}
	if physicalUser {
		return "Careful" // +SpD -SpA
	}
	return "Calm" // +SpD -Atk
}

func movepool(p model.Pokemon, moveOf func(string) (model.Move, bool)) []model.Move {
	seen := map[string]bool{}
	var out []model.Move
	add := func(slug string) {
		if seen[slug] {
			return
		}
		if m, ok := moveOf(slug); ok {
			seen[slug] = true
			out = append(out, m)
		}
	}
	for _, lm := range p.LevelMoves {
		add(lm.Move)
	}
	for _, m := range p.TMMoves {
		add(m)
	}
	for _, m := range p.EggMoves {
		add(m)
	}
	for _, m := range p.TutorMoves {
		add(m)
	}
	return out
}

func containsMove(ms []model.Move, slug string) bool {
	for _, m := range ms {
		if m.Slug == slug {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
