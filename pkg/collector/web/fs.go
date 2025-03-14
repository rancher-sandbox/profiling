package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static/*
var embeddedFiles embed.FS

//go:embed templates/*
var embeddedTemplates embed.FS

func static() (http.Handler, error) {
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(fsys)), nil
}

func htmlTemplates() (*template.Template, error) {
	tmpl := template.New("example").Funcs(template.FuncMap{
		"splitString": strings.Split,
	})
	ts, err := tmpl.ParseFS(embeddedTemplates, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	ts = ts.Funcs(template.FuncMap{
		"split": strings.Split,
	})
	return ts, nil
}
