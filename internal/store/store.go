// Package store carrega a pokedex/moves em memoria e cuida da persistencia do
// time em disco (data/team.json). Sem banco de dados.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"nickspokedex/internal/model"
)

// maxHistory limita quantas consultas ficam no historico.
const maxHistory = 60

// Learner descreve como um Pokemon aprende um determinado golpe.
type Learner struct {
	Slug   string
	Name   string
	Method string // "Lv 12", "TM", "Egg", "Tutor"
}

// Store guarda os dados carregados e o estado do time.
type Store struct {
	dataDir string

	Pokemon  []model.Pokemon
	bySlug   map[string]*model.Pokemon
	byDex    map[int][]*model.Pokemon // dex -> todas as formas (para o seletor de formas)
	Moves    map[string]model.Move
	Learners map[string][]Learner
	preEvo   map[string][]string // slug -> pre-evolucoes (slugs dos "pais")

	mu      sync.Mutex
	team    model.Team
	history []model.HistoryEntry
}

// Load le pokedex.json e moves.json de dataDir e monta os indices.
func Load(dataDir string) (*Store, error) {
	s := &Store{dataDir: dataDir, bySlug: map[string]*model.Pokemon{}, Moves: map[string]model.Move{}}

	if err := readJSON(filepath.Join(dataDir, "pokedex.json"), &s.Pokemon); err != nil {
		return nil, fmt.Errorf("carregando pokedex.json (rodou `go run . import`?): %w", err)
	}
	if err := readJSON(filepath.Join(dataDir, "moves.json"), &s.Moves); err != nil {
		return nil, fmt.Errorf("carregando moves.json: %w", err)
	}

	s.byDex = map[int][]*model.Pokemon{}
	for i := range s.Pokemon {
		p := &s.Pokemon[i]
		s.bySlug[p.Slug] = p
		s.byDex[p.Dex] = append(s.byDex[p.Dex], p)
	}
	s.buildLearners()
	s.buildEvoIndex()

	// team.json e history.json sao opcionais.
	_ = readJSON(filepath.Join(dataDir, "team.json"), &s.team)
	_ = readJSON(filepath.Join(dataDir, "history.json"), &s.history)

	return s, nil
}

func (s *Store) buildLearners() {
	s.Learners = map[string][]Learner{}
	add := func(move, method string, p *model.Pokemon) {
		s.Learners[move] = append(s.Learners[move], Learner{Slug: p.Slug, Name: p.Name, Method: method})
	}
	for i := range s.Pokemon {
		p := &s.Pokemon[i]
		for _, lm := range p.LevelMoves {
			add(lm.Move, fmt.Sprintf("Lv %d", lm.Level), p)
		}
		for _, m := range p.TMMoves {
			add(m, "TM", p)
		}
		for _, m := range p.EggMoves {
			add(m, "Egg", p)
		}
		for _, m := range p.TutorMoves {
			add(m, "Tutor", p)
		}
	}
}

func (s *Store) buildEvoIndex() {
	s.preEvo = map[string][]string{}
	for i := range s.Pokemon {
		p := &s.Pokemon[i]
		for _, e := range p.Evolutions {
			s.preEvo[e.To] = append(s.preEvo[e.To], p.Slug)
		}
	}
}

// EvoNode e um no da arvore de evolucao (suporta ramificacoes, ex.: Eevee).
type EvoNode struct {
	Slug       string
	Name       string
	Dex        int
	Types      []string
	SpriteSlug string // basename do sprite de forma ("" usa o sprite por Dex)
	Condition  string // condicao para chegar neste no vindo do pai ("" na raiz)
	Current    bool
	Next       []*EvoNode
}

// Flatten lineariza a arvore de evolucao em ordem (DFS), para o carrossel.
func (n *EvoNode) Flatten() []*EvoNode {
	var out []*EvoNode
	var walk func(*EvoNode)
	walk = func(x *EvoNode) {
		if x == nil {
			return
		}
		out = append(out, x)
		for _, c := range x.Next {
			walk(c)
		}
	}
	walk(n)
	return out
}

// Variants retorna todas as formas que compartilham a especie (mesmo Dex) do
// slug dado, na ordem da pokedex. Usado no seletor de formas. Retorna nil se so
// existe a forma base.
func (s *Store) Variants(slug string) []*model.Pokemon {
	p, ok := s.bySlug[slug]
	if !ok {
		return nil
	}
	group := s.byDex[p.Dex]
	if len(group) <= 1 {
		return nil
	}
	return group
}

