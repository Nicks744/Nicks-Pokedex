# Nick's Pokédex

Pokédex pessoal + wiki de **Cobblemon**, feita em Go (stdlib, **zero dependências**).
Busca instantânea, filtros, moveset por level e análise de fraquezas do seu time.

## Como rodar

```bash
# 1) Gera a base a partir do .jar do Cobblemon (roda uma vez).
#    Baixa também as stats dos golpes do Pokémon Showdown.
go run . import

# 2) Sobe o servidor web.
go run . serve
# abre http://localhost:8080
```

### Opções

```bash
go run . import -jar "CAMINHO\Cobblemon-....jar"   # usar outro jar (senão, auto-detecta)
go run . import -pixeljar "CAMINHO\Pixelmon-...jar" # spawns do Pixelmon (opcional; auto-detecta)
go run . import -offline                            # sem baixar stats de move
go run . serve  -addr :9000                         # outra porta
go run . build  -out docs -base /Repo/             # gera o site estático
```

## Deploy (GitHub Pages, grátis)

O comando `build` pré-renderiza todas as páginas em `docs/` (site 100% estático;
Time e Histórico ficam no `localStorage` do navegador). O workflow
[`.github/workflows/pages.yml`](.github/workflows/pages.yml) builda e publica no
GitHub Pages **a cada push na `main`** — sem servidor, sem banco, sem custo. Usa
os dados versionados em `data/`, então não precisa do jar do Cobblemon na nuvem.

O caminho padrão do jar aponta para o Cobblemon detectado na máquina
(`main.go`, constante `defaultJar`). Ao atualizar o mod, troque a constante
ou passe `-jar`. Depois da importação o app funciona **100% offline**.

## O que tem

- **Pokédex** (`/`): busca em tempo real + filtros por tipo, geração e ordenação,
  com **sprite pixelart** em cada card e um **botão "+"** para adicionar ao time
  sem sair da página. Inclui **todas as formas** — regionais (Alola/Galar/Hisui/
  Paldea), Mega, Primal, Origin etc. —, cada uma como sua própria entrada.
- **Wiki do Pokémon** (`/pokemon/{slug}`): sprite, tipos, base stats, habilidades,
  matchups defensivos e **golpes por level / TM / ovo / tutor** (com **gem do tipo**,
  categoria, poder, precisão e PP).
  - **Foco + carrossel evolutivo**: a foto grande é o centro, com setas ◀ ▶ e uma
    tira de miniaturas para percorrer a família (com ramificações tipo Eevee) e um
    botão **Ver condições** que revela em que level/condição cada evolução acontece.
  - **Abas de forma** logo abaixo do nome para alternar entre as versões da espécie
    (ex.: Kanto ↔ Alola).
  - **Onde encontrar**: onde/como conseguir o Pokémon, com seções separadas para
    **Cobblemon** e **Pixelmon** — raridade, faixa de level, **biomas (em inglês,
    como no jogo)** e condições (dia/noite, chuva, Nether/End, pesca, curry, etc.).
  - **Natureza & Builds**: natureza que mais combina com base nas stats + builds
    sugeridas (papel, natureza, EVs e golpes tirados do que ele aprende).
- **Ataques** (`/moves`): browser de todos os golpes — busca, filtro por **tipo**
  (com ícone) e **categoria** (Físico/Especial/Status), ordenação e um **modal**
  com os detalhes completos (poder, precisão, PP e efeito).
- **Página de golpe** (`/move/{slug}`): quem aprende e como.
- **Itens** (`/items`): catálogo dos itens do **Pixelmon** — o que cada um faz
  (tooltip do mod) e **de quais Pokémon dropa, com a chance** (de `pokedrops.json`).
  Busca, filtro por categoria e "só com drop".
- **Berries & Sucos** (`/berries`): todas as berries e o que fazem, com foco em
  **EV/IV** — sucos e vitaminas (EV), penas/asas (+1 EV), berries que reduzem EV
  e **Bottle Caps** (Hyper Training de IV), com um guia rápido de como treinar.
- **Ícones de tipo**: sempre que um Pokémon ou golpe tem tipo, a interface mostra
  o **ícone redondo do tipo** (círculo colorido + símbolo) ao lado do rótulo — nos
  cards, badges, tabelas de golpes, matchups e filtros. Ícones vetoriais (SVG) em
  `web/static/img/types/`, nítidos em qualquer tamanho.
- **Meu Time** (`/team`): monte até 6 e veja a **matriz de cobertura defensiva**
  — quais tipos são o buraco do time. Salvo em `data/team.json`.
- **Histórico** (`/history`): registra automaticamente os Pokémon que você abriu,
  do mais recente ao mais antigo. Salvo em `data/history.json`.

Os **sprites** são baixados uma vez do PokéAPI para `data/sprites/{dex}.png`
(offline depois). Use `import -nosprites` para pular esse passo. Os **gems de tipo**
são extraídos do próprio jar do Cobblemon para `data/types/{tipo}.png`.

## Estrutura

```
main.go              dispatcher (import | serve) + embed do web/
internal/
  model/             structs compartilhadas
  typechart/         tabela de efetividade de tipos (Gen 6+)
  importer/          jar Cobblemon + lang + Showdown -> data/*.json
  store/             carrega dex/moves em memória + CRUD do time (JSON)
  server/            HTTP: páginas (html/template) + API JSON
web/
  templates/         partials reutilizáveis + páginas
  static/css/        design system (tokens + componentes, responsivo)
  static/js/         Web Components + busca ao vivo + botão de time
data/                gerado pelo import (pokedex.json, moves.json, team.json)
```

## Notas

- A base é gerada, não versionada como fonte — rode `import` após clonar.
- Os dados de golpe vêm do Pokémon Showdown (idênticos aos jogos oficiais,
  que o Cobblemon segue); nomes/descrições de Pokémon e golpes vêm do lang do mod.
