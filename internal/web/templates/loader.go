package templates

import (
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/gaetanlhf/ZIMServer/html"
)

type Templates struct {
	templates map[string]*template.Template
}

type noListingFS struct {
	fs http.FileSystem
}

var timeZero = time.Time{}

func Load() (*Templates, error) {
	templates := make(map[string]*template.Template)

	homeTemplate, err := template.ParseFS(html.TemplatesFS, "static/templates/base.html", "static/templates/home.html")
	if err != nil {
		return nil, err
	}
	templates["home"] = homeTemplate

	viewerTemplate, err := template.ParseFS(html.TemplatesFS, "static/templates/base.html", "static/templates/viewer.html")
	if err != nil {
		return nil, err
	}
	templates["viewer"] = viewerTemplate

	return &Templates{templates: templates}, nil
}

func (t *Templates) Render(w http.ResponseWriter, name string, data interface{}) error {
	tmpl, exists := t.templates[name]
	if !exists {
		return http.ErrNotSupported
	}

	return tmpl.ExecuteTemplate(w, "base.html", data)
}

func GetAssetsFS() http.FileSystem {
	assetsSubFS, err := fs.Sub(html.AssetsFS, "static/assets")
	if err != nil {
		panic(err)
	}

	return noListingFS{http.FS(assetsSubFS)}
}

func (nfs noListingFS) Open(name string) (http.File, error) {
	f, err := nfs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if stat.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}

	return f, nil
}