// EvolutionFamily monta a linha evolutiva completa a partir de qualquer membro,
// subindo ate a forma-base e descendo por todas as evolucoes.
func (s *Store) EvolutionFamily(slug string) *EvoNode {
	if _, ok := s.bySlug[slug]; !ok {
		return nil
	}
	// sobe ate a raiz (forma-base).
	root := slug
	guard := map[string]bool{}
	for {
		parents := s.preEvo[root]
		if len(parents) == 0 || guard[root] {
			break
		}
		guard[root] = true
		root = parents[0]
	}
	return s.buildEvoNode(root, "", slug, map[string]bool{})
}

func (s *Store) buildEvoNode(cur, condition, current string, seen map[string]bool) *EvoNode {
	p, ok := s.bySlug[cur]
	if !ok || seen[cur] {
		return nil
	}
	seen[cur] = true
	n := &EvoNode{
		Slug: cur, Name: p.Name, Dex: p.Dex, Types: p.Types, SpriteSlug: p.SpriteSlug,
		Condition: condition, Current: cur == current,
	}
	for _, e := range p.Evolutions {
		if child := s.buildEvoNode(e.To, e.Condition, current, seen); child != nil {
			n.Next = append(n.Next, child)
		}
	}
	return n
}

// Get retorna um Pokemon pelo slug.
func (s *Store) Get(slug string) (*model.Pokemon, bool) {
	p, ok := s.bySlug[slug]
	return p, ok
}

// Move retorna o metadado de um golpe.
func (s *Store) Move(slug string) (model.Move, bool) {
	m, ok := s.Moves[slug]
	return m, ok
}

// Search filtra Pokemon por nome/slug/tipo (case-insensitive, substring).
func (s *Store) Search(q string) []model.Pokemon {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []model.Pokemon
	for i := range s.Pokemon {
		p := s.Pokemon[i]
		if q == "" ||
			strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(p.Slug, q) ||
			typesContain(p.Types, q) {
			out = append(out, p)
		}
	}
	return out
}

func typesContain(types []string, q string) bool {
	for _, t := range types {
		if t == q {
			return true
		}
	}
	return false
}

// ---------- time ----------

// Team retorna uma copia do time atual.
func (s *Store) Team() model.Team {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.team
}

// AddToTeam adiciona um slug ao time (max 6, sem duplicar).
func (s *Store) AddToTeam(slug string) error {
	if _, ok := s.bySlug[slug]; !ok {
		return fmt.Errorf("pokemon desconhecido: %s", slug)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.team.Members {
		if m == slug {
			return nil // ja esta no time
		}
	}
	if len(s.team.Members) >= 6 {
		return fmt.Errorf("o time ja tem 6 pokemon")
	}
	s.team.Members = append(s.team.Members, slug)
	return s.saveTeamLocked()
}

// RemoveFromTeam remove um slug do time.
func (s *Store) RemoveFromTeam(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.team.Members[:0]
	for _, m := range s.team.Members {
		if m != slug {
			out = append(out, m)
		}
	}
	s.team.Members = out
	return s.saveTeamLocked()
}

// ClearTeam esvazia o time.
func (s *Store) ClearTeam() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.team.Members = nil
	return s.saveTeamLocked()
}

// TeamPokemon retorna os Pokemon do time, na ordem salva.
func (s *Store) TeamPokemon() []model.Pokemon {
	s.mu.Lock()
	members := append([]string(nil), s.team.Members...)
	s.mu.Unlock()

	var out []model.Pokemon
	for _, slug := range members {
		if p, ok := s.bySlug[slug]; ok {
			out = append(out, *p)
		}
	}
	return out
}

func (s *Store) saveTeamLocked() error {
	return writeJSON(filepath.Join(s.dataDir, "team.json"), s.team)
}

// ---------- historico ----------

// RecordView registra uma consulta a um Pokemon (move pro topo, sem duplicar).
func (s *Store) RecordView(slug string) {
	if _, ok := s.bySlug[slug]; !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]model.HistoryEntry, 0, len(s.history)+1)
	out = append(out, model.HistoryEntry{Slug: slug, At: time.Now().Unix()})
	for _, e := range s.history {
		if e.Slug == slug {
			continue // remove ocorrencia antiga; a nova ja esta no topo
		}
		out = append(out, e)
		if len(out) >= maxHistory {
			break
		}
	}
	s.history = out
	_ = writeJSON(filepath.Join(s.dataDir, "history.json"), s.history)
}

// History retorna o historico de consultas (mais recente primeiro).
func (s *Store) History() []model.HistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.HistoryEntry(nil), s.history...)
}

// ClearHistory apaga o historico.
func (s *Store) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = nil
	return writeJSON(filepath.Join(s.dataDir, "history.json"), s.history)
}

// ---------- util ----------

// AllSorted retorna todos os Pokemon ordenados por numero da dex.
func (s *Store) AllSorted() []model.Pokemon {
	out := append([]model.Pokemon(nil), s.Pokemon...)
	sort.Slice(out, func(i, j int) bool { return out[i].Dex < out[j].Dex })
	return out
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
