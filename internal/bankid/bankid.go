package bankid

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

type Requirement struct {
	CardReader             string `json:"cardReader,omitempty"`
	CertificatePolicies    string `json:"certificatePolicies,omitempty"`
	IssuerCn               string `json:"issuerCn,omitempty"`
	AutoStartTokenRequired string `json:"autoStartTokenRequired,omitempty"`
	AllowFingerprint       bool   `json:"allowFingerprint,omitempty"`
	TokenStartRequired     bool   `json:"tokenStartRequired,omitempty"`
}

type AuthRequest struct {
	PersonalNumber string      `json:"personalNumber,omitempty"`
	EndUserIP      string      `json:"endUserIp"`
	Requirement    Requirement `json:"requirement,omitempty"` // Optional settings for the authentication process
}
type AuthResponse struct {
	OrderRef       string `json:"orderRef"`
	AutoStartToken string `json:"autoStartToken"`
	QrStartToken   string `json:"qrStartToken"`
	QrStartSecret  string `json:"qrStartSecret"`
}

type CollectRequest struct {
	OrderRef string `json:"orderRef"`
}

type CompletionDataUser struct {
	PersonalNumber string `json:"personalNumber"`
	Name           string `json:"name"`
	GivenName      string `json:"givenName"`
	Surname        string `json:"surname"`
}

type CompletionDataDevice struct {
	IPAddress string `json:"ipAddress"`
}

type CompletionDataCert struct {
	NotBefore string `json:"notBefore"`
	NotAfter  string `json:"notAfter"`
}

type CompletionData struct {
	User         CompletionDataUser   `json:"user"`
	Device       CompletionDataDevice `json:"device"`
	Cert         CompletionDataCert   `json:"cert"`
	Signature    string               `json:"signature"`
	OcspResponse string               `json:"ocspResponse"`
}

type CollectResponse struct {
	OrderRef       string         `json:"orderRef"`
	Status         string         `json:"status"` // pending, failed, complete
	HintCode       string         `json:"hintCode"`
	CompletionData CompletionData `json:"completionData"`
}

type BankID struct {
	authURL    string
	collectURL string
	client     *http.Client
}

func (b *BankID) makeRequest(request interface{}, endpoint string) (*http.Response, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	return b.client.Do(req)
}

func (b *BankID) Auth(request AuthRequest) (AuthResponse, error) {
	var response AuthResponse

	res, err := b.makeRequest(request, b.authURL)
	if err != nil {
		return response, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bs, err := io.ReadAll(res.Body)
		if err != nil {
			return response, err
		}
		return response, BankIDError{
			code:   res.StatusCode,
			status: res.Status,
			body:   fmt.Sprintf("%d; %s; %s", res.StatusCode, res.Status, string(bs)),
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil

}

func (b *BankID) Collect(request CollectRequest) (CollectResponse, error) {
	var response CollectResponse

	res, err := b.makeRequest(request, b.collectURL)
	if err != nil {
		return response, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bs, err := io.ReadAll(res.Body)
		if err != nil {
			return response, err
		}
		return response, BankIDError{
			code:   res.StatusCode,
			status: res.Status,
			body:   fmt.Sprintf("%d; %s; %s", res.StatusCode, res.Status, string(bs)),
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

type CancelRequest struct {
	OrderRef string `json:"orderRef"`
}

type BankIDError struct {
	code   int
	status string
	body   string
}

func (e BankIDError) Status() (int, string) {
	return e.code, e.status
}

func (e BankIDError) Error() string {
	return e.body
}

func (b *BankID) Cancel(request CancelRequest) (CollectResponse, error) {
	var response CollectResponse

	res, err := b.makeRequest(request, b.collectURL)
	if err != nil {
		return response, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bs, err := io.ReadAll(res.Body)
		if err != nil {
			return response, err
		}
		return response, BankIDError{
			code:   res.StatusCode,
			status: res.Status,
			body:   string(bs),
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return response, err
	}

	return response, nil
}

func (b *BankID) TryAuth(request AuthRequest) (CollectResponse, error) {
	var data CollectResponse
	var err error

	res, err := b.Auth(request)
	if err != nil {
		return data, err
	}

	req := CollectRequest{
		OrderRef: res.OrderRef,
	}

	// 30 second timeout, sleep for two seconds
	for i := 0; i < 15; i++ {
		data, err = b.Collect(req)
		if err != nil {
			return data, err
		}

		log.Println("OrderRef", data.OrderRef)
		log.Println("Status", data.Status)
		log.Println("HintCode", data.HintCode)
		log.Println("CompletionData", data.CompletionData)

		if data.Status == "pending" {
			time.Sleep(2 * time.Second)
			continue
		} else {
			break
		}
	}

	return data, err
}

type Config struct {
	BaseURL         string
	CertificatePath string
	CertificatePass string
	CaPath          string
	Insecure        bool
}

func New(c *Config) (*BankID, error) {
	file, err := os.ReadFile(c.CertificatePath)
	if err != nil {
		return nil, err
	}

	key, cert, caCerts, err := pkcs12.DecodeChain(file, c.CertificatePass)
	if err != nil {
		return nil, err
	}

	var certChain [][]byte
	certChain = append(certChain, cert.Raw)
	for _, ca := range caCerts {
		certChain = append(certChain, ca.Raw)
	}

	certificate := tls.Certificate{
		Certificate: certChain,
		PrivateKey:  key,
		Leaf:        cert,
	}

	// --- Load BankID Root CA ---
	caPEM, err := os.ReadFile(c.CaPath)
	if err != nil {
		return nil, err
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if !rootCAs.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("BankID CA PEM could not be parsed")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{certificate},
			RootCAs:            rootCAs,
			MinVersion:         tls.VersionTLS12,
			ServerName:         "appapi2.test.bankid.com", // IMPORTANT
			InsecureSkipVerify: c.Insecure,
		},
	}

	authURL := fmt.Sprintf("%s/rp/v6.0/auth", c.BaseURL)
	collectURL := fmt.Sprintf("%s/rp/v6.0/collect", c.BaseURL)

	return &BankID{
		authURL:    authURL,
		collectURL: collectURL,
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}, nil
}
