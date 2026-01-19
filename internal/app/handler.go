package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"upforschool/internal/model"
	"upforschool/internal/upforauth"
	"upforschool/internal/viewer"
)

func (a *App) handleIsLoggedIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user, err := a.loggedInUser(r)
		if err != nil {
			// not logged in, handle next.
			next.ServeHTTP(w, r)
			return
		}

		if user != nil {
			// logged in with profile
			http.Redirect(w, r, "/home", http.StatusFound)
		}
	}
}

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

		var tutor model.Tutor
		t, err := a.repo.TutorByUserID(userID)
		if err != nil && err != sql.ErrNoRows {
			log.Println("handleAuth: unable to fetch tutor:", err)
			http.Redirect(w, r, "/auth/login", http.StatusFound)
			return
		}
		if t != nil {
			tutor = *t
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

		// tutorID := ""
		// if tutor != nil {
		// 	tutorID = tutor.ID
		// }

		ctx := context.WithValue(r.Context(), model.ContextKeyProfile, model.Profile{
			User:    *user,
			IsTutor: tutor.ID != "",
		})
		ctx = context.WithValue(ctx, model.ContextKeyTutor, tutor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// func (a *App) handleSignup() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 	}
// }

func (a *App) profile(r *http.Request) model.Profile {
	return r.Context().Value(model.ContextKeyProfile).(model.Profile)
}

func (a *App) tutor(r *http.Request) model.Tutor {
	return r.Context().Value(model.ContextKeyTutor).(model.Tutor)
}

func (a *App) activeTutor(r *http.Request) bool {
	return a.profile(r).User.ActiveRole == string(model.ActiveRoleTutor)
}

func (a *App) page(name string) http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := a.view.Page(name)

		if err := a.view.Execute(w, page); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {

	LessonRequests, err := a.repo.SentLessonRequests(a.profile(r).User.ID)
	if err != nil {
		log.Println("failed to get lesson requests:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var page *viewer.Page
	if a.activeTutor(r) {
		page = a.view.Page("home-tutor.html")
	} else {
		page = a.view.Page("home-student.html")
	}

	page.Add("Profile", a.profile(r)).
		Add("LessonRequests", LessonRequests)

	if err := a.view.Execute(w, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

func (a *App) loggedInUser(r *http.Request) (*upforauth.User, error) {
	cookie, err := r.Cookie("c")
	if err != nil {
		return nil, err
	}

	value := make(map[string]string)
	if err := a.cookie.Decode("c", cookie.Value, &value); err != nil {
		return nil, err
	}

	// loginID from cookie
	loginID, ok := value["loginID"]
	if !ok {
		return nil, err
	}

	return a.auth.User(loginID)
}

func (a *App) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			page := a.view.Page("signin.html").Add("Error", r.URL.Query().Get("error"))
			if err := a.view.Execute(w, page); err != nil {
				log.Printf("handleLogin: %v", err)
			}
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Redirect(w, r, "/auth/login?error=form", http.StatusFound)
				return
			}

			bidToken := r.PostFormValue("bid-token")
			username := strings.ToLower(r.PostFormValue("username"))
			password := r.PostFormValue("password")

			var loginID string
			var err error
			if bidToken != "" {
				claim := BidTokenClaim{}
				err = a.jwt.Validate(&claim, bidToken)
				if err != nil {
					http.Error(w, "invalid BankID token", http.StatusForbidden)
					return
				}

				loginID, err = a.auth.LoginSSN(claim.SSN)
				if err != nil {
					http.Redirect(w, r, "/auth/login?error=login", http.StatusFound)
					return
				}
			} else {
				user, err := a.auth.UserByEmail(username)
				if err != nil {
					http.Redirect(w, r, "/auth/login?error=login", http.StatusFound)
				}

				tutorProfile, err := a.repo.TutorByUserID(user.ID)
				if err != nil && err != sql.ErrNoRows {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				log.Println("found tutor profile:", tutorProfile)

				// User has profile that requires BankID
				if tutorProfile != nil || user.SSN != nil && *user.SSN != "" {
					http.Redirect(w, r, "/auth/login?error=login", http.StatusFound)
					return
				}
				log.Println("---> XX")

				loginID, err = a.auth.Login(username, password)
				if err != nil {
					http.Redirect(w, r, "/auth/login?error=login", http.StatusFound)
					return
				}
			}

			value := map[string]string{
				"loginID": loginID,
			}

			encoded, err := a.cookie.Encode("c", value)
			if err != nil {
				log.Println("x-x-x 5", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
				return
			}

			cookie := &http.Cookie{
				Name:     "c",
				Value:    encoded,
				Path:     "/",
				Secure:   false,
				HttpOnly: true,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusFound)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed)
		}
	}
}

func (a *App) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("c")
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		value := make(map[string]string)
		if err := a.cookie.Decode("c", cookie.Value, &value); err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// loginID from cookie
		loginID, ok := value["loginID"]
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if err := a.auth.Logout(loginID); err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		cookie = &http.Cookie{
			Name:     "c",
			Value:    "",
			Path:     "/",
			Secure:   false,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (a *App) handleSignupStudent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
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

			showAllLessons := r.URL.Query().Get("lessons") == "all"

			page := a.view.Page("signup-student.html").
				Add("Subjects", subjects).
				Add("Levels", levels).
				Add("Locations", locations).
				Add("AllLessons", showAllLessons)

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

			firstname := r.FormValue("firstname")
			lastname := r.FormValue("lastname")
			phone := r.FormValue("phone")
			email := r.FormValue("email")
			location := r.FormValue("location")
			smsOptIn := r.FormValue("sms-opt-in")
			password := r.FormValue("password")

			log.Println("password", password)
			log.Println("phone", phone)
			log.Println("email", email)
			log.Println("location", location)
			log.Println("smsOptIn", smsOptIn)

			log.Println("firstname", "firstname")
			log.Println("lastname", "lastname")
			log.Println("year", "year")

			token, err := a.auth.AddUser(firstname, lastname, email, phone, password, nil, smsOptIn == "on", model.ActiveRoleStudent)
			if err != nil {
				log.Printf("auth: add user: %s, %v", email, err)
				http.Redirect(w, r, "/signup?error=adduser", http.StatusFound)
				return
			}

			// log.Println("signup attempt", token.UserEmail, token.ID, token.Value, add.Language)
			go a.email.SendActivationEmail(firstname, token.UserEmail, token.ID, token.Value)

			http.Redirect(w, r, "/auth/confirm/"+token.ID, http.StatusFound)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func (a *App) handleSignupTutor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
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

			page := a.view.Page("signup-tutor.html").
				Add("Subjects", subjects).
				Add("Levels", levels).
				Add("Locations", locations)

			if err := a.view.Execute(w, page); err != nil {
				log.Printf("handleSignup: %v", err)
			}
		case http.MethodPost:
			log.Println("tutor post")
			// maximum 10 MB files
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			phone := r.FormValue("phone")
			email := r.FormValue("email")
			locations := r.Form["locations"]
			subjects := r.Form["subject"]
			levels := r.Form["level"]
			onlineLessons := r.FormValue("online-lessons")
			description := r.FormValue("description")
			smsOptIn := r.FormValue("sms-opt-in")
			bidToken := r.FormValue("bid-token")
			alias := r.FormValue("alias")

			log.Println("phone", phone)
			log.Println("email", email)
			log.Println("locations", locations)
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
			token, err := a.auth.AddUser(firstname, lastname, email, phone, unusedPassword, &claim.SSN, smsOptIn == "on", model.ActiveRoleTutor)
			if err != nil {
				log.Printf("auth: add user: %s, %v", email, err)
				http.Redirect(w, r, "/signup?error=adduser", http.StatusFound)
				return
			}

			tutorID, err := a.core.AddTutor(model.Tutor{
				UserID:        token.UserID,
				Alias:         alias,
				OnlineLessons: onlineLessons == "on",
				Bio:           description,
			}, locations, subjects, levels)

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

			go a.email.SendActivationEmail(firstname, token.UserEmail, token.ID, token.Value)

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

func (a *App) handleSwitchRole(w http.ResponseWriter, r *http.Request) {

	profile := a.profile(r)

	targetRole := model.ActiveRole(r.URL.Query().Get("role"))

	if targetRole != model.ActiveRoleStudent &&
		targetRole != model.ActiveRoleTutor {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	err := a.core.SwitchRole(profile.User.ID, targetRole)
	if err != nil {
		log.Println("handleSwitchRole: unable to switch role:", err)
		http.Error(w, "unable to switch role", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)

}

func (a *App) handleProfile(w http.ResponseWriter, r *http.Request) {

	profile := a.profile(r)
	tutor := a.tutor(r)

	a.repo.TutorByUserID(profile.User.ID)

	page := a.view.
		Page("profile.html").
		Add("ActiveRole", profile.User.ActiveRole).
		Add("Profile", profile).
		Add("Tutor", tutor)

	if err := a.view.Execute(w, page); err != nil {
		log.Printf("handleConfirm: %v", err)
	}
}

func (a *App) handleNewLesson(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
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

		page := a.view.
			Page("lesson-new.html").
			Add("Subjects", subjects).
			Add("Levels", levels).
			Add("Locations", locations)

		if err := a.view.Execute(w, page); err != nil {
			log.Printf("handleConfirm: %v", err)
		}

	case http.MethodPost:

		log.Println("handleNewLesson: post")
		var req model.LessonRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		lessonID, err := a.core.AddLesson(a.profile(r).User.ID, req)
		if err != nil {
			log.Println("handleNewLesson: unable to add lesson:", err)
			http.Error(w, "unable to add lesson", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"lessonID": lessonID,
			"status":   "ok",
		})

	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (a *App) handleGetTutors(w http.ResponseWriter, r *http.Request) {
	subjectID := r.URL.Query().Get("subject")
	levelID := r.URL.Query().Get("level")
	locationID := r.URL.Query().Get("location")

	onlineLessonsBool := locationID == "online"
	subjectIDInt, _ := strconv.Atoi(subjectID)
	levelIDInt, _ := strconv.Atoi(levelID)
	locationIDInt, _ := strconv.Atoi(locationID)

	log.Println("Getting tutors")

	tutors, err := a.repo.Tutors(onlineLessonsBool, int(locationIDInt), int(subjectIDInt), int(levelIDInt))
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to fetch tutors", http.StatusInternalServerError)
		return
	}

	log.Println("tutors found:", len(tutors))

	page := a.view.
		Page("tutors-partial.html").
		Add("Tutors", tutors)

	if err := a.view.Execute(w, page); err != nil {
		log.Printf("handleGetTutors: %v", err)
	}

}

func (a *App) handleListLessons(w http.ResponseWriter, r *http.Request) {

	var err error
	var lessonRequests []model.LessonView
	if a.activeTutor(r) {
		lessonRequests, err = a.repo.ReceivedLessonRequests(a.tutor(r).ID)
		if err != nil {
			log.Println("failed to get lesson requests:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		lessonRequests, err = a.repo.SentLessonRequests(a.profile(r).User.ID)
		if err != nil {
			log.Println("failed to get lesson requests:", err)
		}
	}

	// Remove deleted lessons unless "lessons=all" is set
	// also remove old accepted lessons
	showAllLessons := r.URL.Query().Get("lessons") == "all"
	if !showAllLessons {
		filteredLessons := []model.LessonView{}
		for _, lr := range lessonRequests {
			if lr.DeletedAt.Valid ||
				lr.AcceptedAt.Valid && lr.AcceptedAt.Time.Add(7*24*time.Hour).Before(time.Now()) { // accepted more than 7 days ago
				continue
			}
			filteredLessons = append(filteredLessons, lr)
		}
		lessonRequests = filteredLessons
	}

	log.Println("AT", a.activeTutor(r))
	page := a.view.
		Page("home-lessons-partial.html").
		Add("IsTutor", a.activeTutor(r)).
		Add("Lessons", lessonRequests).
		Add("ShowAllLessons", showAllLessons)

	if err := a.view.Execute(w, page); err != nil {
		log.Printf("handleListLessons: %v", err)
	}

}

func (a *App) handleLessonAccept(w http.ResponseWriter, r *http.Request) {

	if !a.activeTutor(r) {
		http.Error(w, "only tutors can accept lessons", http.StatusForbidden)
	}

	lessonID := r.URL.Query().Get("lesson_id")
	l, err := a.repo.GetLesson(lessonID)
	if err != nil {
		log.Println("handleLessonAccept: unable to fetch lesson:", err)
		http.Error(w, "unable to fetch lesson", http.StatusInternalServerError)
		return
	}

	tutor := a.tutor(r)
	log.Println("rs", tutor.OnlineLessons)
	log.Println(
		l.OnlineLesson, l.LocationID, l.SubjectID, l.LevelID,
	)

	if !tutor.MeetsRequirements(l.OnlineLesson, l.LocationID, l.SubjectID, l.LevelID) {
		http.Error(w, "tutor does not meet lesson requirements", http.StatusForbidden)
		return
	}

	_, err = a.core.AcceptLesson(lessonID, tutor.ID)
	if err != nil {
		log.Println("handleLessonAccept: unable to accept lesson:", err)
		http.Error(w, "unable to accept lesson", http.StatusInternalServerError)
		return
	}

	log.Println("lesson accepted", lessonID, "by tutor", tutor.ID)
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (a *App) handleLessonDelete(w http.ResponseWriter, r *http.Request) {
	user := a.profile(r)

	lessonID := r.URL.Query().Get("lesson_id")
	l, err := a.repo.GetLesson(lessonID)
	if err != nil {
		log.Println("handleLessonDelete: unable to fetch lesson:", err)
		http.Error(w, "unable to fetch lesson", http.StatusInternalServerError)
		return
	}

	if l.StudentID != user.User.ID {
		http.Error(w, "only the student who created the lesson can delete it", http.StatusForbidden)
		return
	}

	err = a.core.DeleteLesson(lessonID)
	if err != nil {
		log.Println("handleLessonDelete: unable to delete lesson:", err)
		http.Error(w, "unable to delete lesson", http.StatusInternalServerError)
		return
	}

	log.Println("lesson deleted", lessonID, "by student", user.User.ID)
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (a *App) handleGetTutorSummary(w http.ResponseWriter, r *http.Request) {
	tutorID := r.URL.Query().Get("tutor_id")

	tutor, err := a.repo.Tutor(tutorID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to fetch tutor", http.StatusInternalServerError)
		return
	}

	user, err := a.repo.User(tutor.UserID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to fetch tutor user", http.StatusInternalServerError)
		return
	}

	tutorView := model.TutorView{
		ID:            tutorID,
		UserID:        user.ID,
		Image:         tutor.Image,
		OnlineLessons: tutor.OnlineLessons,
		Bio:           tutor.Bio,
		CreatedAt:     tutor.CreatedAt,
		UpdatedAt:     tutor.UpdatedAt,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Email:         user.Email,
		Phone:         user.Phone,
		SMSOptIn:      user.SMSOptIn,
		Status:        user.Status,
	}

	page := a.view.
		Page("tutor-summary-partial.html").
		Add("Tutor", tutorView)

	if err := a.view.Execute(w, page); err != nil {
		log.Printf("handleGetTutors: %v", err)
	}

}

func (a *App) handleProfilesEdit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			p := a.profile(r)
			t := a.tutor(r)

			subjects, err := a.repo.Subjects()
			if err != nil {
				http.Error(w, "Unable to fetch subjects", http.StatusInternalServerError)
				return
			}
			for i := range subjects {
				for _, ts := range t.Subjects {
					if subjects[i].ID == ts.ID {
						subjects[i].Selected = true
					}
				}
			}

			levels, err := a.repo.Levels()
			if err != nil {
				http.Error(w, "Unable to fetch levels", http.StatusInternalServerError)
				return
			}
			for i := range levels {
				for _, ts := range t.Levels {
					if levels[i].ID == ts.ID {
						levels[i].Selected = true
					}
				}
			}

			locations, err := a.repo.Locations()
			if err != nil {
				http.Error(w, "Unable to fetch locations", http.StatusInternalServerError)
				return
			}
			for i := range locations {
				for _, ts := range t.Locations {
					if locations[i].ID == ts.ID {
						locations[i].Selected = true
					}
				}
			}

			page := a.view.Page("profiles-edit.html").
				Add("Profile", p).
				Add("Tutor", t).
				Add("Subjects", subjects).
				Add("Levels", levels).
				Add("Locations", locations)

			if err := a.view.Execute(w, page); err != nil {
				log.Printf("handleProfilesEdit: %v", err)
			}
		}
		// case http.MethodPost:
		// 	profile := a.profile(r)

		// 	if err := r.ParseForm(); err != nil {
		// 		http.Error(w, err.Error(), http.StatusInternalServerError)
		// 		return
		// 	}

		// 	// individual instructors cannot change their year, firstname or lastname.
		// 	if profile.ProfileRole == "INSTRUCTOR" && !profile.ProfileIsOrg {
		// 		r.Form.Set("year", strconv.Itoa(int(profile.ProfileYear)))
		// 		r.Form.Set("firstname", profile.ProfileFirstname)
		// 		r.Form.Set("lastname", profile.ProfileLastname)
		// 	}

		// 	update, err := service.ParseUpdateProfileRequest(r.Form)
		// 	if err != nil {
		// 		http.Error(w, err.Error(), http.StatusInternalServerError)
		// 		return
		// 	}

		// 	if profile.ProfileID != update.ID {
		// 		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		// 		return
		// 	}

		// 	if err := a.svc.UpdateProfile(update); err != nil {
		// 		http.Error(w, err.Error(), http.StatusInternalServerError)
		// 		return
		// 	}

		// 	http.Redirect(w, r, "/profiles", http.StatusFound)
		// default:
		// 	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		// 	return
		// }
	}
}
