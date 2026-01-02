package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"upforschool/internal/model"
)

func (a *App) handleStatic(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "max-age=3600")
		next.ServeHTTP(w, r)
	}
}

func (a *App) handleAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("c")
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		value := make(map[string]string)
		if err := a.cookie.Decode("c", cookie.Value, &value); err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		// loginID from cookie
		loginID, ok := value["loginID"]
		if !ok {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		userID, err := a.auth.UserID(loginID)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		user, err := a.repo.User(userID)
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		tutor, err := a.repo.TutorByUserID(userID)
		if err != nil && err != sql.ErrNoRows {
			log.Println("handleAuth: unable to fetch tutor:", err)
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}

		switch user.CookieConsent {
		case "yes":
			http.SetCookie(w, &http.Cookie{
				Name:    "cookie-consent",
				Value:   "yes",
				Path:    "/",
				Expires: time.Now().AddDate(2, 0, 0),
			})
		case "no":
			http.SetCookie(w, &http.Cookie{
				Name:    "cookie-consent",
				Value:   "no",
				Path:    "/",
				Expires: time.Now().AddDate(2, 0, 0),
			})
		default:
			// store user answer if answered before login,
			// this should only happen once per user.
			consent, err := r.Cookie("cookie-consent")
			if err == nil && (consent.Value == "yes" || consent.Value == "no") {
				if err := a.core.UpdateCookieConsent(user.ID, consent.Value); err != nil {
					log.Printf("handleAuth: unable to update cookie consent: %v", err)
				}
			}
		}

		tutorID := ""
		if tutor != nil {
			tutorID = tutor.ID
		}
		ctx := context.WithValue(r.Context(), model.ContextKeyTutorID, tutorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// func (a *App) handleSignup() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 	}
// }

// indexHandler handles the home page
func (a *App) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	page := a.view.Page("index.html")

	if err := a.view.Execute(w, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {

	page := a.view.Page("home.html")

	if err := a.view.Execute(w, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) handleSignup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			log.Println("get example")

			subjects, err := a.repo.Subjects()
			if err != nil {
				http.Error(w, "Unable to fetch subjects", http.StatusInternalServerError)
				return
			}

			levels, err := a.repo.Levels()
			if err != nil {
				http.Error(w, "Unable to fetch levels", http.StatusInternalServerError)
				return
			}

			locations, err := a.repo.Locations()
			if err != nil {
				http.Error(w, "Unable to fetch locations", http.StatusInternalServerError)
				return
			}

			page := a.view.Page("signup.html").
				Add("Subjects", subjects).
				Add("Levels", levels).
				Add("Locations", locations)

			if err := a.view.Execute(w, page); err != nil {
				log.Printf("handleSignup: %v", err)
			}
		case http.MethodPost:
			log.Println("post example")
			// maximum 10 MB files
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			phone := r.FormValue("phone")
			email := r.FormValue("email")
			location := r.FormValue("location")
			subjects := r.Form["subject"]
			levels := r.Form["level"]
			onlineLessons := r.FormValue("online-lessons")
			description := r.FormValue("description")
			smsOptIn := r.FormValue("sms-opt-in")
			bidToken := r.FormValue("bid-token")

			log.Println("phone", phone)
			log.Println("email", email)
			log.Println("location", location)
			log.Println("subject", subjects)
			log.Println("level", levels)
			log.Println("onlineLessons", onlineLessons)
			log.Println("description", description)
			log.Println("smsOptIn", smsOptIn)
			log.Println("bidToken", bidToken)

			claim := BidTokenClaim{}
			err := a.jwt.Validate(&claim, bidToken)
			if err != nil {
				http.Error(w, "invalid BankID token", http.StatusForbidden)
				return
			}
			firstname := claim.GivenName
			lastname := claim.Surname
			year, _ := strconv.Atoi(claim.SSN[:4])

			log.Println("firstname", firstname)
			log.Println("lastname", lastname)
			log.Println("year", year)

			unusedPassword := "r2UHtjbsZ5GKPEyYWdpBeg"
			token, err := a.auth.AddUser(firstname, lastname, email, phone, unusedPassword, &claim.SSN, smsOptIn == "on")
			if err != nil {
				log.Printf("auth: add user: %s, %v", email, err)
				http.Redirect(w, r, "/signup?error=adduser", http.StatusFound)
				return
			}

			tutorID, err := a.core.AddTutor(model.Tutor{
				UserID:        token.UserID,
				LocationID:    location,
				OnlineLessons: onlineLessons == "on",
				Description:   description,
			}, subjects, levels)

			if err != nil {
				log.Printf("core: add tutor: %s, %v", email, err)
				http.Redirect(w, r, "/signup?error=addtutor", http.StatusFound)
				return
			}

			file, _, err := r.FormFile("image")
			if err == nil {
				defer file.Close()
				if err := a.core.UpdateImage(file, tutorID); err != nil {
					log.Printf("signup: unable to update image: %v", err)
				}
			}

			// log.Println("signup attempt", token.UserEmail, token.ID, token.Value, add.Language)
			go a.email.SendSignupCode(token.UserEmail, token.ID, token.Value)

			http.Redirect(w, r, "/auth/confirm/"+token.ID, http.StatusFound)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// ParseStringID of http.Request.
func ParseStringID(r *http.Request) (string, error) {
	id := path.Base(r.URL.EscapedPath())
	if len(id) != 36 {
		return id, errors.New("invalid ID")
	}
	return id, nil
}

func (a *App) handleConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			id, err := ParseStringID(r)
			if err != nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			token, err := a.auth.TokenByID(id)
			if err != nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			if token.Name != "CONFIRMATION" {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			if !token.Valid() {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			page := a.view.
				Page("signup-confirm.html").
				Add("ID", token.ID)

			if err := a.view.Execute(w, page); err != nil {
				log.Printf("handleConfirm: %v", err)
			}
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			id := r.FormValue("id")
			code := r.FormValue("code")

			if len(code) != 4 {
				http.Redirect(w, r, "/auth/confirm/"+id, http.StatusFound)
				return
			}

			token, err := a.auth.TokenByID(id)
			if err != nil {
				http.Redirect(w, r, "/auth/confirm/"+id, http.StatusFound)
				return
			}

			if err := a.auth.Confirm(token, code); err != nil {
				log.Println(err)
				http.Redirect(w, r, "/auth/confirm/"+id, http.StatusFound)
				return
			}

			// user, err := a.auth.UserByID(profile.AccountID)
			// if err != nil {
			// 	log.Println(err)
			// 	http.Redirect(w, r, "/auth/confirm/"+id, http.StatusFound)
			// }

			// go a.email.SendWelcome(&mailer.WelcomeMessage{
			// 	Role:  profile.ProfileRole,
			// 	Email: user.Email,
			// })

			http.Redirect(w, r, "/auth/login", http.StatusFound)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed)
		}
	}
}
