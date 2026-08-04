// Tradução PT-BR das descrições dos itens do Pixelmon. O jar só traduz os NOMES
// (item.{id}); as descrições (tooltips) não têm PT no mod, então traduzimos aqui:
// (1) templates para as famílias formulaicas (Mega Stones, Z-Crystals, Plates,
// Gems, berries, vitaminas…) e (2) um dicionário embutido (items_pt.json) para o
// resto. Estrutura pronta pra atualizar: é só editar o JSON ou os templates.
package importer

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strings"
)

//go:embed items_pt.json
var itemsPtRaw []byte

var itemDescDict = func() map[string]string {
	m := map[string]string{}
	_ = json.Unmarshal(itemsPtRaw, &m)
	return m
}()

// typePt traduz o nome de um tipo (Fire -> Fogo) para as descrições.
var typePt = map[string]string{
	"Normal": "Normal", "Fire": "Fogo", "Water": "Água", "Electric": "Elétrico",
	"Grass": "Planta", "Ice": "Gelo", "Fighting": "Lutador", "Poison": "Venenoso",
	"Ground": "Terra", "Flying": "Voador", "Psychic": "Psíquico", "Bug": "Inseto",
	"Rock": "Pedra", "Ghost": "Fantasma", "Dragon": "Dragão", "Dark": "Sombrio",
	"Steel": "Aço", "Fairy": "Fada",
}

// statPhrase converte uma stat em inglês numa frase PT com artigo (para "Lowers X").
var statPhrase = map[string]string{
	"Max HP": "o HP máximo", "HP": "o HP", "Attack": "o Ataque", "Defense": "a Defesa",
	"Sp. Attack": "o Ataque Especial", "Sp. Defense": "a Defesa Especial",
	"Special Attack": "o Ataque Especial", "Special Defense": "a Defesa Especial",
	"Speed": "a Velocidade",
}

// statName converte uma stat em inglês no rótulo PT (para EVs).
var statName = map[string]string{
	"HP": "HP", "Attack": "Ataque", "Defense": "Defesa",
	"Special Attack": "Ataque Especial", "Special Defense": "Defesa Especial",
	"Sp. Attack": "Ataque Especial", "Sp. Defense": "Defesa Especial", "Speed": "Velocidade",
}

// statPinch: stat usada em "Raises X in a pinch".
var statPinch = map[string]string{
	"Attack": "o Ataque", "Defense": "a Defesa", "Speed": "a Velocidade",
	"Sp. Attack": "o Ataque Especial", "Sp. Defense": "a Defesa Especial",
	"Special Attack": "o Ataque Especial", "Special Defense": "a Defesa Especial",
}

// curePt traduz o alvo de "Cures X".
var curePt = map[string]string{
	"burn": "a queimadura", "paralysis": "a paralisia", "sleep": "o sono",
	"poison": "o envenenamento", "freeze": "o congelamento", "confusion": "a confusão",
	"infatuation": "a atração", "stat drops": "as reduções de atributo",
	"status effects": "os efeitos de status", "confusion.": "a confusão",
	"paralysis, poison, burn, sleep, freeze, confusion, or infatuation": "paralisia, veneno, queimadura, sono, congelamento, confusão ou atração",
}

