// Nicks-Pokedex: pokedex pessoal + wiki de Cobblemon, feita em Go.
//
// Uso:
//
//	go run . import                  # gera data/pokedex.json e data/moves.json
//	go run . import -offline         # sem baixar stats de move (Showdown)
//	go run . serve                   # sobe o servidor web em http://localhost:8080
package main

import (
	"flag"
	"fmt"
	iofs "io/fs"
	"os"

	"nickspokedex/internal/importer"
	"nickspokedex/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "import":
		fs := flag.NewFlagSet("import", flag.ExitOnError)
		jar := fs.String("jar", os.Getenv("COBBLEMON_JAR"), "caminho do .jar do Cobblemon (auto-detecta se vazio)")
		out := fs.String("data", "data", "pasta de saida dos JSON")
		offline := fs.Bool("offline", false, "nao baixar stats de move do Showdown")
		nosprites := fs.Bool("nosprites", false, "nao baixar os sprites pixelart")
		_ = fs.Parse(os.Args[2:])

		err := importer.Run(importer.Options{JarPath: *jar, OutDir: *out, Offline: *offline, Sprites: !*nosprites})
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro na importacao:", err)
			os.Exit(1)
		}

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := fs.String("addr", ":8080", "endereco de escuta")
		data := fs.String("data", "data", "pasta com os JSON gerados")
		_ = fs.Parse(os.Args[2:])

		sub, err := iofs.Sub(webFS, "web")
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro ao abrir assets:", err)
			os.Exit(1)
		}
		if err := server.Run(*addr, *data, sub); err != nil {
			fmt.Fprintln(os.Stderr, "erro no servidor:", err)
			os.Exit(1)
		}

	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `Nicks-Pokedex

Comandos:
  import [-jar PATH] [-data DIR] [-offline]   gera os JSON a partir do Cobblemon
  serve  [-addr :8080] [-data DIR]            sobe o servidor web`)
}
