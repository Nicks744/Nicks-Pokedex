// Spawns: enriquece cada Pokemon com "onde encontrar" a partir dos dados de
// spawn do Cobblemon (data/cobblemon/spawn_pool_world) e, se um jar do Pixelmon
// for fornecido, do Pixelmon (data/pixelmon/spawning). Os biomas sao traduzidos
// para PT-BR; a raridade vira um rotulo comum/incomum/raro/ultra-raro.
package importer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"nickspokedex/internal/model"
)

// ---------- Cobblemon ----------

type cobSpawnFile struct {
	Enabled bool       `json:"enabled"`
	Spawns  []cobSpawn `json:"spawns"`
}

type cobSpawn struct {
	Pokemon   string   `json:"pokemon"`
	Presets   []string `json:"presets"`
	Bucket    string   `json:"bucket"`
	Level     string   `json:"level"`
	Condition cobCond  `json:"condition"`
}

type cobCond struct {
	Biomes             []string `json:"biomes"`
	TimeRange          string   `json:"timeRange"`
	IsRaining          *bool    `json:"isRaining"`
	IsThundering       *bool    `json:"isThundering"`
	CanSeeSky          *bool    `json:"canSeeSky"`
	Structures         []string `json:"structures"`
	NeededNearbyBlocks []string `json:"neededNearbyBlocks"`
	MinY               *int     `json:"minY"`
	MaxY               *int     `json:"maxY"`
}

// attachCobblemonSpawns le os spawns do jar do Cobblemon e anexa os encontros.
func attachCobblemonSpawns(zr *zip.ReadCloser, bySlug map[string]*model.Pokemon) {
	const prefix = "data/cobblemon/spawn_pool_world/"
	n := 0
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		var sf cobSpawnFile
		if err := json.Unmarshal(data, &sf); err != nil || !sf.Enabled {
			continue
		}
		for _, sp := range sf.Spawns {
			slug := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(sp.Pokemon)), " ", "-")
			p := bySlug[slug]
			if p == nil {
				continue
			}
			p.Encounters = append(p.Encounters, cobEncounter(sp))
			n++
		}
	}
	fmt.Printf("spawns Cobblemon: %d encontros\n", n)
}

func cobEncounter(sp cobSpawn) model.Encounter {
	e := model.Encounter{Source: "Cobblemon", Method: cobMethod(sp.Presets)}
	e.Rarity, e.RarityRank = cobRarity(sp.Bucket)
	e.Levels = levelRange(sp.Level)

	// Biomas (traduzidos, dedup, cap).
	e.Biomes = translateBiomes(sp.Condition.Biomes, cobBiome, 8)

	// Condicoes (em inglês, como no jogo).
	var c []string
	if d := cobDimension(sp.Condition.Biomes); d != "" {
		c = append(c, d)
	}
	if t := timeLabel(sp.Condition.TimeRange); t != "" {
		c = append(c, t)
	}
	if sp.Condition.IsThundering != nil && *sp.Condition.IsThundering {
		c = append(c, "Thunderstorm")
	} else if sp.Condition.IsRaining != nil && *sp.Condition.IsRaining {
		c = append(c, "Raining")
	}
	if sp.Condition.CanSeeSky != nil {
		if *sp.Condition.CanSeeSky {
			c = append(c, "Open sky")
		} else {
			c = append(c, "Underground")
		}
	}
	if len(sp.Condition.Structures) > 0 {
		c = append(c, "Structure: "+prettify(lastPath(sp.Condition.Structures[0])))
	}
	if len(sp.Condition.NeededNearbyBlocks) > 0 {
		c = append(c, "Near "+prettify(lastPath(sp.Condition.NeededNearbyBlocks[0])))
	}
	if sp.Condition.MaxY != nil && *sp.Condition.MaxY <= 0 {
		c = append(c, "Deep caves")
	}
	e.Conditions = c
	return e
}

func cobMethod(presets []string) string {
	has := func(s string) bool {
		for _, p := range presets {
			if p == s {
				return true
			}
		}
		return false
	}
	switch {
	case has("fishing"):
		return "Fishing"
	case has("water"):
		return "In water"
	case has("treetop"):
		return "Treetops"
	case has("urban"):
		return "Villages / urban"
	case has("mansion"), has("trail_ruins"), has("jungle_pyramid"), has("derelict"), has("ruined_portal"):
		return "In structure"
	default:
		return "Grass (land)"
	}
}

func cobRarity(bucket string) (string, int) {
	switch bucket {
	case "common":
		return "Common", 1
	case "uncommon":
		return "Uncommon", 2
	case "rare":
		return "Rare", 3
	case "ultra-rare":
		return "Ultra-rare", 4
	default:
		return "Common", 1
	}
}

