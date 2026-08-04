// Package server expoe a pokedex/wiki via HTTP e tambem gera a versao estatica
// (para hospedar no GitHub Pages). Time e Historico sao client-side (localStorage).
package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"nickspokedex/internal/analysis"
	"nickspokedex/internal/model"
	"nickspokedex/internal/store"
	"nickspokedex/internal/typechart"
)

// Server agrega o store, os templates e o base path (para o GitHub Pages).
type Server struct {
	st   *store.Store
	tpl  *template.Template
	base string // "/" local, "/repo/" no Pages — vai no <base href>
}

func newServer(dataDir string, webFS fs.FS, base string) (*Server, error) {
	st, err := store.Load(dataDir)
	if err != nil {
		return nil, err
	}
	if base == "" {
		base = "/"
	}
	tpl, err := template.New("").Funcs(funcMap()).ParseFS(webFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{st: st, tpl: tpl, base: base}, nil
}

func (s *Server) mux(dataDir string, webFS fs.FS) *http.ServeMux {
	mux := http.NewServeMux()
	staticFS, _ := fs.Sub(webFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Sprites e type gems gerados pela importacao.
	mux.Handle("GET /sprites/", http.StripPrefix("/sprites/", http.FileServer(http.Dir(filepath.Join(dataDir, "sprites")))))
	mux.Handle("GET /types/", http.StripPrefix("/types/", http.FileServer(http.Dir(filepath.Join(dataDir, "types")))))
	mux.Handle("GET /itemtex/", http.StripPrefix("/itemtex/", http.FileServer(http.Dir(filepath.Join(dataDir, "itemtex")))))

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /pokemon/{slug}", s.handlePokemon)
	mux.HandleFunc("GET /moves", s.handleMoves)
	mux.HandleFunc("GET /move/{slug}", s.handleMove)
	mux.HandleFunc("GET /items", s.handleItems)
	mux.HandleFunc("GET /berries", s.handleBerries)
	mux.HandleFunc("GET /team", s.handleTeam)
	mux.HandleFunc("GET /history", s.handleHistory)

	// API JSON consumida pela busca no cliente. O caminho .json e o usado pela
	// versao estatica (arquivo real em docs/api/*.json).
	mux.HandleFunc("GET /api/pokemon", s.handleAPIPokemon)
	mux.HandleFunc("GET /api/pokemon.json", s.handleAPIPokemon)
	mux.HandleFunc("GET /api/moves", s.handleAPIMoves)
	mux.HandleFunc("GET /api/moves.json", s.handleAPIMoves)
	mux.HandleFunc("GET /api/items", s.handleAPIItems)
	mux.HandleFunc("GET /api/items.json", s.handleAPIItems)
	return mux
}

// Run carrega os dados e sobe o servidor HTTP.
func Run(addr, dataDir string, webFS fs.FS) error {
	s, err := newServer(dataDir, webFS, "/")
	if err != nil {
		return err
	}
	fmt.Printf("Nick's Pokedex no ar em http://localhost%s\n", addr)
	return http.ListenAndServe(addr, s.mux(dataDir, webFS))
}

// ---------- view models ----------

// Base guarda campos comuns a todas as paginas (nav + base path para URLs).
type Base struct {
	Title   string
	Active  string
	Query   string
	BaseURL string
}

func (s *Server) base0(title, active string) Base {
	return Base{Title: title, Active: active, BaseURL: s.base}
}

type indexView struct {
	Base
}

type moveRow struct {
	Level    int
	HasLevel bool
	model.Move
}

type pokemonView struct {
	Base
	P          model.Pokemon
	LevelUp    []moveRow
	TM         []moveRow
	Egg        []moveRow
	Tutor      []moveRow
	Weak       []typechart.Matchup
	Resist     []typechart.Matchup
	Immune     []typechart.Matchup
	Family     *store.EvoNode
	Line       []*store.EvoNode // familia linearizada (carrossel)
	Variants   []*model.Pokemon
	Encounters []encGroup // "onde encontrar", agrupado por mod
	Builds     []analysis.Build
}

// encGroup agrupa os encontros de um Pokemon por fonte (Cobblemon/Pixelmon).
type encGroup struct {
	Source string
	Items  []model.Encounter
}

// groupEncounters agrupa por fonte preservando a ordem ja definida no importer.
func groupEncounters(es []model.Encounter) []encGroup {
	var groups []encGroup
	byIdx := map[string]int{}
	for _, e := range es {
		i, ok := byIdx[e.Source]
		if !ok {
			byIdx[e.Source] = len(groups)
			groups = append(groups, encGroup{Source: e.Source})
			i = len(groups) - 1
		}
		groups[i].Items = append(groups[i].Items, e)
	}
	return groups
}

type moveDetailView struct {
	Base
	M        model.Move
	Learners []store.Learner
}

// ---------- handlers ----------

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	b := s.base0("Pokédex", "dex")
	b.Query = r.URL.Query().Get("q")
	s.render(w, "index", indexView{Base: b})
}

func (s *Server) handlePokemon(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	p, ok := s.st.Get(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	weak, resist, immune := typechart.DefensiveProfile(p.Types)
	family := s.st.EvolutionFamily(slug)
	s.render(w, "pokemon", pokemonView{
		Base:       s.base0(p.Name, "dex"),
		P:          *p,
		LevelUp:    s.levelRows(p.LevelMoves),
		TM:         s.plainRows(p.TMMoves),
		Egg:        s.plainRows(p.EggMoves),
		Tutor:      s.plainRows(p.TutorMoves),
		Weak:       weak,
		Resist:     resist,
		Immune:     immune,
		Family:     family,
		Line:       family.Flatten(),
		Variants:   s.st.Variants(slug),
		Encounters: groupEncounters(p.Encounters),
		Builds:     analysis.Suggest(*p, s.st.Move),
	})
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

// handleMoves renderiza o shell do browser de ataques; a lista vem do
// api/moves.json via moves.js (funciona tambem no site estatico).
func (s *Server) handleMoves(w http.ResponseWriter, r *http.Request) {
	b := s.base0("Ataques", "moves")
	b.Query = r.URL.Query().Get("q")
	s.render(w, "moves", indexView{Base: b})
}

// handleItems e handleBerries renderizam shells; a lista vem de api/items.json.
func (s *Server) handleItems(w http.ResponseWriter, r *http.Request) {
	b := s.base0("Itens", "items")
	b.Query = r.URL.Query().Get("q")
	s.render(w, "items", indexView{Base: b})
}

func (s *Server) handleBerries(w http.ResponseWriter, r *http.Request) {
	s.render(w, "berries", indexView{Base: s.base0("Berries & Sucos", "berries")})
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	mv, ok := s.st.Move(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.render(w, "move", moveDetailView{
		Base:     s.base0(mv.Name, ""),
		M:        mv,
		Learners: s.st.Learners[slug],
	})
}

// Time e Historico sao renderizados como shells; o conteudo vem do localStorage
// via JS (team.js / history.js), o que funciona tambem no site estatico.
func (s *Server) handleTeam(w http.ResponseWriter, r *http.Request) {
	s.render(w, "team", indexView{Base: s.base0("Meu Time", "team")})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.render(w, "history", indexView{Base: s.base0("Histórico", "history")})
}

// ---------- API JSON ----------

type apiPokemon struct {
	Dex         int      `json:"dex"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Types       []string `json:"types"`
	Gen         string   `json:"gen"`
	BST         int      `json:"bst"`
	Sprite      string   `json:"sprite"`
	ShinySprite string   `json:"shiny"`
	Form        string   `json:"form,omitempty"`
	IsForm      bool     `json:"isForm,omitempty"`
}

// handleAPIMoves serve todos os golpes (ordenados por nome) para o browser de
// ataques. Reaproveita o model.Move (que ja tem as tags JSON corretas).
func (s *Server) handleAPIMoves(w http.ResponseWriter, r *http.Request) {
	out := make([]model.Move, 0, len(s.st.Moves))
	for _, mv := range s.st.Moves {
		out = append(out, mv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, out)
}

// handleAPIItems serve o catalogo de itens do Pixelmon (ja ordenado no import).
func (s *Server) handleAPIItems(w http.ResponseWriter, r *http.Request) {
	items := s.st.Items
	if items == nil {
		items = []model.Item{}
	}
	writeJSON(w, items)
}

func (s *Server) handleAPIPokemon(w http.ResponseWriter, r *http.Request) {
	all := s.st.AllSorted()
	out := make([]apiPokemon, 0, len(all))
	for _, p := range all {
		out = append(out, apiPokemon{
			Dex: p.Dex, Name: p.Name, Slug: p.Slug,
			Types: p.Types, Gen: p.Generation, BST: p.BaseStats.Total(),
			Sprite:      spriteURL(p.Dex, p.SpriteSlug),
			ShinySprite: spriteShinyURL(p.Dex, p.SpriteSlug),
			Form:        p.Form,
			IsForm:      p.IsForm(),
		})
	}
	writeJSON(w, out)
}

// spriteURL retorna o caminho RELATIVO do sprite normal (resolvido pelo <base
// href>). Formas usam o sprite por nome (Showdown) em sprites/forms/.
func spriteURL(dex int, spriteSlug string) string {
	if spriteSlug != "" {
		return "sprites/forms/" + spriteSlug + ".png"
	}
	return fmt.Sprintf("sprites/%d.png", dex)
}

// spriteShinyURL e o equivalente shiny de spriteURL.
func spriteShinyURL(dex int, spriteSlug string) string {
	if spriteSlug != "" {
		return "sprites/forms/shiny/" + spriteSlug + ".png"
	}
	return fmt.Sprintf("sprites/shiny/%d.png", dex)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ---------- geracao estatica (GitHub Pages) ----------

// BuildStatic gera o site estatico completo em outDir, reaproveitando os mesmos
// handlers via httptest. base e o prefixo de URL (ex.: "/Nicks-Pokedex/").
func BuildStatic(dataDir string, webFS fs.FS, outDir, base string) error {
	if base == "" {
		base = "/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	s, err := newServer(dataDir, webFS, base)
	if err != nil {
		return err
	}
	h := s.mux(dataDir, webFS)

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	// .nojekyll evita o processamento do Jekyll no GitHub Pages.
	if err := writeFile(filepath.Join(outDir, ".nojekyll"), nil); err != nil {
		return err
	}

	// Paginas HTML (usa index.html dentro de cada pasta -> URLs limpas).
	pages := map[string]string{
		"/":        "index.html",
		"/moves":   "moves/index.html",
		"/items":   "items/index.html",
		"/berries": "berries/index.html",
		"/team":    "team/index.html",
		"/history": "history/index.html",
	}
	for _, p := range s.st.AllSorted() {
		pages["/pokemon/"+p.Slug] = "pokemon/" + p.Slug + "/index.html"
	}
	for slug := range s.st.Moves {
		pages["/move/"+slug] = "move/" + slug + "/index.html"
	}

	n := 0
	for url, out := range pages {
		body, code := snapshot(h, url)
		if code != http.StatusOK {
			return fmt.Errorf("gerando %s: status %d", url, code)
		}
		if err := writeFile(filepath.Join(outDir, out), body); err != nil {
			return err
		}
		n++
	}
	fmt.Printf("paginas geradas: %d\n", n)

	// API JSON estatica.
	for _, api := range []string{"pokemon", "moves", "items"} {
		body, code := snapshot(h, "/api/"+api+".json")
		if code != http.StatusOK {
			return fmt.Errorf("gerando api %s: status %d", api, code)
		}
		if err := writeFile(filepath.Join(outDir, "api/"+api+".json"), body); err != nil {
			return err
		}
	}

	// 404 (GitHub Pages usa 404.html na raiz).
	if body, code := snapshot(h, "/"); code == http.StatusOK {
		_ = writeFile(filepath.Join(outDir, "404.html"), body)
	}

	// Assets: static (embed), sprites e types (disco).
	if err := copyFSDir(webFS, "static", filepath.Join(outDir, "static")); err != nil {
		return err
	}
	for _, d := range []string{"sprites", "types", "itemtex"} {
		if err := copyDiskDir(filepath.Join(dataDir, d), filepath.Join(outDir, d)); err != nil {
			return err
		}
	}
	fmt.Printf("assets copiados. site estatico em %s (base %s)\n", outDir, base)
	return nil
}

func snapshot(h http.Handler, url string) ([]byte, int) {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Body.Bytes(), rec.Code
}

func writeFile(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func copyFSDir(fsys fs.FS, sub, dst string) error {
	sysFS, err := fs.Sub(fsys, sub)
	if err != nil {
		return err
	}
	return fs.WalkDir(sysFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(sysFS, p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(dst, filepath.FromSlash(p)), data)
	})
}

func copyDiskDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(dst, rel), data)
	})
}

// ---------- template funcs ----------

func funcMap() template.FuncMap {
	return template.FuncMap{
		"typeColor":   typeColor,
		"title":       title,
		"mult":        multStr,
		"multClass":   multClass,
		"statPct":     func(v int) int { return min(v*100/255, 100) },
		"statColor":   statColor,
		"lower":       strings.ToLower,
		"catPt":       catPt,
		"dict":        dict,
		"sprite":      spriteURL,
		"spriteShiny": spriteShinyURL,
		"typeGem":     func(t string) string { return "static/img/types/" + strings.ToLower(t) + ".svg" },
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

// catPt traduz a categoria do golpe para português (para o seletor de idioma).
func catPt(c string) string {
	switch c {
	case "Physical":
		return "Físico"
	case "Special":
		return "Especial"
	case "Status":
		return "Status"
	}
	return c
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
