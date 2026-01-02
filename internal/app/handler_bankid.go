package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"upforschool/internal/bankid"
	"upforschool/internal/jwt"

	"github.com/skip2/go-qrcode"
)

var (
	bid          *bankid.BankID
	authSessions = make(map[string]authSession)
	mu           sync.Mutex
)

type authSession struct {
	QRStartToken   string
	QRStartSecret  string
	OrderTime      int64
	EndUserIP      string
	AutoStartToken string
	CollectResult  bankid.CollectResponse
	Canceled       bool
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8008")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func startAuthHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		EndUserIP string `json:"endUserIp"`
		SigningID string `json:"signingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	authResponse, err := bid.Auth(bankid.AuthRequest{
		PersonalNumber: "", // Optional, can be empty
		EndUserIP:      requestData.EndUserIP,
	})
	if err != nil {
		log.Println("Error starting auth:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderTime := time.Now().Unix()
	qrData := generateQRData(authResponse.QrStartToken, authResponse.QrStartSecret, orderTime)

	mu.Lock()
	authSessions[authResponse.OrderRef] = authSession{
		QRStartToken:   authResponse.QrStartToken,
		QRStartSecret:  authResponse.QrStartSecret,
		OrderTime:      orderTime,
		EndUserIP:      requestData.EndUserIP,
		AutoStartToken: authResponse.AutoStartToken,
	}
	mu.Unlock()

	response := map[string]string{
		"orderRef":       authResponse.OrderRef,
		"qrData":         qrData,
		"autoStartToken": authResponse.AutoStartToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Poll for the result to collect
	go pollForResult(authResponse.OrderRef)
}

func pollForResult(orderRef string) {
	for i := 0; i < 14; i++ {
		time.Sleep(2 * time.Second)
		mu.Lock()
		session, exists := authSessions[orderRef]
		mu.Unlock()
		if !exists || session.Canceled {
			return
		}

		collectResponse, err := bid.Collect(bankid.CollectRequest{OrderRef: orderRef})
		if err != nil {
			log.Println("Error collecting result:", err)
			continue
		}

		mu.Lock()
		session.CollectResult = collectResponse
		authSessions[orderRef] = session
		mu.Unlock()

		if collectResponse.Status != "pending" {
			log.Println("Final status:", collectResponse.Status)
			return
		}
	}

	mu.Lock()
	session, exists := authSessions[orderRef]
	if exists {
		session.Canceled = true
		authSessions[orderRef] = session
	}
	mu.Unlock()

}

func generateQRData(qrStartToken, qrStartSecret string, orderTime int64) string {
	qrTime := strconv.FormatInt(time.Now().Unix()-orderTime, 10)
	h := hmac.New(sha256.New, []byte(qrStartSecret))
	h.Write([]byte(qrTime))
	qrAuthCode := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("bankid.%s.%s.%s", qrStartToken, qrTime, qrAuthCode)
}

func generateQRHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		OrderRef string `json:"orderRef"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	session, exists := authSessions[requestData.OrderRef]
	mu.Unlock()
	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if session.Canceled {
		w.WriteHeader(http.StatusNoContent)
	} else if session.CollectResult.Status == "complete" {
		w.WriteHeader(http.StatusOK)
	} else if session.CollectResult.Status == "failed" {
		w.WriteHeader(http.StatusUnauthorized)
		// }else if session.S
	} else {
		w.Header().Set("Content-Type", "image/png")
		qrData := generateQRData(session.QRStartToken, session.QRStartSecret, session.OrderTime)
		qrCode, err := qrcode.New(qrData, qrcode.Medium)
		qrCode.DisableBorder = true
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		png.Encode(w, qrCode.Image(150))
	}
}