func cobDimension(biomes []string) string {
	for _, b := range biomes {
		low := strings.ToLower(b)
		if strings.Contains(low, "nether/") {
			return "Nether"
		}
		if strings.HasSuffix(low, "is_end") || strings.Contains(low, ":end") {
			return "The End"
		}
	}
	return ""
}

// cobBiome devolve a tag de bioma do Cobblemon em inglês (sem tradução), apenas
// limpando o namespace/prefixo. Ex.: "#cobblemon:is_forest" -> "Forest".
func cobBiome(tag string) string {
	key := biomeKey(tag) // ex.: "is_forest", "nether/is_forest"
	nether := strings.HasPrefix(key, "nether/")
	key = strings.TrimPrefix(key, "nether/")
	key = strings.TrimPrefix(key, "is_")
	name := prettify(key)
	if nether {
		name += " (Nether)"
	}
	return name
}

// ---------- Pixelmon ----------

type pxSet struct {
	SpawnInfos []pxInfo `json:"spawnInfos"`
}

type pxInfo struct {
	Spec                string   `json:"spec"`
	StringLocationTypes []string `json:"stringLocationTypes"`
	MinLevel            int      `json:"minLevel"`
	MaxLevel            int      `json:"maxLevel"`
	TypeID              string   `json:"typeID"`
	Rarity              float64  `json:"rarity"`
	Condition           pxCond   `json:"condition"`
}

type pxCond struct {
	Times    []string `json:"times"`
	Biomes   []string `json:"biomes"`
	Weathers []string `json:"weathers"`
}

// attachPixelmonSpawns abre o jar do Pixelmon e anexa os encontros. Silencioso se
// jarPath vazio ou ilegivel (Pixelmon e opcional).
func attachPixelmonSpawns(jarPath string, bySlug map[string]*model.Pokemon) {
	if jarPath == "" {
		return
	}
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		fmt.Printf("aviso: nao abriu o jar do Pixelmon (%v); seguindo sem spawns do Pixelmon\n", err)
		return
	}
	defer zr.Close()

	const prefix = "data/pixelmon/spawning/"
	n := 0
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".set.json") {
			continue
		}
		method := pxMethod(strings.SplitN(strings.TrimPrefix(f.Name, prefix), "/", 2)[0])
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		var set pxSet
		if err := json.Unmarshal(data, &set); err != nil {
			continue
		}
		for _, in := range set.SpawnInfos {
			if in.TypeID != "" && in.TypeID != "pokemon" {
				continue
			}
			slug := pxSlug(in.Spec, bySlug)
			if slug == "" {
				continue
			}
			bySlug[slug].Encounters = append(bySlug[slug].Encounters, pxEncounter(in, method))
			n++
		}
	}
	fmt.Printf("spawns Pixelmon: %d encontros\n", n)
}

// pxSlug resolve o spec ("species:Bulbasaur form:alolan") para um slug existente.
func pxSlug(spec string, bySlug map[string]*model.Pokemon) string {
	var species, form string
	for _, tok := range strings.Fields(spec) {
		k, v, ok := strings.Cut(tok, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "species", "pokemon":
			species = strings.ToLower(v)
		case "form":
			form = strings.ToLower(v)
		}
	}
	if species == "" {
		return ""
	}
	if form != "" {
		if _, ok := bySlug[species+"-"+form]; ok {
			return species + "-" + form
		}
	}
	if _, ok := bySlug[species]; ok {
		return species
	}
	return ""
}

func pxEncounter(in pxInfo, method string) model.Encounter {
	e := model.Encounter{Source: "Pixelmon", Method: method}
	e.Rarity, e.RarityRank = pxRarity(in.Rarity, method)
	if in.MinLevel != 0 || in.MaxLevel != 0 {
		e.Levels = fmt.Sprintf("%d–%d", in.MinLevel, in.MaxLevel)
	}
	e.Biomes = translateBiomes(in.Condition.Biomes, pxBiome, 8)

	var c []string
	if t := pxTimes(in.Condition.Times); t != "" {
		c = append(c, t)
	}
	for _, w := range in.Condition.Weathers {
		switch strings.ToUpper(w) {
		case "RAIN":
			c = append(c, "Raining")
		case "STORM", "THUNDER":
			c = append(c, "Thunderstorm")
		}
	}
	if hasStr(in.StringLocationTypes, "Water") && method != "Fishing" {
		c = append(c, "In water")
	}
	if hasStr(in.StringLocationTypes, "Air") {
		c = append(c, "In air")
	}
	e.Conditions = c
	return e
}

