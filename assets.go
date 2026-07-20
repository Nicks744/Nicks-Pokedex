package main

import "embed"

// webFS embute os templates e assets estaticos no binario, para o `serve`
// funcionar de qualquer diretorio sem depender da pasta web/ em disco.
//
//go:embed web
var webFS embed.FS
