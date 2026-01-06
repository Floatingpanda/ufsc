package main

import (
	"embed"
	"html/template"
	"log"
	"os"
	"upforschool/internal/app"
)

//go:embed templates/*.html
var tmplFS embed.FS

func loadTemplatesProd() *template.Template {
	t := template.New("")
	return template.Must(t.ParseFS(tmplFS, "templates/*.html"))
}

func loadTemplatesDev() *template.Template {
	return template.Must(template.ParseGlob("templates/*.html"))
}

var templates *template.Template

func init() {
	env := os.Getenv("MODE")

	if env == "PROD" {
		templates = loadTemplatesProd()
	} else {
		templates = loadTemplatesDev()
	}
}

func main() {
	log.Println("welcome to upforschool")

	cfg, err := app.ReadConfig("config.json")
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	app.Run()

}
