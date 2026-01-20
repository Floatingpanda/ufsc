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
	"upforschool/internal/postmark"
	"upforschool/internal/upforauth"
	"upforschool/internal/viewer"

	"github.com/gorilla/securecookie"
)

func cookie(c *Config) *securecookie.SecureCookie {
	return securecookie.New([]byte(c.Cookie.HashKey), []byte(c.Cookie.BlockKey))
}

type mailer interface {
	SendActivationEmail(name, toEmail, tokenID, tokenValue string)
}

// App structure.
type App struct {
	config *Config
	wl     *worldline.Worldline
	core   *model.Core
	repo   *model.Repository
	view   *viewer.Viewer
	router *http.ServeMux
	cookie *securecookie.SecureCookie
	auth   *upforauth.Service
	email  mailer
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

	r.HandleFunc("/policy", a.page("policy.html"))
	r.HandleFunc("/terms", a.page("terms.html"))
	r.HandleFunc("/cookies", a.page("cookies.html"))

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

	r.HandleFunc("/role/switch", a.handleAuth(a.handleSwitchRole))

	r.HandleFunc("/profile", a.handleAuth(a.handleProfile))
	a.router.HandleFunc("/profile/edit", a.handleAuth(a.handleProfilesEdit()))
	a.router.HandleFunc("/profiles/image", a.handleAuth(a.handleProfileImage()))
	a.router.HandleFunc("/profiles/password", a.handleAuth(a.handleProfilePassword()))
	a.router.HandleFunc("/profiles/email", a.handleAuth(a.handleProfileEmail()))

	r.HandleFunc("/lesson/new", a.handleAuth(a.handleNewLesson))
	r.HandleFunc("/lesson/edit", a.handleAuth(a.handleLessonEdit))

	r.HandleFunc("GET /tutors/list", a.handleAuth(a.handleGetTutors))
	r.HandleFunc("GET /lessons/list", a.handleAuth(a.handleListLessons))
	r.HandleFunc("GET /lesson/accept", a.handleAuth(a.handleLessonAccept))
	r.HandleFunc("GET /lesson/delete", a.handleAuth(a.handleLessonDelete))

	r.HandleFunc("GET /tutor/summary", a.handleAuth(a.handleGetTutorSummary))

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

	wl := worldline.New(
		c.App.URL,
		c.Worldline.Merchant,
		c.Worldline.Username,
		c.Worldline.Password,
		c.Worldline.MD5key)

	props := make(map[string]any)
	props["AppURL"] = c.App.URL
	props["Static"] = c.App.StaticPath
	// props["MapsKey"] = c.App.MapsKey
	// props["GoogleAnalytics"] = c.App.GoogleAnalytics

	modelReposity := model.NewRepository(db)

	modelCore := model.NewCore(db, wl)

	icons := map[string]string{
		"Biologi":         "icon-biology.png",
		"Ekonomi":         "icon-economy.png",
		"Engelska":        "icon-language.png",
		"Franska":         "icon-language.png",
		"Fysik":           "icon-physics.png",
		"Historia":        "icon-history.png",
		"Italienska":      "icon-language.png",
		"Juridik":         "icon-law.png",
		"Kemi":            "icon-chemistry.png",
		"Matematik":       "icon-math.png",
		"Nationella prov": "icon-math.png",
		"Programmering":   "icon-programming.png",
		"Psykologi":       "icon-psychology.png",
		"Samhällskunskap": "icon-social.png",
		"Spanska":         "icon-language.png",
		"Svenska":         "icon-language.png",
		"Tyska":           "icon-language.png",
		"Övriga språk":    "icon-language.png",
		"Övriga ämnen":    "icon-language.png",
	}

	v := viewer.New(files, props)
	v.Funcs(template.FuncMap{
		"JSON": func(item any) string {
			result, _ := json.Marshal(item)
			return string(result)
		},
		"multiply": func(a int64, b int) int64 {
			return a * int64(b)
		},
		"icons": func(key string) string {

			if val, ok := icons[key]; ok {
				return val
			}
			return "icon-language.png"
		},
	})

	if c.App.IsDev {
		v.Dev()
	}

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

	// email := mailer.NewService(v, db)
	// if err != nil {
	// 	return nil, err
	// }

	emailSvc := postmark.NewService(c.PostmarkToken)

	a := &App{
		config: c,
		wl:     wl,
		view:   v,
		auth:   auth,
		repo:   modelReposity,
		core:   modelCore,
		bankid: bid,
		cookie: cookie(c),
		email:  emailSvc,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		jwt: jwtSvc,
	}

	a.routes()

	return a, nil
}

func (a *App) Run() {
	log.Fatal(http.ListenAndServe(a.config.App.Addr, a.router))
}
