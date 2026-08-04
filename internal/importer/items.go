// Itens do Pixelmon: gera data/items.json com o nome e a descricao (tooltip do
// mod) de cada item relevante, sua categoria, e — quando existir — de quais
// Pokemon ele dropa e com que chance (data/pixelmon/drops/pokedrops.json).
package importer

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"nickspokedex/internal/model"
)

// RunItems gera apenas data/items.json a partir do jar do Pixelmon (útil para
// atualizar os itens sem precisar do jar do Cobblemon / reimportar a pokedex).
func RunItems(pixelJar, outDir string) error {
	if outDir == "" {
		outDir = "data"
	}
	if pixelJar == "" {
		pixelJar = findPixelmonJar()
	}
	if pixelJar == "" {
		return fmt.Errorf("jar do Pixelmon nao encontrado; passe -pixeljar CAMINHO ou defina PIXELMON_JAR")
	}
	fmt.Printf("jar Pixelmon: %s\n", pixelJar)
	items := buildPixelmonItems(pixelJar, outDir)
	if len(items) == 0 {
		return fmt.Errorf("nenhum item lido do jar %q", pixelJar)
	}
	if err := writeJSON(filepath.Join(outDir, "items.json"), items); err != nil {
		return err
	}
	fmt.Printf("gerado: %s/items.json (%d itens)\n", outDir, len(items))
	return nil
}

// pxDropFile: pokedrops.json é um array de { pokemon, items:[{item,min,max,chance}] }.
type pxDropEntry struct {
	Pokemon string       `json:"pokemon"`
	Items   []pxDropItem `json:"items"`
}
type pxDropItem struct {
	Item   json.RawMessage `json:"item"` // string "pixelmon:x" OU objeto {"id":"pixelmon:x",...}
	Min    int             `json:"min"`
	Max    int             `json:"max"`
	Chance float64         `json:"chance"`
}

// itemID extrai o id do item, aceitando os dois formatos do pokedrops.json.
func (d pxDropItem) itemID() string {
	var s string
	if json.Unmarshal(d.Item, &s) == nil {
		return s
	}
	var obj struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(d.Item, &obj) == nil {
		return obj.ID
	}
	return ""
}

