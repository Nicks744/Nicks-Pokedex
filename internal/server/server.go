// Package server expoe a pokedex/wiki e o construtor de time via HTTP.
package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"nickspokedex/internal/analysis"
	"nickspokedex/internal/model"
	"nickspokedex/internal/store"
	"nickspokedex/internal/typechart"
)

// Server agrega o store e os templates.
type Server struct {
	st  *store.Store
	tpl *template.Template
}

// Run carrega os dados e sobe o servidor HTTP.
func Run(addr, dataDir string, webFS fs.FS) error {
	st, err := store.Load(dataDir)
	if err != nil {
		return err
	}
	tpl, err := template.New("").Funcs(funcMap()).ParseFS(webFS, "templates/*.html")
	if err != nil {
		return err
	}
	s := &Server{st: st, tpl: tpl}

	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(webFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Sprites e type gems gerados pela importacao.
	sprites := http.FileServer(http.Dir(filepath.Join(dataDir, "sprites")))
	mux.Handle("GET /sprites/", http.StripPrefix("/sprites/", sprites))
	gems := http.FileServer(http.Dir(filepath.Join(dataDir, "types")))
	mux.Handle("GET /types/", http.StripPrefix("/types/", gems))

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /pokemon/{slug}", s.handlePokemon)
	mux.HandleFunc("GET /move/{slug}", s.handleMove)
	mux.HandleFunc("GET /team", s.handleTeam)
	mux.HandleFunc("POST /team/add", s.handleTeamAdd)
	mux.HandleFunc("POST /team/remove", s.handleTeamRemove)
	mux.HandleFunc("POST /team/clear", s.handleTeamClear)
	mux.HandleFunc("GET /history", s.handleHistory)
	mux.HandleFunc("POST /history/clear", s.handleHistoryClear)

	// API JSON (consumida pela busca/filtros no cliente).
	mux.HandleFunc("GET /api/pokemon", s.handleAPIPokemon)
	mux.HandleFunc("GET /api/team", s.handleAPITeam)

	fmt.Printf("Nick's Pokedex no ar em http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// ---------- view models ----------

// Base guarda campos comuns a todas as paginas (Title/Active para o nav).
type Base struct {
	Title  string
	Active string
	Query  string
}

type indexView struct {
	Base
	Results  []model.Pokemon
	Count    int
	TeamSize int
}

type moveRow struct {
	Level    int
	HasLevel bool
	model.Move
}

type pokemonView struct {
	Base
	P       model.Pokemon
	LevelUp []moveRow
	TM      []moveRow
	Egg     []moveRow
	Tutor   []moveRow
	Weak    []typechart.Matchup
	Resist  []typechart.Matchup
	Immune  []typechart.Matchup
	InTeam  bool
	Family  *store.EvoNode
	Builds  []analysis.Build
}

type moveDetailView struct {
	Base
	M        model.Move
	Learners []store.Learner
}

type teamRow struct {
	Type      string
	Cells     []float64
	WeakCount int
}

type teamView struct {
	Base
	Members []model.Pokemon
	Rows    []teamRow
}

// ---------- handlers ----------

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results := s.st.Search(q)
	s.render(w, "index", indexView{
		Base:     Base{Title: "Pokédex", Active: "dex", Query: q},
		Results:  results,
		Count:    len(results),
		TeamSize: len(s.st.Team().Members),
	})
}

func (s *Server) handlePokemon(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, ok := s.st.Get(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.st.RecordView(slug)

	weak, resist, immune := typechart.DefensiveProfile(p.Types)

	inTeam := false
	for _, m := range s.st.Team().Members {
		if m == slug {
			inTeam = true
		}
	}

	view := pokemonView{
		Base:    Base{Title: p.Name, Active: "dex"},
		P:       *p,
		LevelUp: s.levelRows(p.LevelMoves),
		TM:      s.plainRows(p.TMMoves),
		Egg:     s.plainRows(p.EggMoves),
		Tutor:   s.plainRows(p.TutorMoves),
		Weak:    weak,
		Resist:  resist,
		Immune:  immune,
		InTeam:  inTeam,
		Family:  s.st.EvolutionFamily(slug),
		Builds:  analysis.Suggest(*p, s.st.Move),
	}
	s.render(w, "pokemon", view)
}

func (s *Server) levelRows(lms []model.LevelMove) []moveRow {
	rows := make([]moveRow, 0, len(lms))
	for _, lm := range lms {
		mv, _ := s.st.Move(lm.Move)
		if mv.Slug == "" {
			mv = model.Move{Slug: lm.Move, Name: lm.Move}
		}
		rows = append(rows, moveRow{Level: lm.Level, HasLevel: true, Move: mv})
	}
	return rows
}

func (s *Server) plainRows(slugs []string) []moveRow {
	rows := make([]moveRow, 0, len(slugs))
	for _, slug := range slugs {
		mv, _ := s.st.Move(slug)
		if mv.Slug == "" {
			mv = model.Move{Slug: slug, Name: slug}
		}
		rows = append(rows, moveRow{Move: mv})
	}
	return rows
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	mv, ok := s.st.Move(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	learners := s.st.Learners[slug]
	s.render(w, "move", moveDetailView{
		Base:     Base{Title: mv.Name, Active: ""},
		M:        mv,
		Learners: learners,
	})
}

func (s *Server) handleTeam(w http.ResponseWriter, r *http.Request) {
	members := s.st.TeamPokemon()

	rows := make([]teamRow, 0, len(typechart.Types))
	for _, atk := range typechart.Types {
		row := teamRow{Type: atk, Cells: make([]float64, len(members))}
		for i, m := range members {
			mult := typechart.DefensiveMultiplier(atk, m.Types)
			row.Cells[i] = mult
			if mult > 1 {
				row.WeakCount++
			}
		}
		rows = append(rows, row)
	}

	s.render(w, "team", teamView{
		Base:    Base{Title: "Meu Time", Active: "team"},
		Members: members,
		Rows:    rows,
	})
}

func (s *Server) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	slug := r.FormValue("slug")
	if err := s.st.AddToTeam(slug); err != nil {
		if isXHR(r) {
			writeJSONErr(w, err)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.finishTeamMutation(w, r)
}

func (s *Server) handleTeamRemove(w http.ResponseWriter, r *http.Request) {
	_ = s.st.RemoveFromTeam(r.FormValue("slug"))
	s.finishTeamMutation(w, r)
}

// finishTeamMutation responde JSON pra requisicoes fetch, ou redireciona no
// fluxo sem JavaScript (progressive enhancement).
func (s *Server) finishTeamMutation(w http.ResponseWriter, r *http.Request) {
	if isXHR(r) {
		writeJSON(w, map[string]any{"ok": true, "team": s.st.Team().Members})
		return
	}
	redirectBack(w, r)
}

// ---------- API JSON ----------

type apiPokemon struct {
	Dex    int      `json:"dex"`
	Name   string   `json:"name"`
	Slug   string   `json:"slug"`
	Types  []string `json:"types"`
	Gen    string   `json:"gen"`
	BST    int      `json:"bst"`
	Sprite string   `json:"sprite"`
}

func (s *Server) handleAPIPokemon(w http.ResponseWriter, r *http.Request) {
	all := s.st.AllSorted()
	out := make([]apiPokemon, 0, len(all))
	for _, p := range all {
		out = append(out, apiPokemon{
			Dex: p.Dex, Name: p.Name, Slug: p.Slug,
			Types: p.Types, Gen: p.Generation, BST: p.BaseStats.Total(),
			Sprite: spriteURL(p.Dex),
		})
	}
	writeJSON(w, out)
}

// spriteURL retorna o caminho do sprite servido localmente.
func spriteURL(dex int) string {
	return fmt.Sprintf("/sprites/%d.png", dex)
}

func (s *Server) handleAPITeam(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"team": s.st.Team().Members})
}

func isXHR(r *http.Request) bool {
	return r.Header.Get("X-Requested-With") == "fetch" ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (s *Server) handleTeamClear(w http.ResponseWriter, r *http.Request) {
	_ = s.st.ClearTeam()
	http.Redirect(w, r, "/team", http.StatusSeeOther)
}

// ---------- historico ----------

type historyItem struct {
	model.Pokemon
	When string
}

type historyView struct {
	Base
	Items []historyItem
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var items []historyItem
	for _, e := range s.st.History() {
		p, ok := s.st.Get(e.Slug)
		if !ok {
			continue
		}
		items = append(items, historyItem{
			Pokemon: *p,
			When:    relTime(now, time.Unix(e.At, 0)),
		})
	}
	s.render(w, "history", historyView{
		Base:  Base{Title: "Histórico", Active: "history"},
		Items: items,
	})
}

func (s *Server) handleHistoryClear(w http.ResponseWriter, r *http.Request) {
	_ = s.st.ClearHistory()
	http.Redirect(w, r, "/history", http.StatusSeeOther)
}

// relTime formata "há X" em pt-br.
func relTime(now, then time.Time) string {
	d := now.Sub(then)
	switch {
	case d < time.Minute:
		return "agora mesmo"
	case d < time.Hour:
		return fmt.Sprintf("há %d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("há %d h", int(d.Hours()))
	case d < 48*time.Hour:
		return "ontem"
	default:
		return fmt.Sprintf("há %d dias", int(d.Hours()/24))
	}
}

func redirectBack(w http.ResponseWriter, r *http.Request) {
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/team"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ---------- template funcs ----------

func funcMap() template.FuncMap {
	return template.FuncMap{
		"typeColor": typeColor,
		"title":     title,
		"mult":      multStr,
		"multClass": multClass,
		"statPct":   func(v int) int { return min(v*100/255, 100) },
		"statColor": statColor,
		"lower":     strings.ToLower,
		"dict":      dict,
		"sprite":    spriteURL,
		"typeGem":   func(t string) string { return "/types/" + strings.ToLower(t) + ".png" },
	}
}

// dict monta um map a partir de pares chave/valor, para passar multiplos
// argumentos a um sub-template.
func dict(kv ...any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("dict: numero impar de argumentos")
	}
	m := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: chave nao e string")
		}
		m[key] = kv[i+1]
	}
	return m, nil
}

// statColor vai do vermelho (baixo) ao verde (alto) conforme a base stat.
func statColor(v int) string {
	switch {
	case v >= 130:
		return "#22c55e"
	case v >= 100:
		return "#84cc16"
	case v >= 70:
		return "#eab308"
	case v >= 45:
		return "#f97316"
	default:
		return "#ef4444"
	}
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func multStr(m float64) string {
	switch m {
	case 0:
		return "0"
	case 0.25:
		return "¼"
	case 0.5:
		return "½"
	default:
		return fmt.Sprintf("%g", m)
	}
}

func multClass(m float64) string {
	switch {
	case m == 0:
		return "m0"
	case m < 0.5:
		return "m025"
	case m < 1:
		return "m05"
	case m == 1:
		return "m1"
	case m <= 2:
		return "m2"
	default:
		return "m4"
	}
}

var typeColors = map[string]string{
	"normal": "#A8A77A", "fire": "#EE8130", "water": "#6390F0", "electric": "#F7D02C",
	"grass": "#7AC74C", "ice": "#96D9D6", "fighting": "#C22E28", "poison": "#A33EA1",
	"ground": "#E2BF65", "flying": "#A98FF3", "psychic": "#F95587", "bug": "#A6B91A",
	"rock": "#B6A136", "ghost": "#735797", "dragon": "#6F35FC", "dark": "#705746",
	"steel": "#B7B7CE", "fairy": "#D685AD",
}

func typeColor(t string) string {
	if c, ok := typeColors[strings.ToLower(t)]; ok {
		return c
	}
	return "#777"
}