func pxMethod(folder string) string {
	m := map[string]string{
		"standard": "Grass", "grass": "Grass", "fishing": "Fishing",
		"curry": "Curry (camp)", "sweetscent": "Sweet Scent", "headbutt": "Headbutt (trees)",
		"rocksmash": "Rock Smash", "caverock": "Cave rocks", "forage": "Foraging",
		"legendaries": "Legendary", "megas": "Mega", "npcs": "NPC",
	}
	if v, ok := m[folder]; ok {
		return v
	}
	return prettify(folder)
}

func pxRarity(r float64, method string) (string, int) {
	if method == "Legendary" {
		return "Legendary", 4
	}
	switch {
	case r >= 40:
		return "Common", 1
	case r >= 15:
		return "Uncommon", 2
	case r >= 3:
		return "Rare", 3
	default:
		return "Ultra-rare", 4
	}
}

func pxTimes(times []string) string {
	if len(times) == 0 {
		return ""
	}
	set := map[string]bool{}
	for _, t := range times {
		switch strings.ToUpper(t) {
		case "MORNING", "DAY", "AFTERNOON":
			set["dia"] = true
		case "NIGHT", "MIDNIGHT":
			set["noite"] = true
		case "DAWN":
			set["amanhecer"] = true
		case "DUSK":
			set["anoitecer"] = true
		}
	}
	if len(set) >= 3 { // cobre praticamente o dia todo
		return ""
	}
	var out []string
	if set["dia"] {
		out = append(out, "Day")
	}
	if set["noite"] {
		out = append(out, "Night")
	}
	if set["amanhecer"] {
		out = append(out, "Dawn")
	}
	if set["anoitecer"] {
		out = append(out, "Dusk")
	}
	return strings.Join(out, " / ")
}

// pxBiome devolve a tag/bioma do Pixelmon em inglês (sem tradução), usando só a
// parte final da tag. Ex.: "#pixelmon:spawning/forests" -> "Forests".
func pxBiome(tag string) string {
	if !strings.HasPrefix(tag, "#") {
		// bioma especifico (namespace:bioma) — usa so a parte final.
		return prettify(lastPath(tag))
	}
	return prettify(lastPath(biomeKey(tag))) // "spawning/forests" -> "Forests"
}

// ---------- helpers comuns ----------

// translateBiomes aplica tr a cada tag, deduplica preservando ordem e limita a max.
func translateBiomes(tags []string, tr func(string) string, max int) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tags {
		name := tr(t)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
		if len(out) >= max {
			break
		}
	}
	return out
}

// biomeKey extrai a parte apos "namespace:" de uma tag (#ns:algo -> "algo").
func biomeKey(tag string) string {
	tag = strings.TrimPrefix(tag, "#")
	if i := strings.Index(tag, ":"); i >= 0 {
		return tag[i+1:]
	}
	return tag
}

func lastPath(s string) string {
	s = strings.TrimPrefix(s, "#")
	if i := strings.LastIndex(s, ":"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func timeLabel(tr string) string {
	switch strings.ToLower(tr) {
	case "day":
		return "Day"
	case "night":
		return "Night"
	case "twilight":
		return "Dawn / dusk"
	case "dawn":
		return "Dawn"
	case "dusk":
		return "Dusk"
	case "noon":
		return "Noon"
	case "midnight":
		return "Midnight"
	default:
		return ""
	}
}

func levelRange(l string) string {
	l = strings.TrimSpace(l)
	return strings.ReplaceAll(l, "-", "–")
}

func hasStr(ss []string, want string) bool {
	for _, s := range ss {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

// dedupeEncounters funde encontros que so diferem no bioma (mesma fonte/metodo/
// raridade/level/condicoes) unindo os biomas, e ordena por fonte e raridade.
func dedupeEncounters(es []model.Encounter) []model.Encounter {
	idx := map[string]int{} // assinatura sem biomas -> posicao em out
	var out []model.Encounter
	for _, e := range es {
		sig := e.Source + "|" + e.Method + "|" + e.Rarity + "|" + e.Levels + "|" + strings.Join(e.Conditions, ",")
		if pos, ok := idx[sig]; ok {
			out[pos].Biomes = unionCap(out[pos].Biomes, e.Biomes, 10)
			continue
		}
		idx[sig] = len(out)
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source // Cobblemon antes de Pixelmon
		}
		if out[i].RarityRank != out[j].RarityRank {
			return out[i].RarityRank < out[j].RarityRank
		}
		return out[i].Method < out[j].Method
	})
	return out
}

func unionCap(a, b []string, max int) []string {
	seen := map[string]bool{}
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		if len(a) >= max {
			break
		}
		if !seen[s] {
			seen[s] = true
			a = append(a, s)
		}
	}
	return a
}
