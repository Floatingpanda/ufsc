package worldline

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Currency string

const (
	SEK = "SEK"
	EUR = "EUR"
	USD = "USD"
)

type Worldline struct {
	apiURL   string
	appURL   string
	username string
	password string
	md5Key   string
	client   *http.Client
}

type OrderCreate struct {
	Currency    Currency
	OrderID     string
	Amount      int64
	AcceptURL   string
	CancelURL   string
	CallbackURL string
}

type OrderResponse struct {
	URL string
}

// amounts in ören.
type order struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type callback struct {
	URL string `json:"url"`
}

type url struct {
	Accept    string     `json:"accept"`
	Callbacks []callback `json:"callbacks"`
	Cancel    string     `json:"cancel"`
}

type paymentwindow struct {
	PaymentMethods []paymentmethod `json:"paymentmethods"`
}

type paymentmethod struct {
	ID     string `json:"id"`
	Action string `json:"action"`
}

type data struct {
	Order         order         `json:"order"`
	URL           url           `json:"url"`
	PaymentWindow paymentwindow `json:"paymentwindow"`
}

type session struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

func (w *Worldline) NewSubscription(orderID int64, amount float64, currency Currency) OrderCreate {
	amountInCents := int64(amount * 100)

	// subscriptions are prefixed with "S".
	id := fmt.Sprintf("S%d", orderID)
	return OrderCreate{
		OrderID:     id,
		Currency:    currency,
		Amount:      amountInCents,
		AcceptURL:   w.appURL + "/organizations/subscriptions/accept",
		CancelURL:   w.appURL + "/organizations/subscriptions",
		CallbackURL: w.appURL + "/organizations/subscriptions/callback",
	}
}

func (w *Worldline) NewOrder(orderID int64, amount float64, currency Currency) OrderCreate {
	amountInCents := int64(amount * 100)

	// orders are prefixed with "O".
	id := fmt.Sprintf("O%d", orderID)
	return OrderCreate{
		OrderID:     id,
		Currency:    currency,
		Amount:      amountInCents,
		AcceptURL:   w.appURL + "/games/orders/accept",
		CancelURL:   w.appURL + "/games/orders",
		CallbackURL: w.appURL + "/games/orders/callback",
	}
}

// CreateOrder. A worldline/bambora checkout session is valid for one (1) hour.
func (w *Worldline) CreateOrder(o OrderCreate) (*OrderResponse, error) {
	data := data{
		Order: order{
			ID:       o.OrderID,
			Amount:   o.Amount,
			Currency: string(o.Currency),
		},
		URL: url{
			Accept: o.AcceptURL,
			Cancel: o.CancelURL,
			Callbacks: []callback{
				{
					URL: o.CallbackURL,
				},
			},
		},
		PaymentWindow: paymentwindow{
			[]paymentmethod{
				{
					ID:     "paymentcard",
					Action: "include",
				},
				{
					ID:     "swish",
					Action: "include",
				},
			},
		},
	}

	var buff bytes.Buffer
	if err := json.NewEncoder(&buff).Encode(&data); err != nil {
		return nil, err
	}
	fmt.Print(buff.String())

	req, err := http.NewRequest(http.MethodPost, w.apiURL, &buff)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(w.username, w.password)
	req.Header.Set("Content-Type", "application/json")

	res, err := w.client.Do(req)
	if err != nil {
		return nil, err

	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var unexpectedResponse bytes.Buffer

		_, err := io.Copy(&unexpectedResponse, res.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(unexpectedResponse.String())
	}

	var s session
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return nil, err
	}

	return &OrderResponse{s.URL}, nil
}

// Validate query by concatenating all params (in order, except hash param)
// with md5key and compare with hash param.
// md5(values + md5key) == hash
func (w *Worldline) Validate(rawQuery string) bool {
	params := strings.Split(rawQuery, "&")

	var value string
	var hash string

	for _, p := range params {
		before, after, _ := strings.Cut(p, "=")

		if before == "hash" {
			hash = after
			continue
		}

		value += after
	}

	hashed := md5Hash([]byte(value + w.md5Key))
	return hash == hashed
}

func md5Hash(b []byte) string {
	hash := md5.Sum(b)
	return hex.EncodeToString(hash[:])
}

func New(appURL, mechant, username, password, md5key string) *Worldline {
	apiURL := "https://api.v1.checkout.bambora.com/sessions"

	return &Worldline{
		apiURL:   apiURL,
		appURL:   appURL,
		username: username + "@" + mechant,
		password: password,
		md5Key:   md5key,
		client: &http.Client{
			Timeout: time.Second * 60,
		},
	}
}
