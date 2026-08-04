// Package importer le o .jar do Cobblemon (species + lang), baixa as stats de
// golpes do Pokemon Showdown e gera data/pokedex.json e data/moves.json.
package importer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nickspokedex/internal/model"
)

const (
	showdownMovesURL = "https://play.pokemonshowdown.com/data/moves.json"
	spriteBaseURL    = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png"
	shinyBaseURL     = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/shiny/%d.png"
	// Sprites de formas (regionais/mega/origin...) vem do Showdown, que os nomeia
	// por forma (ex.: vulpix-alola.png).
	formSpriteURL = "https://play.pokemonshowdown.com/sprites/gen5/%s.png"
	formShinyURL  = "https://play.pokemonshowdown.com/sprites/gen5-shiny/%s.png"
)

// Options controla a importacao.
type Options struct {
	JarPath  string // caminho do .jar do Cobblemon
	PixelJar string // caminho do .jar do Pixelmon (opcional; so para spawns)
	OutDir   string // pasta de saida (data/)
	Offline  bool   // se true, nao baixa stats de move do Showdown
	Sprites  bool   // se true, baixa os sprites pixelart do PokeAPI
}

// ---------- estruturas cruas (formato Cobblemon) ----------

type rawSpecies struct {
	Implemented           bool           `json:"implemented"`
	NationalPokedexNumber int            `json:"nationalPokedexNumber"`
	Name                  string         `json:"name"`
	PrimaryType           string         `json:"primaryType"`
	SecondaryType         string         `json:"secondaryType"`
	Abilities             []string       `json:"abilities"`
	EggGroups             []string       `json:"eggGroups"`
	BaseStats             map[string]int `json:"baseStats"`
	Moves                 []string       `json:"moves"`
	Labels                []string       `json:"labels"`
	Aspects               []string       `json:"aspects"`
	Pokedex               []string       `json:"pokedex"`
	Evolutions            []rawEvo       `json:"evolutions"`
	Forms                 []rawForm      `json:"forms"`
}

// rawForm e uma variante da especie (regional, mega, origin...). Campos ausentes
// sao herdados da especie base.
type rawForm struct {
	Name          string         `json:"name"`
	PrimaryType   string         `json:"primaryType"`
	SecondaryType string         `json:"secondaryType"`
	Aspects       []string       `json:"aspects"`
	Labels        []string       `json:"labels"`
	Abilities     []string       `json:"abilities"`
	EggGroups     []string       `json:"eggGroups"`
	BaseStats     map[string]int `json:"baseStats"`
	Moves         []string       `json:"moves"`
	Pokedex       []string       `json:"pokedex"`
	Evolutions    []rawEvo       `json:"evolutions"`
	BattleOnly    bool           `json:"battleOnly"`
}

type rawEvo struct {
	Variant         string           `json:"variant"`
	Result          string           `json:"result"`
	RequiredContext json.RawMessage  `json:"requiredContext"`
	Requirements    []map[string]any `json:"requirements"`
}

// ---------- estrutura crua (Pokemon Showdown) ----------