var (
	reMega    = regexp.MustCompile(`^One of a variety of mysterious Mega Stones\. Have (.+?) hold it, and this stone will enable it to Mega Evolve during battle\.$`)
	reZ       = regexp.MustCompile(`^A crystallized form of Z-Power\. It upgrades (.+?)'s (.+?) to a Z-Move\.$`)
	rePlate   = regexp.MustCompile(`^A held item that raises the power of the holder's (\w+)-type moves by 20%\.$`)
	reGemBer  = regexp.MustCompile(`^Reduces the effect of a super-effective (\w+) move by 50%\. Can be infused into a Crystal Block to make a \w+ Gem\.$`)
	reLowEV   = regexp.MustCompile(`^Lowers (.+?) and increases Happiness$`)
	rePinch   = regexp.MustCompile(`^Raises (Attack|Defense|Speed|Sp\. Attack|Sp\. Defense) in a pinch\. 10x infused into a (.+? Wing) will create an? (.+?)\.$`)
	reIncense = regexp.MustCompile(`^Infuse 3x into an Incense Burner to make an? (.+?)\.$`)
	reVitamin = regexp.MustCompile(`^Boosts a Pokémon's (.+?) EVs by 10\. Can be obtained by infusing a (.+? Wing) with 10 (.+? Berries)\.$`)
	reFeather = regexp.MustCompile(`^Increases the (.+?) EV of a Pokémon by 1\. Can be infused with 10 (.+? Berries) to create an? (.+?)\.$`)
	reCure    = regexp.MustCompile(`^Cures (.+)$`)
	reHP      = regexp.MustCompile(`^Restores (\d+) HP\.?$`)
	reNature  = regexp.MustCompile(`^Restores 12\.5% of maximum HP, confuses Pokémon with (.+?)-lowering Natures\.$`)

	reZType   = regexp.MustCompile(`^A crystallized form of Z-Power\. It upgrades (\w+)-type moves to Z-Moves\.$`)
	reMemory  = regexp.MustCompile(`^A memory disc that contains (\w+)-type data\. It changes the type of the holder if held by a certain species of Pokémon\.$`)
	reTypeGem = regexp.MustCompile(`^A held item that boosts the power of a single (\w+)-type move by 1\.3×, being consumed upon use\. Created by infusing a Crystal Block with an? (.+?) Berry\.$`)
	reGenes   = regexp.MustCompile(`^A cassette to be held by Genesect\. It changes Genesect's Techno Blast move so it becomes (\w+) type\.$`)
	reSweet   = regexp.MustCompile(`^A (\w+)-shaped sweet\. When a Milcery holds this, it (?:will )?spins? around happily\.$`)
	reCandy   = regexp.MustCompile(`^A candy that is packed with energy\. When consumed, it will grant a single Pokémon (.+?) of Exp\. Points\.$`)
	reStone   = regexp.MustCompile(`^A peculiar stone that can make certain species of Pokémon evolve\. (.+)$`)
	reReset   = regexp.MustCompile(`^An item that sharply boosts the (.+?) (?:stat )?of a Pokémon during a battle\. It wears off once the Pokémon is withdrawn\.$`)
)

var sweetPt = map[string]string{
	"berry": "fruta", "clover": "trevo", "flower": "flor", "heart": "coração",
	"ribbon": "fita", "star": "estrela", "strawberry": "morango",
}
var candyPt = map[string]string{
	"a very small amount": "uma quantidade muito pequena", "a small amount": "uma pequena quantidade",
	"a moderate amount": "uma quantidade moderada", "a large amount": "uma grande quantidade",
	"a very large amount": "uma quantidade enorme",
}
var battleStatPt = map[string]string{
	"accuracy": "a precisão", "Attack": "o Ataque", "Defense": "a Defesa",
	"Sp. Atk": "o Ataque Especial", "Sp. Def": "a Defesa Especial", "Speed": "a Velocidade",
}

// stoneFlavorPt traduz a frase de sabor das pedras evolutivas "peculiar stone".
var stoneFlavorPt = map[string]string{
	"It burns as red as the evening sun.":                "Arde tão vermelha quanto o sol poente.",
	"It has a distinct thunderbolt pattern.":             "Tem um padrão nítido de raio.",
	"It has an unmistakable leaf pattern.":               "Tem um inconfundível padrão de folha.",
	"It has an unmistakable snowflake pattern.":          "Tem um inconfundível padrão de floco de neve.",
	"It holds shadows as dark as can be.":                "Guarda sombras tão escuras quanto possível.",
	"It is as black as the night sky.":                   "É tão negra quanto o céu noturno.",
	"It is the blue of a pool of clear water.":           "Tem o azul de uma poça de água cristalina.",
	"It shines with a dazzling light.":                   "Brilha com uma luz deslumbrante.",
	"It sparkles like a glittering eye.":                 "Cintila como um olho reluzente.",
	"It's as round as a Pokémon Egg.":                    "É tão redonda quanto um Ovo de Pokémon.",
	"The stone has a fiery orange heart.":                "A pedra tem um coração alaranjado e ardente.",
}

