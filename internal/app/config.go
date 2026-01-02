package app

import (
	"encoding/json"
	"os"

	"upforschool/internal/database"
)

// Config for App.
type Config struct {
	App struct {
		IsDev      bool   `json:"isDev"`
		Addr       string `json:"addr"`
		URL        string `json:"url"`
		StaticPath string `json:"staticPath"`
		UploadPath string `json:"uploadPath"`
		Templates  string `json:"templates"`
		// MapsKey         string `json:"mapsKey"`
		// GoogleAnalytics string `json:"googleAnalytics"`
	} `json:"app"`
	Cookie struct {
		HashKey  string `json:"hashKey"`
		BlockKey string `json:"blockKey"`
	} `json:"cookie"`
	BankID struct {
		BaseURL         string `json:"baseURL"`
		CertificatePath string `json:"certPath"`
		CertificatePass string `json:"certPass"`
		CaPath          string `json:"caPath"`
	} `json:"bankid"`
	Email struct {
		AccesskeyID     string `json:"accesskeyID"`
		SecretAccessKey string `json:"secretAccessKey"`
		Region          string `json:"region"`
	} `json:"email"`
	Worldline struct {
		Merchant string
		Username string
		Password string
		MD5key   string
	} `json:"worldline"`
	DB *database.Config `json:"db"`
}

// ReadConfig from file name.
func ReadConfig(name string) (*Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	return &cfg, json.NewDecoder(f).Decode(&cfg)
}