// buildPixelmonItems abre o jar do Pixelmon e monta o catalogo de itens, extraindo
// tambem a textura de cada item para outDir/itemtex/{id}.png. Retorna nil se o jar
// for vazio/ilegivel (Pixelmon é opcional).
func buildPixelmonItems(jarPath, outDir string) []model.Item {
	if jarPath == "" {
		return nil
	}
	zr, err := zip.OpenReader(jarPath)
	if err != nil {
		return nil
	}
	defer zr.Close()

	lang := readPixelmonLang(zr)
	if len(lang) == 0 {
		return nil
	}
	langPt := readPixelmonLangFile(zr, "assets/pixelmon/lang/pt_br.json") // nomes em PT (item.{id})
	drops := readPixelmonDrops(zr) // id do item (sem "pixelmon:") -> fontes
	juice := readJuiceIngredients(zr, lang) // suco -> berries que o produzem
	tex := indexItemTextures(zr)   // basename minusculo -> textura no jar
	iconDir := filepath.Join(outDir, "itemtex")
	_ = os.MkdirAll(iconDir, 0o755)
	nIcons := 0

	var items []model.Item
	for key, name := range lang {
		id, ok := strings.CutPrefix(key, "item.pixelmon.")
		if !ok || strings.Contains(id, ".") { // pula subchaves (.tooltip etc.)
			continue
		}
		cat, ev, iv, keep := classifyItem(id)
		desc := cleanTooltip(lang[key+".tooltip"])
		if !keep && desc == "" {
			continue // sem categoria conhecida e sem descricao: provavelmente decoracao
		}
		if !keep {
			switch { // refina alguns "Outros" reconheciveis pelo tooltip
			case strings.Contains(desc, "Mega Stone"):
				cat = "Mega Stone"
			case strings.Contains(desc, "Z-Power"), strings.HasSuffix(id, "ium_z"):
				cat = "Z-Crystal"
			default:
				cat = "Outros"
			}
		}
		icon := extractItemIcon(tex, id, iconDir)
		if icon {
			nIcons++
		}
		items = append(items, model.Item{
			ID: id, Name: name, NamePt: langPt["item."+id],
			Desc: desc, DescPt: translateItemDescPt(desc), Category: cat,
			EVStat: ev, IV: iv, Icon: icon, Drops: drops[id],
			Ingredients: juice[id],
		})
	}
	fmt.Printf("itens: %d texturas extraidas -> %s\n", nIcons, iconDir)

	sort.Slice(items, func(i, j int) bool {
		if items[i].Category != items[j].Category {
			return categoryRank(items[i].Category) < categoryRank(items[j].Category)
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// texAlias cobre itens cujo id nao bate com o nome do arquivo de textura.
var texAlias = map[string]string{
	"gold_bottle_cap":   "golden_bottlecap",
	"silver_bottle_cap": "silver_bottlecap",
}

// indexItemTextures mapeia o basename minusculo (sem .png) -> textura, para todas
// as texturas em assets/pixelmon/textures/items/** (primeira ocorrencia vence).
func indexItemTextures(zr *zip.ReadCloser) map[string]*zip.File {
	const prefix = "assets/pixelmon/textures/items/"
	out := map[string]*zip.File{}
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".png") {
			continue
		}
		base := strings.ToLower(f.Name)
		base = base[strings.LastIndex(base, "/")+1:]
		base = strings.TrimSuffix(base, ".png")
		if _, ok := out[base]; !ok {
			out[base] = f
		}
	}
	return out
}

// extractItemIcon copia a textura do item (id, id sem "_", ou alias) para
// dir/{id}.png. Retorna true se encontrou e escreveu.
func extractItemIcon(tex map[string]*zip.File, id, dir string) bool {
	cands := []string{id, strings.ReplaceAll(id, "_", "")}
	if a := texAlias[id]; a != "" {
		cands = append([]string{a}, cands...)
	}
	for _, c := range cands {
		f := tex[c]
		if f == nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return false
		}
		if os.WriteFile(filepath.Join(dir, id+".png"), data, 0o644) != nil {
			return false
		}
		return true
	}
	return false
}

// juiceColors mapeia a cor (pasta da tag) para o id do item de suco.
var juiceColors = map[string]string{
	"red": "red_juice", "blue": "blue_juice", "pink": "pink_juice",
	"purple": "purple_juice", "yellow": "yellow_juice", "green": "green_juice",
}

// readJuiceIngredients lê as tags `berries/juice/{cor}/{tier}` do jar e devolve,
// para cada suco, as berries que o produzem (aquecendo na Cooking Pot), com o
// tier (1..3). O pinap_juice vem da receita do infuser (uma berry só).
func readJuiceIngredients(zr *zip.ReadCloser, lang map[string]string) map[string][]model.ItemIngredient {
	out := map[string][]model.ItemIngredient{}
	name := func(id string) string {
		if n := lang["item.pixelmon."+id]; n != "" {
			return n
		}
		return prettify(id)
	}
	for color, juice := range juiceColors {
		for tier := 1; tier <= 3; tier++ {
			var tag struct {
				Values []string `json:"values"`
			}
			path := fmt.Sprintf("data/pixelmon/tags/items/berries/juice/%s/%d.json", color, tier)
			if !readZipJSON(zr, path, &tag) {
				continue
			}
			for _, v := range tag.Values {
				if strings.HasPrefix(v, "#") { // ignora referencias a outras tags
					continue
				}
				id := strings.TrimPrefix(v, "pixelmon:")
				out[juice] = append(out[juice], model.ItemIngredient{ID: id, Name: name(id), Tier: tier})
			}
		}
	}
	out["pinap_juice"] = []model.ItemIngredient{{ID: "pinap_berry", Name: name("pinap_berry")}}
	return out
}

// readZipJSON lê um arquivo (por nome exato) do jar e faz o Unmarshal em v.
func readZipJSON(zr *zip.ReadCloser, name string, v any) bool {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return false
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		return json.Unmarshal(data, v) == nil
	}
	return false
}