// translateItemDescPt devolve a descrição em PT (via dicionário ou template) ou
// "" se não houver tradução (a UI então cai no inglês).
func translateItemDescPt(desc string) string {
	if desc == "" {
		return ""
	}
	if v, ok := itemDescDict[desc]; ok {
		return v
	}
	if m := reMega.FindStringSubmatch(desc); m != nil {
		return "Uma das várias Mega Stones misteriosas. Faça " + m[1] + " segurá-la para que possa Mega Evoluir durante a batalha."
	}
	if m := reZ.FindStringSubmatch(desc); m != nil {
		return "Uma forma cristalizada de Z-Power. Transforma o golpe " + m[2] + " de " + m[1] + " num Movimento-Z."
	}
	if m := rePlate.FindStringSubmatch(desc); m != nil {
		return "Item equipável que aumenta em 20% o poder dos golpes do tipo " + typeOrRaw(m[1]) + " de quem o segura."
	}
	if m := reGemBer.FindStringSubmatch(desc); m != nil {
		t := typeOrRaw(m[1])
		return "Reduz em 50% o efeito de um golpe supereficaz do tipo " + t + ". Pode ser infundida num Bloco de Cristal para criar uma Gema de " + t + "."
	}
	if m := reLowEV.FindStringSubmatch(desc); m != nil {
		p := statPhrase[m[1]]
		if p == "" {
			p = m[1]
		}
		return "Reduz " + p + " e aumenta a felicidade."
	}
	if m := rePinch.FindStringSubmatch(desc); m != nil {
		s := statPinch[m[1]]
		if s == "" {
			s = m[1]
		}
		return "Aumenta " + s + " numa emergência. Infundida 10x numa " + m[2] + " cria " + article(m[3]) + m[3] + "."
	}
	if m := reIncense.FindStringSubmatch(desc); m != nil {
		return "Infunda 3x num Queimador de Incenso para criar o " + m[1] + "."
	}
	if m := reVitamin.FindStringSubmatch(desc); m != nil {
		s := statName[m[1]]
		if s == "" {
			s = m[1]
		}
		return "Aumenta em 10 os EVs de " + s + " do Pokémon. Obtida infundindo uma " + m[2] + " com 10 " + m[3] + "."
	}
	if m := reFeather.FindStringSubmatch(desc); m != nil {
		s := statName[m[1]]
		if s == "" {
			s = m[1]
		}
		return "Aumenta em 1 o EV de " + s + " do Pokémon. Pode ser infundida com 10 " + m[2] + " para criar " + article(m[3]) + m[3] + "."
	}
	if m := reNature.FindStringSubmatch(desc); m != nil {
		s := statName[m[1]]
		if s == "" {
			s = m[1]
		}
		return "Restaura 12,5% do HP máximo; confunde Pokémon com naturezas que reduzem " + s + "."
	}
	if m := reZType.FindStringSubmatch(desc); m != nil {
		return "Uma forma cristalizada de Z-Power. Transforma golpes do tipo " + typeOrRaw(m[1]) + " em Movimentos-Z."
	}
	if m := reMemory.FindStringSubmatch(desc); m != nil {
		return "Um disco de memória com dados do tipo " + typeOrRaw(m[1]) + ". Muda o tipo de quem o segura, se for de uma certa espécie."
	}
	if m := reTypeGem.FindStringSubmatch(desc); m != nil {
		return "Item equipável que aumenta em 1,3× o poder de um único golpe do tipo " + typeOrRaw(m[1]) + ", sendo consumido ao usar. Criado infundindo um Bloco de Cristal com uma " + m[2] + " Berry."
	}
	if m := reGenes.FindStringSubmatch(desc); m != nil {
		return "Uma fita para o Genesect segurar. Muda o golpe Techno Blast dele para o tipo " + typeOrRaw(m[1]) + "."
	}
	if m := reSweet.FindStringSubmatch(desc); m != nil {
		s := sweetPt[strings.ToLower(m[1])]
		if s == "" {
			s = m[1]
		}
		return "Um doce em formato de " + s + ". Quando uma Milcery o segura, ela roda de felicidade."
	}
	if m := reCandy.FindStringSubmatch(desc); m != nil {
		a := candyPt[m[1]]
		if a == "" {
			a = m[1]
		}
		return "Um doce cheio de energia. Ao ser consumido, concede a um único Pokémon " + a + " de Pontos de Exp."
	}
	if m := reStone.FindStringSubmatch(desc); m != nil {
		if f, ok := stoneFlavorPt[strings.TrimSpace(m[1])]; ok {
			return "Uma pedra peculiar que faz certas espécies de Pokémon evoluírem. " + f
		}
	}
	if m := reReset.FindStringSubmatch(desc); m != nil {
		s := battleStatPt[strings.TrimSpace(m[1])]
		if s == "" {
			s = m[1]
		}
		return "Item que aumenta bruscamente " + s + " de um Pokémon durante a batalha. O efeito passa quando o Pokémon é recolhido."
	}
	if m := reCure.FindStringSubmatch(desc); m != nil {
		t := strings.TrimRight(strings.TrimSpace(m[1]), ".")
		if v, ok := curePt[strings.ToLower(t)]; ok {
			return "Cura " + v + "."
		}
		if v, ok := curePt[t]; ok {
			return "Cura " + v + "."
		}
	}
	if m := reHP.FindStringSubmatch(desc); m != nil {
		return "Restaura " + m[1] + " de HP."
	}
	return ""
}

func typeOrRaw(t string) string {
	if v, ok := typePt[t]; ok {
		return v
	}
	return t
}

// article devolve "um "/"uma " conforme o item (heurística simples p/ vitaminas).
func article(name string) string {
	switch name {
	case "Iron", "HP Up", "Zinc":
		return "um "
	case "Protein", "Calcium":
		return "uma "
	case "Carbos":
		return "um "
	}
	return "um "
}
