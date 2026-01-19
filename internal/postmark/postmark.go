package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Service struct {
	apiToken string
}

func NewService(apiToken string) *Service {
	return &Service{
		apiToken: apiToken,
	}
}

var DefaultParams = map[string]string{
	"product_url":  "upforschool.se",
	"product_name": "Up For School",
	"action_url":   "upforschool.se",
	"company_name": "Up For School AB",
	// "operating_system": "operating_system_Value",
	// "browser_name":     "browser_name_Value",
	// "support_url":      "support_url_Value",
	// "company_address":  "company_address_Value",
}

func (s *Service) SendActivationEmail(name, toEmail, tokenID, tokenValue string) {

	log.Println("activation code:", toEmail, tokenValue)
	url := "https://api.postmarkapp.com/email/withTemplate"

	// Request payload
	templateModel := map[string]any{
		"activationID":   tokenID,
		"activationCode": tokenValue,
		"name":           name,
	}

	for k, v := range DefaultParams {
		templateModel[k] = v
	}

	payload := map[string]any{
		"From":          "no-reply@upforschool.se",
		"To":            toEmail,
		"TemplateAlias": "account-activation",
		"TemplateModel": templateModel,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", s.apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)
}