func readPixelmonLang(zr *zip.ReadCloser) map[string]string {
	return readPixelmonLangFile(zr, "assets/pixelmon/lang/en_us.json")
}

func readPixelmonLangFile(zr *zip.ReadCloser, name string) map[string]string {
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		var m map[string]string
		if json.Unmarshal(data, &m) != nil {
			return nil
		}
		return m
	}
	return nil
}

func readPixelmonDrops(zr *zip.ReadCloser) map[string][]model.ItemDrop {
	out := map[string][]model.ItemDrop{}
	for _, f := range zr.File {
		if f.Name != "data/pixelmon/drops/pokedrops.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return out
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		var entries []pxDropEntry
		if json.Unmarshal(data, &entries) != nil {
			return out
		}
		for _, e := range entries {
			display, slug := parseDropMon(e.Pokemon)
			for _, it := range e.Items {
				id, ok := strings.CutPrefix(it.itemID(), "pixelmon:")
				if !ok {
					continue // so itens do proprio Pixelmon entram no catalogo
				}
				out[id] = append(out[id], model.ItemDrop{
					Pokemon: display, Slug: slug, Chance: it.Chance, Min: it.Min, Max: it.Max,
				})
			}
		}
	}
	// dedup por Pokemon (mantendo a maior chance), ordena por chance desc e limita.
	for id, ds := range out {
		best := map[string]model.ItemDrop{}
		for _, d := range ds {
			if cur, ok := best[d.Slug]; !ok || d.Chance > cur.Chance {
				best[d.Slug] = d
			}
		}
		uniq := make([]model.ItemDrop, 0, len(best))
		for _, d := range best {
			uniq = append(uniq, d)
		}
		sort.SliceStable(uniq, func(i, j int) bool {
			if uniq[i].Chance != uniq[j].Chance {
				return uniq[i].Chance > uniq[j].Chance
			}
			return uniq[i].Pokemon < uniq[j].Pokemon
		})
		if len(uniq) > 12 {
			uniq = uniq[:12]
		}
		out[id] = uniq
	}
	return out
}

// parseDropMon converte o campo "pokemon" do pokedrops ("Rattata form:alolan")
// no nome de exibicao e no slug da pokedex ("Rattata (Alolan)" / "rattata-alolan").
func parseDropMon(name string) (display, slug string) {
	var parts []string
	form := ""
	for _, tok := range strings.Fields(name) {
		if v, ok := strings.CutPrefix(strings.ToLower(tok), "form:"); ok {
			form = v
			continue
		}
		parts = append(parts, tok)
	}
	species := strings.Join(parts, " ")
	slugify := func(s string) string {
		s = strings.ToLower(s)
		s = strings.NewReplacer(".", "", "'", "", ":", "", " ", "-").Replace(s)
		return s
	}
	slug = slugify(species)
	display = species
	if form != "" && form != "base" {
		slug += "-" + form
		display = species + " (" + prettify(form) + ")"
	}
	return display, slug
}

