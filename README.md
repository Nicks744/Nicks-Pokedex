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
  sem sair da página.
- **Wiki do Pokémon** (`/pokemon/{slug}`): sprite, tipos, base stats, habilidades,
  matchups defensivos e **golpes por level / TM / ovo / tutor** (com **gem do tipo**,
  categoria, poder, precisão e PP).
  - **Linha evolutiva** completa (a família toda, com ramificações tipo Eevee) e
    botão **Inspecionar** que revela em que level/condição cada evolução acontece.
  - **Natureza & Builds**: natureza que mais combina com base nas stats + builds
    sugeridas (papel, natureza, EVs e golpes tirados do que ele aprende).
- **Página de golpe** (`/move/{slug}`): quem aprende e como.
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