type sdMove struct {
	Num       int    `json:"num"`
	Accuracy  any    `json:"accuracy"`
	BasePower int    `json:"basePower"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	PP        int    `json:"pp"`
	Type      string `json:"type"`
	ShortDesc string `json:"shortDesc"`
	Desc      string `json:"desc"`
}

// Run executa a importacao completa.
func Run(opts Options) error {
	if opts.OutDir == "" {
		opts.OutDir = "data"
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	if opts.JarPath == "" {
		opts.JarPath = findCobblemonJar()
		if opts.JarPath == "" {
			return fmt.Errorf("jar do Cobblemon nao encontrado; passe -jar CAMINHO ou defina a variavel COBBLEMON_JAR")
		}
		fmt.Printf("jar detectado: %s\n", opts.JarPath)
	}

	zr, err := zip.OpenReader(opts.JarPath)
	if err != nil {
		return fmt.Errorf("abrindo jar %q: %w", opts.JarPath, err)
	}
	defer zr.Close()

	lang, err := readLang(zr)
	if err != nil {
		return fmt.Errorf("lendo lang: %w", err)
	}
	fmt.Printf("lang: %d chaves\n", len(lang))

	pokes, referenced, err := readSpecies(zr, lang)
	if err != nil {
		return fmt.Errorf("lendo species: %w", err)
	}
	fmt.Printf("species: %d pokemon, %d moves referenciados\n", len(pokes), len(referenced))

	// Onde encontrar: spawns do Cobblemon (mesmo jar) + Pixelmon (jar opcional).
	bySlug := make(map[string]*model.Pokemon, len(pokes))
	for i := range pokes {
		bySlug[pokes[i].Slug] = &pokes[i]
	}
	attachCobblemonSpawns(zr, bySlug)
	pixelJar := opts.PixelJar
	if pixelJar == "" {
		pixelJar = findPixelmonJar()
	}
	if pixelJar != "" {
		fmt.Printf("jar Pixelmon: %s\n", pixelJar)
	}
	attachPixelmonSpawns(pixelJar, bySlug)
	for i := range pokes {
		pokes[i].Encounters = dedupeEncounters(pokes[i].Encounters)
	}

	var sd map[string]sdMove
	if !opts.Offline {
		sd, err = fetchShowdownMoves()
		if err != nil {
			fmt.Printf("aviso: falha ao baixar stats de moves (%v). Seguindo sem stats.\n", err)
		} else {
			fmt.Printf("showdown: %d moves com stats\n", len(sd))
		}
	}

	moves := buildMoves(referenced, sd, lang)

	if err := writeJSON(filepath.Join(opts.OutDir, "pokedex.json"), pokes); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(opts.OutDir, "moves.json"), moves); err != nil {
		return err
	}
	fmt.Printf("gerado: %s/pokedex.json e %s/moves.json\n", opts.OutDir, opts.OutDir)

	// Itens do Pixelmon (nome + o que faz + drops). Só gera se o jar existir;
	// senão mantém o items.json versionado. Fica embutido em data/.
	if items := buildPixelmonItems(pixelJar, opts.OutDir); len(items) > 0 {
		if err := writeJSON(filepath.Join(opts.OutDir, "items.json"), items); err != nil {
			return err
		}
		fmt.Printf("gerado: %s/items.json (%d itens)\n", opts.OutDir, len(items))
	} else {
		fmt.Println("itens: jar do Pixelmon ausente — mantendo items.json versionado")
	}

	if err := extractTypeGems(zr, opts.OutDir); err != nil {
		fmt.Printf("aviso: falha ao extrair type gems (%v)\n", err)
	}

	if opts.Sprites {
		if err := downloadSprites(pokes, opts.OutDir); err != nil {
			fmt.Printf("aviso: falha ao baixar sprites (%v)\n", err)
		}
	}
	return nil
}

// downloadSprites baixa os sprites pixelart (normal e shiny) de cada Pokemon
// para OutDir/sprites/{dex}.png e OutDir/sprites/shiny/{dex}.png. E idempotente:
// pula os que ja existem, entao rodar de novo e rapido.
func downloadSprites(pokes []model.Pokemon, outDir string) error {
	dir := filepath.Join(outDir, "sprites")
	shinyDir := filepath.Join(dir, "shiny")
	for _, d := range []string{dir, shinyDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// Sprites de forma ficam em sprites/forms/ (normal) e sprites/forms/shiny/.
	formDir := filepath.Join(dir, "forms")
	formShinyDir := filepath.Join(formDir, "shiny")
	for _, d := range []string{formDir, formShinyDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	type task struct{ url, path string }
	var tasks []task
	seen := map[int]bool{}
	for _, p := range pokes {
		if p.SpriteSlug != "" {
			// Forma: sprite por nome no Showdown.
			if norm := filepath.Join(formDir, p.SpriteSlug+".png"); !fileExists(norm) {
				tasks = append(tasks, task{fmt.Sprintf(formSpriteURL, p.SpriteSlug), norm})
			}
			if shiny := filepath.Join(formShinyDir, p.SpriteSlug+".png"); !fileExists(shiny) {
				tasks = append(tasks, task{fmt.Sprintf(formShinyURL, p.SpriteSlug), shiny})
			}
			continue
		}
		if p.Dex <= 0 || seen[p.Dex] {
			continue
		}
		seen[p.Dex] = true
		if norm := filepath.Join(dir, fmt.Sprintf("%d.png", p.Dex)); !fileExists(norm) {
			tasks = append(tasks, task{fmt.Sprintf(spriteBaseURL, p.Dex), norm})
		}
		if shiny := filepath.Join(shinyDir, fmt.Sprintf("%d.png", p.Dex)); !fileExists(shiny) {
			tasks = append(tasks, task{fmt.Sprintf(shinyBaseURL, p.Dex), shiny})
		}
	}
	if len(tasks) == 0 {
		fmt.Println("sprites: ja estao em dia")
		return nil
	}

	fmt.Printf("sprites: baixando %d (normais + shiny)...\n", len(tasks))
	client := &http.Client{Timeout: 20 * time.Second}
	var ok, fail int64
	sem := make(chan struct{}, 16) // limita concorrencia
	var wg sync.WaitGroup

	for _, t := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(t task) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fetchToFile(client, t.url, t.path); err != nil {
				atomic.AddInt64(&fail, 1)
				return
			}
			atomic.AddInt64(&ok, 1)
		}(t)
	}
	wg.Wait()
	fmt.Printf("sprites: %d baixados, %d falharam\n", ok, fail)
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// extractTypeGems copia os PNGs de gem de tipo do jar (16x16, na cor de cada
// tipo) para OutDir/types/{tipo}.png.
func extractTypeGems(zr *zip.ReadCloser, outDir string) error {
	const prefix = "assets/cobblemon/textures/item/type_gem/"
	dir := filepath.Join(outDir, "types")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	n := 0
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, "_gem.png") {
			continue
		}
		typ := strings.TrimSuffix(strings.TrimPrefix(f.Name, prefix), "_gem.png")
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, typ+".png"), data, 0o644); err != nil {
			return err
		}
		n++
	}
	fmt.Printf("type gems: %d extraidos\n", n)
	return nil
}

func fetchToFile(client *http.Client, url, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()
	return os.Rename(tmp, dest)
}

// findCobblemonJar procura o .jar do Cobblemon nos locais comuns de instalacao
// (PrismLauncher, .minecraft, CurseForge, Modrinth). Retorna o mais recente.
func findCobblemonJar() string {
	var patterns []string
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		patterns = append(patterns,
			filepath.Join(appdata, "PrismLauncher", "instances", "*", "minecraft", "mods", "*obblemon*.jar"),
			filepath.Join(appdata, ".minecraft", "mods", "*obblemon*.jar"),
			filepath.Join(appdata, "com.modrinth.theseus", "profiles", "*", "mods", "*obblemon*.jar"),
		)
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		patterns = append(patterns,
			filepath.Join(home, "curseforge", "minecraft", "Instances", "*", "mods", "*obblemon*.jar"),
		)
	}
	if home := os.Getenv("HOME"); home != "" {
		patterns = append(patterns,
			filepath.Join(home, ".minecraft", "mods", "*obblemon*.jar"),
			filepath.Join(home, ".local", "share", "PrismLauncher", "instances", "*", "minecraft", "mods", "*obblemon*.jar"),
		)
	}

	return newestMatch(patterns)
}

// findPixelmonJar procura o .jar do Pixelmon nos mesmos locais comuns.
func findPixelmonJar() string {
	var patterns []string
	add := func(dirs ...string) {
		for _, d := range dirs {
			patterns = append(patterns, filepath.Join(d, "*ixelmon*.jar"))
		}
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		add(
			filepath.Join(appdata, "PrismLauncher", "instances", "*", "minecraft", "mods"),
			filepath.Join(appdata, ".minecraft", "mods"),
			filepath.Join(appdata, "com.modrinth.theseus", "profiles", "*", "mods"),
		)
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		add(filepath.Join(home, "curseforge", "minecraft", "Instances", "*", "mods"))
	}
	if home := os.Getenv("HOME"); home != "" {
		add(
			filepath.Join(home, ".minecraft", "mods"),
			filepath.Join(home, ".local", "share", "PrismLauncher", "instances", "*", "minecraft", "mods"),
		)
	}
	return newestMatch(patterns)
}

// newestMatch retorna o arquivo mais recente entre os que casam com os padroes.
func newestMatch(patterns []string) string {
	var newest string
	var newestMod time.Time
	for _, pat := range patterns {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil && fi.ModTime().After(newestMod) {
				newest, newestMod = m, fi.ModTime()
			}
		}
	}
	return newest
}

// ---------- lang ----------

func readLang(zr *zip.ReadCloser) (map[string]string, error) {
	for _, f := range zr.File {
		if f.Name == "assets/cobblemon/lang/en_us.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			var m map[string]string
			if err := json.NewDecoder(rc).Decode(&m); err != nil {
				return nil, err
			}
			return m, nil
		}
	}
	return map[string]string{}, nil
}

// ---------- species ----------

func readSpecies(zr *zip.ReadCloser, lang map[string]string) ([]model.Pokemon, map[string]bool, error) {
	var pokes []model.Pokemon
	referenced := map[string]bool{}

	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "data/cobblemon/species/") || !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, nil, err
		}
		var rs rawSpecies
		if err := json.Unmarshal(data, &rs); err != nil {
			// pula arquivos que nao sao species (ex.: indices)
			continue
		}
		if rs.Name == "" {
			continue
		}

		slug := strings.TrimSuffix(path.Base(f.Name), ".json")
		gen := generationLabel(f.Name)
		base := buildPokemon(rs, slug, gen, lang, referenced)

		forms := meaningfulForms(rs)
		if len(forms) > 0 {
			// Rotula a base como uma "forma" tambem, para o seletor de formas.
			base.Form = baseFormLabel(rs)
			base.BaseSlug = slug
		}
		pokes = append(pokes, base)

		for _, rf := range forms {
			pokes = append(pokes, buildForm(base, rs, rf, slug, lang, referenced))
		}
	}

	sort.Slice(pokes, func(i, j int) bool {
		if pokes[i].Dex != pokes[j].Dex {
			return pokes[i].Dex < pokes[j].Dex
		}
		return pokes[i].Slug < pokes[j].Slug
	})
	return pokes, referenced, nil
}

func buildPokemon(rs rawSpecies, slug, gen string, lang map[string]string, referenced map[string]bool) model.Pokemon {
	p := model.Pokemon{
		Dex:         rs.NationalPokedexNumber,
		Name:        rs.Name,
		Slug:        slug,
		Types:       parseTypes(rs.PrimaryType, rs.SecondaryType),
		BaseStats:   parseStats(rs.BaseStats),
		Abilities:   parseAbilities(rs.Abilities, lang),
		EggGroups:   rs.EggGroups,
		Description: parseDescription(rs.Pokedex, lang),
		Generation:  gen,
		Labels:      rs.Labels,
		Implemented: rs.Implemented,
	}
	setMoves(&p, rs.Moves, referenced)
	p.Evolutions = parseEvolutions(rs.Evolutions)
	return p
}

// buildForm monta uma entrada de forma a partir da base, herdando o que a forma
// nao redefine (stats/moves/habilidades/tipos).
func buildForm(base model.Pokemon, rs rawSpecies, rf rawForm, speciesSlug string, lang map[string]string, referenced map[string]bool) model.Pokemon {
	aspect := aspectKey(rf)
	p := model.Pokemon{
		Dex:         base.Dex,
		Name:        base.Name + " (" + prettify(rf.Name) + ")",
		Slug:        speciesSlug + "-" + aspect,
		Implemented: base.Implemented,
		Form:        prettify(rf.Name),
		Aspect:      aspect,
		BaseSlug:    speciesSlug,
		SpriteSlug:  speciesSlug + "-" + showdownSuffix(rf.Name),
		BattleOnly:  rf.BattleOnly,
	}

	// Tipos: a forma quase sempre define os seus; senao herda.
	if rf.PrimaryType != "" {
		p.Types = parseTypes(rf.PrimaryType, rf.SecondaryType)
	} else {
		p.Types = base.Types
	}
	// Stats: se a forma nao traz, herda.
	if len(rf.BaseStats) > 0 {
		p.BaseStats = parseStats(rf.BaseStats)
	} else {
		p.BaseStats = base.BaseStats
	}
	// Habilidades.
	if len(rf.Abilities) > 0 {
		p.Abilities = parseAbilities(rf.Abilities, lang)
	} else {
		p.Abilities = base.Abilities
	}
	// Grupos de ovo.
	if len(rf.EggGroups) > 0 {
		p.EggGroups = rf.EggGroups
	} else {
		p.EggGroups = base.EggGroups
	}
	// Descricao.
	if d := parseDescription(rf.Pokedex, lang); d != "" {
		p.Description = d
	} else {
		p.Description = base.Description
	}
	// Geracao (a partir dos labels genN da forma).
	if g := genFromLabels(rf.Labels); g != "" {
		p.Generation = g
	} else {
		p.Generation = base.Generation
	}
	p.Labels = rf.Labels
	// Moves: se a forma traz seu proprio moveset, usa; senao herda o da base.
	if len(rf.Moves) > 0 {
		setMoves(&p, rf.Moves, referenced)
	} else {
		p.LevelMoves, p.TMMoves, p.EggMoves, p.TutorMoves = base.LevelMoves, base.TMMoves, base.EggMoves, base.TutorMoves
	}
	// Evolucoes proprias da forma (se a chave existir, mesmo vazia).
	p.Evolutions = parseEvolutions(rf.Evolutions)
	return p
}

func parseTypes(primary, secondary string) []string {
	types := []string{}
	if primary != "" {
		types = append(types, strings.ToLower(primary))
	}
	if secondary != "" {
		types = append(types, strings.ToLower(secondary))
	}
	return types
}

func parseStats(m map[string]int) model.Stats {
	return model.Stats{
		HP: m["hp"], Attack: m["attack"], Defence: m["defence"],
		SpAttack: m["special_attack"], SpDefence: m["special_defence"], Speed: m["speed"],
	}
}

func parseAbilities(abilities []string, lang map[string]string) []model.Ability {
	var out []model.Ability
	for _, a := range abilities {
		hidden := false
		id := a
		if strings.HasPrefix(a, "h:") {
			hidden = true
			id = strings.TrimPrefix(a, "h:")
		}
		out = append(out, model.Ability{Name: abilityName(id, lang), Hidden: hidden})
	}
	return out
}

func parseDescription(keys []string, lang map[string]string) string {
	var lines []string
	for _, key := range keys {
		if v, ok := lang[key]; ok {
			lines = append(lines, v)
		}
	}
	return strings.Join(lines, " ")
}

func parseEvolutions(evos []rawEvo) []model.Evolution {
	var out []model.Evolution
	for _, ev := range evos {
		to, cond := summarizeEvo(ev)
		if to == "" {
			continue
		}
		out = append(out, model.Evolution{To: to, Condition: cond})
	}
	return out
}

// setMoves preenche os slices de golpes de p a partir do array cru do Cobblemon.
func setMoves(p *model.Pokemon, moves []string, referenced map[string]bool) {
	p.LevelMoves, p.TMMoves, p.EggMoves, p.TutorMoves = nil, nil, nil, nil
	for _, m := range moves {
		method, id, level := parseMoveEntry(m)
		if id == "" {
			continue
		}
		switch method {
		case "level":
			p.LevelMoves = append(p.LevelMoves, model.LevelMove{Level: level, Move: id})
			referenced[id] = true
		case "tm":
			p.TMMoves = append(p.TMMoves, id)
			referenced[id] = true
		case "egg":
			p.EggMoves = append(p.EggMoves, id)
			referenced[id] = true
		case "tutor":
			p.TutorMoves = append(p.TutorMoves, id)
			referenced[id] = true
			// legacy/special ignorados
		}
	}
	sort.SliceStable(p.LevelMoves, func(i, j int) bool {
		if p.LevelMoves[i].Level != p.LevelMoves[j].Level {
			return p.LevelMoves[i].Level < p.LevelMoves[j].Level
		}
		return p.LevelMoves[i].Move < p.LevelMoves[j].Move
	})
	sort.Strings(p.TMMoves)
	sort.Strings(p.EggMoves)
	sort.Strings(p.TutorMoves)
}

// ---------- formas ----------

// significantAspects sao aspectos sempre mantidos como forma propria, mesmo que
// os stats/tipos coincidam com a base.
var significantAspects = map[string]bool{
	"alolan": true, "galarian": true, "hisuian": true, "paldean": true,
	"mega": true, "mega_x": true, "mega_y": true, "primal": true, "origin": true,
	"altered": true, "therian": true, "incarnate": true, "crowned": true,
	"eternamax": true, "dawn_wings": true, "dusk_mane": true, "ultra": true,
	"black": true, "white": true, "resolute": true, "pirouette": true,
	"zen": true, "noice": true, "school": true, "hangry": true, "gulping": true,
	"gorging": true, "ash": true, "starter": true, "sky": true, "unbound": true,
	"complete": true, "10_percent": true, "blade": true, "sunshine": true,
}

// meaningfulForms retorna as formas que valem uma entrada propria (regionais,
// megas, origins, e qualquer forma que mude tipo/stats/habilidade/evolucao),
// descartando cosmeticas (letras do Unown, padroes de Vivillon...) e Gmax.
func meaningfulForms(rs rawSpecies) []rawForm {
	baseTypes := parseTypes(rs.PrimaryType, rs.SecondaryType)
	var out []rawForm
	for _, rf := range rs.Forms {
		if rf.Name == "" || isGmax(rf) || isBias(rf) {
			continue
		}
		if formIsMeaningful(rs, rf, baseTypes) {
			out = append(out, rf)
		}
	}
	return out
}

// isBias descarta as formas "bias" do Cobblemon (variantes internas de breeding
// regional, ex.: "Cubone" com aspecto alolano) — nao sao formas de verdade.
func isBias(rf rawForm) bool {
	if strings.Contains(strings.ToLower(rf.Name), "bias") {
		return true
	}
	for _, a := range rf.Aspects {
		if strings.Contains(strings.ToLower(a), "bias") {
			return true
		}
	}
	return false
}

func isGmax(rf rawForm) bool {
	if strings.EqualFold(rf.Name, "gmax") {
		return true
	}
	for _, a := range rf.Aspects {
		if strings.EqualFold(a, "gmax") {
			return true
		}
	}
	return false
}

func formIsMeaningful(rs rawSpecies, rf rawForm, baseTypes []string) bool {
	for _, a := range rf.Aspects {
		if significantAspects[strings.ToLower(a)] {
			return true
		}
	}
	if rf.PrimaryType != "" && !sameStrings(parseTypes(rf.PrimaryType, rf.SecondaryType), baseTypes) {
		return true
	}
	if len(rf.BaseStats) > 0 && parseStats(rf.BaseStats) != parseStats(rs.BaseStats) {
		return true
	}
	if len(rf.Abilities) > 0 && !sameStrings(rf.Abilities, rs.Abilities) {
		return true
	}
	if len(rf.Evolutions) > 0 {
		return true
	}
	return false
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.ToLower(a[i]) != strings.ToLower(b[i]) {
			return false
		}
	}
	return true
}

// aspectKey deriva a chave interna do slug da forma (deve casar com os alvos de
// evolucao, que usam a palavra do aspecto, ex.: "alolan").
func aspectKey(rf rawForm) string {
	if len(rf.Aspects) > 0 {
		return strings.Join(lowerAll(rf.Aspects), "-")
	}
	return slugify(rf.Name)
}

// showdownSuffix deriva o sufixo do sprite no Showdown a partir do nome da forma.
// O Showdown concatena os tokens da forma num unico bloco, sem hifens/espacos:
// "Alola"->"alola", "Mega X"->"megax", "Paldea Combat"->"paldeacombat",
// "Dawn Wings"->"dawnwings", "Pa'u"->"pau".
func showdownSuffix(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	repl := strings.NewReplacer(" ", "", "-", "", "'", "", "%", "", ".", "")
	s = repl.Replace(s)
	switch s {
	case "bond": // Greninja: forma Battle Bond = Ash-Greninja no Showdown.
		return "ash"
	case "noiceface": // Eiscue.
		return "noice"
	case "10c":
		return "10"
	case "50c":
		return "complete"
	}
	return s
}

// baseFormLabel rotula a forma-base para o seletor de formas (ex.: "Kanto").
func baseFormLabel(rs rawSpecies) string {
	for _, l := range rs.Labels {
		if r := regionFromLabel(l); r != "" {
			return r
		}
	}
	return "Base"
}

func regionFromLabel(label string) string {
	m := map[string]string{
		"kantonian_form": "Kanto", "johtonian_form": "Johto", "hoennian_form": "Hoenn",
		"sinnohan_form": "Sinnoh", "unovan_form": "Unova", "kalosian_form": "Kalos",
		"alolan_form": "Alola", "galarian_form": "Galar", "hisuian_form": "Hisui",
		"paldean_form": "Paldea",
	}
	return m[strings.ToLower(label)]
}

func genFromLabels(labels []string) string {
	for _, l := range labels {
		if strings.HasPrefix(l, "gen") {
			if n := strings.TrimPrefix(l, "gen"); n != "" && n[0] >= '0' && n[0] <= '9' {
				return "Gen " + n
			}
		}
	}
	return ""
}

func lowerAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToLower(s)
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// parseMoveEntry separa "7:vinewhip" -> ("level","vinewhip",7) e
// "tm:solarbeam" -> ("tm","solarbeam",0).
func parseMoveEntry(entry string) (method, id string, level int) {
	i := strings.Index(entry, ":")
	if i < 0 {
		return "level", entry, 0
	}
	prefix := entry[:i]
	id = entry[i+1:]
	if n, err := strconv.Atoi(prefix); err == nil {
		return "level", id, n
	}
	return prefix, id, 0
}

// ---------- evolucoes ----------

func summarizeEvo(ev rawEvo) (to, cond string) {
	fields := strings.Fields(ev.Result)
	if len(fields) == 0 {
		return "", ""
	}
	// "persian alolan" -> slug "persian-alolan" (especie + aspecto da forma);
	// "charizard" -> "charizard". Casa com o slug gerado em buildForm.
	to = strings.Join(fields, "-")

	level := ""
	var extras []string
	for _, r := range ev.Requirements {
		variant, _ := r["variant"].(string)
		switch variant {
		case "level":
			level = "Level " + numStr(r["minLevel"])
		case "friendship":
			extras = append(extras, "amizade "+numStr(r["amount"])+"+")
		case "time_range":
			extras = append(extras, timeRange(str(r["range"])))
		case "has_move":
			extras = append(extras, "sabendo "+prettify(str(r["move"])))
		case "has_move_type":
			extras = append(extras, "sabendo golpe "+str(r["type"]))
		case "biome":
			// muito ruidoso (formas regionais) — ignorado
		default:
			if variant != "" {
				extras = append(extras, prettify(variant))
			}
		}
	}

	var base string
	switch ev.Variant {
	case "trade":
		base = "Troca"
	case "item_interact":
		base = "Usar " + prettifyItem(rawStr(ev.RequiredContext))
	case "level_up":
		if level != "" {
			base = level
		} else {
			base = "Subir de level"
		}
	default:
		if level != "" {
			base = level
		} else {
			base = prettify(ev.Variant)
		}
	}

	parts := []string{base}
	parts = append(parts, extras...)
	return to, strings.Join(parts, ", ")
}

func timeRange(r string) string {
	switch r {
	case "day":
		return "de dia"
	case "night":
		return "de noite"
	default:
		return r
	}
}

// ---------- moves (Showdown) ----------

func fetchShowdownMoves() (map[string]sdMove, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(showdownMovesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var m map[string]sdMove
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func buildMoves(referenced map[string]bool, sd map[string]sdMove, lang map[string]string) map[string]model.Move {
	out := map[string]model.Move{}
	for slug := range referenced {
		mv := model.Move{Slug: slug, Name: moveName(slug, lang)}
		if s, ok := sd[slug]; ok {
			mv.Type = strings.ToLower(s.Type)
			mv.Category = s.Category
			mv.Power = s.BasePower
			mv.Accuracy = accuracyInt(s.Accuracy)
			mv.PP = s.PP
			mv.Desc = firstNonEmpty(s.ShortDesc, s.Desc)
		}
		if mv.Desc == "" {
			mv.Desc = firstNonEmpty(lang["cobblemon.move."+slug+".desc"], "")
		}
		out[slug] = mv
	}
	return out
}

func accuracyInt(a any) int {
	switch v := a.(type) {
	case float64:
		return int(v)
	case bool:
		return 0 // "true" no Showdown = nunca erra
	default:
		return 0
	}
}

// ---------- helpers de nome/lang ----------

func moveName(slug string, lang map[string]string) string {
	if v, ok := lang["cobblemon.move."+slug]; ok {
		return v
	}
	return prettify(slug)
}

func abilityName(id string, lang map[string]string) string {
	if v, ok := lang["cobblemon.ability."+id]; ok {
		return v
	}
	return prettify(id)
}

func generationLabel(name string) string {
	// .../species/generation1/bulbasaur.json
	parts := strings.Split(name, "/")
	for _, p := range parts {
		if strings.HasPrefix(p, "generation") {
			n := strings.TrimPrefix(p, "generation")
			return "Gen " + n
		}
	}
	return ""
}

// prettify transforma "thunder_stone"/"vinewhip" em texto legivel.
func prettify(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func prettifyItem(s string) string {
	if i := strings.Index(s, ":"); i >= 0 {
		s = s[i+1:]
	}
	return prettify(s)
}

func numStr(v any) string {
	switch n := v.(type) {
	case float64:
		return strconv.Itoa(int(n))
	case int:
		return strconv.Itoa(n)
	case string:
		return n
	default:
		return ""
	}
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func rawStr(r json.RawMessage) string {
	if len(r) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(r, &s); err == nil {
		return s
	}
	return ""
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", " ")
	return enc.Encode(v)
}
