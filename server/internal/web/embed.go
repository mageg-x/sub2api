package web

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/*
var embedded embed.FS

func Dist() (fs.FS, error) {
	return fs.Sub(embedded, "dist")
}
