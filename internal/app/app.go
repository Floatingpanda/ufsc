package app

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
	"upforschool/internal/bankid"
	"upforschool/internal/database"
	"upforschool/internal/jwt"
	"upforschool/internal/model"
	"upforschool/internal/pkg/worldline"
	"upforschool/internal/upforauth"
	"upforschool/internal/viewer"

	"upforschool/internal/mailer"

	"github.com/gorilla/securecookie"
)

func cookie(c *Config) *securecookie.SecureCookie {
	return securecookie.New([]byte(c.Cookie.HashKey), []byte(c.Cookie.BlockKey))
}

// App structure.
type App struct {
	wl     *worldline.Worldline
	core   *model.Core
	repo   *model.Repository
	view   *viewer.Viewer
	router *http.ServeMux
	cookie *securecookie.SecureCookie
	auth   *upforauth.Service
	email  *mailer.Service
	bankid *bankid.BankID
	client *http.Client
	jwt    *jwt.Service
}

// func (a *App) profile(r *http.Request) *model.Profile {
// 	return r.Context().Value(model.ContextKeyProfile).(*model.Profile)
// }

func (a *App) routes() {
	a.router = http.NewServeMux()

	r := a.router

	static := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))

	r.HandleFunc("/static/", a.handleStatic(static))
	r.HandleFunc("/", a.handleIsLoggedIn(a.indexHandler))

	// a.router.HandleFunc("/home", a.handleAuth(a.handleHome()))

	// bankid handlers
	r.HandleFunc("/bankid/start", corsMiddleware(startAuthHandler))
	r.HandleFunc("/bankid/collect", corsMiddleware(a.collectAuthHandler))
	r.HandleFunc("/bankid/qrcode", corsMiddleware(generateQRHandler))
	r.HandleFunc("/bankid/cancel", corsMiddleware(cancelAuthHandler))

	r.HandleFunc("/auth/login", a.handleIsLoggedIn(a.handleLogin()))
	r.HandleFunc("/auth/logout", a.handleLogout())

	r.HandleFunc("/signup", a.page("signup-pick.html"))
	r.HandleFunc("/signup-student", a.handleSignupStudent())
	r.HandleFunc("/signup-tutor", a.handleSignupTutor())
	r.HandleFunc("/auth/confirm/", a.handleConfirm())

	r.HandleFunc("/profile", a.handleAuth(a.handleProfile))

	r.HandleFunc("/lessons/new", a.handleAuth(a.handleNewLesson))

	r.HandleFunc("GET /list/tutors", a.handleAuth(a.handleGetTutors))

	r.HandleFunc("/home", a.handleAuth(a.homeHandler))
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// New creates app for config.
func New(c *Config) (*App, error) {
	db, err := database.New(c.DB)
	if err != nil {
		return nil, err
	}

	auth := upforauth.New(db)

	secret := "upfor-school-happy-dolphin"
	jwtSvc := jwt.New(jwt.JWTConfig{
		Issuer:          "ufs",
		SingleSecretKey: &secret,
	})

	files, err := filepath.Glob(c.App.Templates)
	if err != nil {
		return nil, err
	}

	props := make(map[string]any)
	props["AppURL"] = c.App.URL
	props["Static"] = c.App.StaticPath
	// props["MapsKey"] = c.App.MapsKey
	// props["GoogleAnalytics"] = c.App.GoogleAnalytics

	modelReposity := model.NewRepository(db)

	modelCore := model.NewCore(db)

	v := viewer.New(files, props)
	if c.App.IsDev {
		v.Dev()
	}

	v.Funcs(template.FuncMap{
		"JSON": func(item any) string {
			result, _ := json.Marshal(item)
			return string(result)
		},
		"multiply": func(a int64, b int) int64 {
			return a * int64(b)
		},
	})

	wl := worldline.New(
		c.App.URL,
		c.Worldline.Merchant,
		c.Worldline.Username,
		c.Worldline.Password,
		c.Worldline.MD5key)

	bid, err = bankid.New(&bankid.Config{
		BaseURL:         c.BankID.BaseURL,
		CertificatePath: c.BankID.CertificatePath,
		CertificatePass: c.BankID.CertificatePass,
		CaPath:          c.BankID.CaPath,
		Insecure:        c.App.IsDev,
	})
	if err != nil {
		log.Fatal(err)
	}

	email := mailer.NewService(v, db)
	if err != nil {
		return nil, err
	}

	a := &App{
		wl:     wl,
		view:   v,
		auth:   auth,
		repo:   modelReposity,
		core:   modelCore,
		bankid: bid,
		cookie: cookie(c),
		email:  email,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		jwt: jwtSvc,
	}

	a.routes()

	return a, nil
}

func (a *App) Run() {
	log.Fatal(http.ListenAndServe(":8080", a.router))
}