// cleanTooltip normaliza o tooltip do lang (%% -> %, trims, colapsa espacos).
func cleanTooltip(s string) string {
	s = strings.ReplaceAll(s, "%%", "%")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// evStatByItem mapeia itens de EV/vitamina/pena/suco para a stat que afetam (PT).
var evStatByItem = map[string]string{
	"hp_up": "HP", "health_feather": "HP", "purple_juice": "HP", "pomeg_berry": "HP",
	"protein": "Ataque", "muscle_feather": "Ataque", "red_juice": "Ataque", "kelpsy_berry": "Ataque",
	"iron": "Defesa", "resist_feather": "Defesa", "yellow_juice": "Defesa", "qualot_berry": "Defesa",
	"calcium": "Ataque Esp.", "genius_feather": "Ataque Esp.", "blue_juice": "Ataque Esp.", "hondew_berry": "Ataque Esp.",
	"zinc": "Defesa Esp.", "clever_feather": "Defesa Esp.", "green_juice": "Defesa Esp.", "grepa_berry": "Defesa Esp.",
	"carbos": "Velocidade", "swift_feather": "Velocidade", "pink_juice": "Velocidade", "tamato_berry": "Velocidade",
}

var (
	reHasDigit = regexp.MustCompile(`\d`)
	// blocos/decoracao/moveis/maquinario: nao sao "itens" no sentido do catalogo.
	reDecor = regexp.MustCompile(`(stairs|slab|fence|wall|door|trapdoor|button|pressure_plate|planks|_log|_wood$|_wood_|leaves|sapling|chair|_table|_lamp|lantern|_pole|banner|_brick|_tile|roof|window|chimney|curtain|sofa|couch|_bed$|clock|vase|umbrella|chandelier|fossil_machine|trade_machine|_pc$|shrine|statue|_ore$|_block$|bench|stool|_desk|cabinet|drawer|shelf|fridge|stove|sink|counter|_sign|carpet|_rug|mailbox|pillar|fossil_display|elevator|_frame$|_panel|awning|_gate$|ramp|balcony|railing|shoji|tatami|noticeboard|clothes)`)
)

// classifyItem devolve (categoriaPT, evStat, iv, keep). keep=false quando o item
// nao cai numa categoria de gameplay conhecida (o chamador decide pelo tooltip).
func classifyItem(id string) (cat, ev string, iv, keep bool) {
	ev = evStatByItem[id]
	switch {
	case reDecor.MatchString(id):
		return "", ev, false, false
	case strings.HasSuffix(id, "_berry"):
		return "Berry", ev, false, true
	case strings.HasSuffix(id, "_juice"):
		return "Suco", ev, false, true
	case id == "gold_bottle_cap" || id == "silver_bottle_cap":
		return "IV (Bottle Cap)", ev, true, true
	case ev != "" || id == "hp_up" || id == "pp_up" || id == "pp_max" || id == "rare_candy":
		return "Vitamina / EV", ev, false, true
	case strings.HasSuffix(id, "_feather") || strings.HasSuffix(id, "_wing"):
		return "Pena / Asa (EV)", ev, false, true
	case strings.HasSuffix(id, "_fossil"):
		return "Fóssil", ev, false, true
	case strings.HasSuffix(id, "_ball") && id != "iron_ball":
		return "Poké Bola", ev, false, true
	case strings.HasSuffix(id, "_stone") || strings.HasSuffix(id, "_stone_shard") || strings.HasSuffix(id, "_stone_piece"):
		return "Pedra evolutiva", ev, false, true
	case strings.HasSuffix(id, "_incense"):
		return "Incenso", ev, false, true
	case strings.HasSuffix(id, "_plate"):
		return "Placa (Plate)", ev, false, true
	case strings.HasSuffix(id, "_gem"):
		return "Gema de tipo", ev, false, true
	case strings.HasSuffix(id, "_mint"):
		return "Mint (natureza)", ev, false, true
	case strings.HasPrefix(id, "tm") && reHasDigit.MatchString(id), strings.HasPrefix(id, "tr") && reHasDigit.MatchString(id), id == "tm" || id == "hm":
		return "TM / TR", ev, false, true
	case isMedicine(id):
		return "Cura / Medicina", ev, false, true
	case isHeld(id):
		return "Equipável (held)", ev, false, true
	case isEvoItem(id):
		return "Item de evolução", ev, false, true
	default:
		return "", ev, false, false
	}
}

func isMedicine(id string) bool {
	set := map[string]bool{
		"potion": true, "super_potion": true, "hyper_potion": true, "max_potion": true,
		"full_restore": true, "full_heal": true, "revive": true, "max_revive": true,
		"antidote": true, "burn_heal": true, "ice_heal": true, "awakening": true,
		"paralyze_heal": true, "paralyse_heal": true, "ether": true, "max_ether": true,
		"elixir": true, "max_elixir": true, "fresh_water": true, "soda_pop": true,
		"lemonade": true, "moomoo_milk": true, "energy_powder": true, "energy_root": true,
		"heal_powder": true, "revival_herb": true, "sacred_ash": true, "old_gateau": true,
		"sweet_heart": true, "casteliacone": true, "lava_cookie": true, "rage_candy_bar": true,
	}
	return set[id]
}

func isHeld(id string) bool {
	set := map[string]bool{
		"leftovers": true, "choice_band": true, "choice_scarf": true, "choice_specs": true,
		"life_orb": true, "focus_sash": true, "focus_band": true, "assault_vest": true,
		"eviolite": true, "rocky_helmet": true, "black_sludge": true, "iron_ball": true,
		"light_clay": true, "big_root": true, "toxic_orb": true, "flame_orb": true,
		"expert_belt": true, "muscle_band": true, "wise_glasses": true, "scope_lens": true,
		"razor_claw": true, "kings_rock": true, "quick_claw": true, "bright_powder": true,
		"lagging_tail": true, "sticky_barb": true, "shell_bell": true, "metronome": true,
		"wide_lens": true, "zoom_lens": true, "grip_claw": true, "binding_band": true,
		"safety_goggles": true, "protective_pads": true, "float_stone": true, "air_balloon": true,
		"weakness_policy": true, "red_card": true, "eject_button": true, "mental_herb": true,
		"white_herb": true, "power_herb": true, "cell_battery": true, "absorb_bulb": true,
		"snowball": true, "luminous_moss": true, "adrenaline_orb": true, "throat_spray": true,
		"blunder_policy": true, "room_service": true, "heavy_duty_boots": true, "utility_umbrella": true,
		"terrain_extender": true, "damp_rock": true, "heat_rock": true, "smooth_rock": true,
		"icy_rock": true, "amulet_coin": true, "lucky_egg": true, "exp_share": true,
		"soothe_bell": true, "cleanse_tag": true, "smoke_ball": true, "power_weight": true,
		"power_bracer": true, "power_belt": true, "power_lens": true, "power_band": true,
		"power_anklet": true, "destiny_knot": true, "everstone": true,
	}
	return set[id]
}

func isEvoItem(id string) bool {
	set := map[string]bool{
		"metal_coat": true, "dragon_scale": true, "upgrade": true, "dubious_disc": true,
		"protector": true, "electirizer": true, "magmarizer": true, "reaper_cloth": true,
		"prism_scale": true, "sachet": true, "whipped_dream": true, "oval_stone": true,
		"razor_fang": true, "deep_sea_tooth": true, "deep_sea_scale": true, "chipped_pot": true,
		"cracked_pot": true, "galarica_cuff": true, "galarica_wreath": true, "black_augurite": true,
		"strawberry_sweet": true, "love_sweet": true, "berry_sweet": true, "clover_sweet": true,
		"flower_sweet": true, "star_sweet": true, "ribbon_sweet": true, "auspicious_armor": true,
		"malicious_armor": true, "linking_cord": true, "sweet_apple": true, "tart_apple": true,
	}
	return set[id]
}

// categoryRank ordena as categorias na UI (EV/IV primeiro por serem o foco).
func categoryRank(cat string) int {
	order := []string{
		"Berry", "Suco", "Vitamina / EV", "Pena / Asa (EV)", "IV (Bottle Cap)",
		"Cura / Medicina", "Poké Bola", "Pedra evolutiva", "Item de evolução",
		"Equipável (held)", "Mega Stone", "Z-Crystal", "Placa (Plate)", "Gema de tipo", "Incenso",
		"Mint (natureza)", "Fóssil", "TM / TR", "Outros",
	}
	for i, c := range order {
		if c == cat {
			return i
		}
	}
	return len(order)
}
