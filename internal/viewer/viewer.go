package viewer

import (
	"html/template"
	"io"
	"sync"
)

// Page model.
type Page struct {
	Name  string
	Props map[string]any
	Data  map[string]any
}

// Add data to page.
func (p *Page) Add(key string, value any) *Page {
	p.Data[key] = value
	return p
}

// Viewer for templates.
type Viewer struct {
	props   map[string]any
	files   []string
	tmplFn  func() (*template.Template, error)
	methods template.FuncMap
}

// Template returns template.
func (v *Viewer) Template() (*template.Template, error) {
	return v.tmplFn()
}

// Dev Viewer.
func (v *Viewer) Dev() *Viewer {
	v.tmplFn = func() (*template.Template, error) {
		return template.New("upforschool").Funcs(v.methods).ParseFiles(v.files...)
	}
	return v
}

// Once Viewer.
func (v *Viewer) Once() *Viewer {
	var (
		init   sync.Once
		tpl    *template.Template
		tplerr error
	)

	v.tmplFn = func() (*template.Template, error) {
		init.Do(func() {
			tpl, tplerr = template.New("upforschool").Funcs(v.methods).ParseFiles(v.files...)
		})
		return tpl, tplerr
	}

	return v
}

// Page creates new page.
func (v *Viewer) Page(name string) *Page {
	return &Page{
		Name:  name,
		Props: v.props,
		Data:  make(map[string]any),
	}
}

// Execute template.
func (v *Viewer) Execute(w io.Writer, p *Page) error {
	t, err := v.Template()
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, p.Name, p)
}

func (v *Viewer) Funcs(methods template.FuncMap) {
	v.methods = methods
}

// New Viewer with props.
func New(files []string, props map[string]any) *Viewer {
	v := &Viewer{files: files, props: props, methods: template.FuncMap{}}
	return v.Once()
}
