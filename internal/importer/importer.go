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
)

// Options controla a importacao.
type Options struct {
	JarPath string // caminho do .jar do Cobblemon
	OutDir  string // pasta de saida (data/)
	Offline bool   // se true, nao baixa stats de move do Showdown
	Sprites bool   // se true, baixa os sprites pixelart do PokeAPI
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
	Pokedex               []string       `json:"pokedex"`
	Evolutions            []rawEvo       `json:"evolutions"`
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

	type task struct{ url, path string }
	var tasks []task
	seen := map[int]bool{}
	for _, p := range pokes {
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
		p := buildPokemon(rs, slug, gen, lang, referenced)
		pokes = append(pokes, p)
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
	types := []string{}
	if rs.PrimaryType != "" {
		types = append(types, strings.ToLower(rs.PrimaryType))
	}
	if rs.SecondaryType != "" {
		types = append(types, strings.ToLower(rs.SecondaryType))
	}

	p := model.Pokemon{
		Dex:         rs.NationalPokedexNumber,
		Name:        rs.Name,
		Slug:        slug,
		Types:       types,
		BaseStats:   model.Stats{HP: rs.BaseStats["hp"], Attack: rs.BaseStats["attack"], Defence: rs.BaseStats["defence"], SpAttack: rs.BaseStats["special_attack"], SpDefence: rs.BaseStats["special_defence"], Speed: rs.BaseStats["speed"]},
		EggGroups:   rs.EggGroups,
		Generation:  gen,
		Labels:      rs.Labels,
		Implemented: rs.Implemented,
	}

	// Abilities
	for _, a := range rs.Abilities {
		hidden := false
		id := a
		if strings.HasPrefix(a, "h:") {
			hidden = true
			id = strings.TrimPrefix(a, "h:")
		}
		p.Abilities = append(p.Abilities, model.Ability{Name: abilityName(id, lang), Hidden: hidden})
	}

	// Descricao
	var descLines []string
	for _, key := range rs.Pokedex {
		if v, ok := lang[key]; ok {
			descLines = append(descLines, v)
		}
	}
	p.Description = strings.Join(descLines, " ")

	// Moves
	for _, m := range rs.Moves {
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
			// legacy/special/form ignorados no v1
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

	// Evolucoes
	for _, ev := range rs.Evolutions {
		to, cond := summarizeEvo(ev)
		if to == "" {
			continue
		}
		p.Evolutions = append(p.Evolutions, model.Evolution{To: to, Condition: cond})
	}

	return p
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
	to = fields[0]

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
	if len(fields) > 1 {
		parts = append(parts, "("+strings.Join(fields[1:], " ")+")")
	}
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