func (a *App) collectAuthHandler(w http.ResponseWriter, r *http.Request) {
	var req bankid.CollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	session, exists := authSessions[req.OrderRef]
	mu.Unlock()
	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	collectResponse := session.CollectResult
	if collectResponse.Status == "failed" {
		log.Println("User authentication failed:", collectResponse.HintCode)
		// Handle successful sign-in securely
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if collectResponse.Status == "complete" {
		log.Println("User authenticated successfully:", collectResponse.CompletionData.User.PersonalNumber)

		if collectResponse.Status == "failed" {
			switch collectResponse.HintCode {
			case "expiredTransaction":
				log.Println("message RFA8")
			case "certificateErr":
				log.Println("message RFA16")
			case "userCancel":
				log.Println("message RFA6")
			case "cancelled":
				log.Println("message RFA3")
			case "startFailed":
				log.Println("message RFA17")
			}

			DefaultErrorWriter(ErrBadRequest, collectResponse.HintCode, w, r)
			return
		}

		ssn, _ := cleanSSN(collectResponse.CompletionData.User.PersonalNumber)

		bidTokenClaim := BidTokenClaim{
			SSN:       ssn,
			GivenName: collectResponse.CompletionData.User.GivenName,
			Surname:   collectResponse.CompletionData.User.Surname,
		}
		// create token
		token, err := a.jwt.NewToken(&bidTokenClaim, time.Minute*30)
		if err != nil {
			http.Error(w, "Bad request, failed to tokenize bid responses", http.StatusBadRequest)
			return
		}

		userExists := false
		user, err := a.auth.UserBySSN(ssn)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Println("Failed to check user existence:", err)
			http.Error(w, "Bad request, failed to check user existence", http.StatusInternalServerError)
			return
		} else if user != nil && user.ID != "" {
			userExists = true
		}

		// send token to user
		DefaultWriter(struct {
			SSN       string `json:"ssn"`
			GivenName string `json:"givenName"`
			SurName   string `json:"surName"`
			Token     string `json:"token"`
			UserExiss bool   `json:"userExists"`
		}{
			SSN:       ssn,
			GivenName: collectResponse.CompletionData.User.GivenName,
			SurName:   collectResponse.CompletionData.User.Surname,
			Token:     *token,
			UserExiss: userExists,
		}, nil, w, r)

		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func cancelAuthHandler(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		OrderRef string `json:"orderRef"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mu.Lock()
	session, exists := authSessions[requestData.OrderRef]
	if exists {
		session.Canceled = true
		authSessions[requestData.OrderRef] = session
	}
	mu.Unlock()

	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func cleanSSN(ssn string) (string, error) {
	ssn = strings.ReplaceAll(ssn, "-", "")
	ssn = strings.ReplaceAll(ssn, " ", "")
	_, err := strconv.Atoi(ssn)
	if err != nil {
		return "", fmt.Errorf("invalid ssn: %s", ssn)
	}

	if len(ssn) == 10 {

		shortyear, err := strconv.Atoi(ssn[:2])
		if err != nil {
			return "", err
		}
		if shortyear > 20 {
			ssn = "19" + ssn
		} else {
			ssn = "20" + ssn
		}
	}
	return ssn, nil
}

// ExtendSwedishSSN extends a Swedish SSN from YYMMDD-NNNN to YYYYMMDD-NNNN format.
func ExtendSwedishSSN(ssn string) string {
	if len(ssn) == 12 {
		return ssn
	}

	if len(ssn) == 10 {
		return ssn
	}

	// Extract the year, month, and day
	shortYearStr := ssn[:2]
	monthDay := ssn[2:6]

	shortYear, err := strconv.Atoi(shortYearStr)
	if err != nil {
		return ssn
	}

	// Get the current year and determine the century
	currentYear := time.Now().Year()
	currentShortYear := currentYear % 100
	century := currentYear / 100

	// If the short year is greater than the current year's last two digits, assume it's from the previous century
	if shortYear > currentShortYear {
		century -= 1
	}

	// Create the full year and return the extended SSN
	fullYear := strconv.Itoa(century*100 + shortYear)
	extendedSSN := fullYear + monthDay + ssn[6:] // YYYYMMDD-NNNN

	return extendedSSN
}

type BidTokenClaim struct {
	jwt.BaseClaim
	SSN       string `json:"ssn"`
	GivenName string `json:"givenName"`
	Surname   string `json:"surname"`
}

func (a *App) handleNewBidToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodPost:

			// parse json body
			request := struct {
				SSN      string `json:"ssn"`
				IP       string `json:"ip"`
				Distinct bool   `json:"distinct"`
			}{}
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&request)
			if err != nil {
				http.Error(w, fmt.Errorf("unable to decode body; %w", err).Error(), http.StatusBadRequest)
				return
			}

			ssn, err := cleanSSN(request.SSN)
			if err != nil {
				http.Error(w, "Bad bankid request "+err.Error(), http.StatusBadRequest)
				return
			}

			if request.Distinct {
				usr, err := a.auth.UserBySSN(ExtendSwedishSSN(ssn))
				if err == nil && usr != nil {
					log.Println("User with SSN already exists", usr)
					http.Error(w, "User with SSN already exists", http.StatusConflict)
					return
				} else if err != nil && err != sql.ErrNoRows {
					http.Error(w, "Failed to find user with ssn "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

			// call bankid
			bidResponse, err := a.bankid.TryAuth(bankid.AuthRequest{
				PersonalNumber: ssn,
				EndUserIP:      request.IP,
			})
			if err != nil {
				http.Error(w, "Bad bankid request "+err.Error(), http.StatusBadRequest)
				return
			}

			if bidResponse.Status == "failed" {
				switch bidResponse.HintCode {
				case "expiredTransaction":
					log.Println("message RFA8")
				case "certificateErr":
					log.Println("message RFA16")
				case "userCancel":
					log.Println("message RFA6")
				case "cancelled":
					log.Println("message RFA3")
				case "startFailed":
					log.Println("message RFA17")
				}

				DefaultErrorWriter(ErrBadRequest, bidResponse.HintCode, w, r)
				return
			}

			bidTokenClaim := BidTokenClaim{
				SSN:       ssn,
				GivenName: bidResponse.CompletionData.User.GivenName,
				Surname:   bidResponse.CompletionData.User.Surname,
			}
			// create token
			token, err := a.jwt.NewToken(&bidTokenClaim, time.Minute*30)
			if err != nil {
				http.Error(w, "Bad request, failed to tokenize bid responsesx3", http.StatusBadRequest)
				return
			}

			// send token to user
			DefaultWriter(struct {
				SSN       string `json:"ssn"`
				GivenName string `json:"givenName"`
				SurName   string `json:"surName"`
				Token     string `json:"token"`
			}{
				SSN:       ssn,
				GivenName: bidResponse.CompletionData.User.GivenName,
				SurName:   bidResponse.CompletionData.User.Surname,
				Token:     *token,
			}, nil, w, r)
		}
	}
}

var (
	ErrBadRequest error = errors.New("request parameters are not valid")
	ErrNotFound   error = errors.New("resource not found")
	ErrForbidden  error = errors.New("not authorized, resource is restricted")
	ErrConflict   error = errors.New("request could not be completed due to resource conflict")
)

// not a middlware, but kind of related
func DefaultWriter(response interface{}, err error, w http.ResponseWriter, r *http.Request) {

	if err != nil {
		DefaultErrorWriter(err, err.Error(), w, r)
		return
	}
	w.WriteHeader(http.StatusOK)

	if response != nil {
		if d, ok := response.([]byte); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(d)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// not a middlware, but kind of related
func DefaultErrorWriter(err error, msg string, w http.ResponseWriter, r *http.Request) {
	if errors.Is(err, ErrForbidden) {
		w.WriteHeader(http.StatusForbidden)
	} else if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
	} else if errors.Is(err, ErrBadRequest) {
		w.WriteHeader(http.StatusBadRequest)
	} else if errors.Is(err, ErrConflict) {
		w.WriteHeader(http.StatusConflict)
	} else {
		log.Println("internal error:", err, "message:", msg, printCaller())
		w.WriteHeader(http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": msg,
	})
}

func printCaller() string {
	pc, file, line, ok := runtime.Caller(2) // 1 means get the caller of this function
	if !ok {
		return "UNKNOWN CALLER"
	}

	// Get the function name
	fn := runtime.FuncForPC(pc)
	functionName := "unknown"
	if fn != nil {
		functionName = fn.Name()
	}

	return fmt.Sprintf("Caller: %s File: %s:%d", functionName, file, line)
}
