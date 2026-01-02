package jwt

import (
	"crypto/rsa"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt"
)

var (
	ErrBadRequest   = fmt.Errorf("invalid request")
	ErrAuthExpired  = fmt.Errorf("authentication token has expired")
	ErrUnauthorized = fmt.Errorf("unauthorized request")
)

var (
	ErrMissingPrivateKey     = fmt.Errorf("service cannot create tokens, missing private key")
	ErrInvalid               = fmt.Errorf("%w, parsed jwt is not valid", ErrBadRequest)
	ErrExpired               = fmt.Errorf("%w, parsed jwt is expired", ErrAuthExpired)
	ErrBadPayload            = fmt.Errorf("%w, parsed jwt has unexpected payload", ErrBadRequest)
	ErrSigningMethodMismatch = fmt.Errorf("%w, parsed jwt has unexpected signing method", ErrUnauthorized)
)

const PayloadKey = "payload"

type rsaKeys struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

type JWTConfig struct {
	SigningMethod jwt.SigningMethod
	Issuer        string

	// SingleSecretKey - some unique string that will be used to encrypt and decrypt tokens, if set then RsaKeys are not needed
	// SigningMethod will be defaulted to SigningMethodHS256
	SingleSecretKey *string

	// RsaKeys - required if SingleSecretKey is null
	// SigningMethod will be defaulted to SigningMethodRS256
	RsaPublicKey *[]byte
	// PrivateKey is optional if service will not create tokens
	RsaPrivateKey *[]byte
}

type Service struct {
	config        JWTConfig
	signingMethod jwt.SigningMethod
	rsaKeys       *rsaKeys
}

var DefaultRSASigningMethod jwt.SigningMethod = jwt.SigningMethodRS256
var DefaultHMACSigningMethod jwt.SigningMethod = jwt.SigningMethodHS256

func New(config JWTConfig) *Service {

	if config.RsaPublicKey == nil &&
		config.RsaPrivateKey == nil &&
		config.SingleSecretKey == nil {
		log.Fatal("Failed to create jwt service, RSA keys or single secret key is required")
	}

	svc := Service{
		config: config,
	}

	// parse private rsa key and generate public part
	var err error
	if config.RsaPrivateKey != nil {
		svc.rsaKeys = &rsaKeys{}
		svc.rsaKeys.PrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM(*config.RsaPrivateKey)
		if err != nil {
			log.Fatal("failed to parse private key;", err)
		}
		pubKey := svc.rsaKeys.PrivateKey.PublicKey
		svc.rsaKeys.PublicKey = &pubKey
	}

	// if only public RSA key is provided, parse only public key
	if config.RsaPublicKey != nil && config.RsaPrivateKey == nil {
		svc.rsaKeys = &rsaKeys{}
		svc.rsaKeys.PublicKey, err = jwt.ParseRSAPublicKeyFromPEM(*config.RsaPublicKey)
		if err != nil {
			log.Fatal("failed to parse public key; ", err)
		}
	}

	// find signing method
	var signingMethod jwt.SigningMethod
	if svc.rsaKeys != nil {
		// use RSA, asymmetric signing algorithm (private/public key pair)
		signingMethod = DefaultRSASigningMethod
	} else {
		// use HMAC, symmetric signing algorithm (single secret key)
		signingMethod = DefaultHMACSigningMethod
	}

	// allow setting explicit signingmethod
	if config.SigningMethod != nil {
		signingMethod = config.SigningMethod
	}

	svc.signingMethod = signingMethod

	err = svc.HealthCheck()
	if err != nil {
		log.Fatal("failed sanity test, unable to create auth token; ", err)
	}

	return &svc
}

func (s *Service) HealthCheck() error {

	// Without signing key, we cannot be sure we are healthy
	// Issue could be on signing side or validation side
	_, ok := s.signingKey()
	if !ok {
		return nil
	}

	input := BaseClaim{}
	token, err := s.NewToken(&input, 1*time.Second)
	if err != nil {
		return err
	}

	result := BaseClaim{}
	err = s.Validate(&result, *token)
	return err
}

func (s *Service) signingKey() (interface{}, bool) {
	if s.rsaKeys != nil {
		if s.config.RsaPrivateKey == nil {
			return nil, false
		}
		return s.rsaKeys.PrivateKey, true
	}

	return []byte(*s.config.SingleSecretKey), true
}

func (s *Service) validationKey() interface{} {
	if s.rsaKeys != nil {
		return s.rsaKeys.PublicKey
	}
	return []byte(*s.config.SingleSecretKey)
}

// Make this part of your struct, and it can be passed as tokenizable
type BaseClaim jwt.StandardClaims

func (bc *BaseClaim) SetBaseClaim(baseClaim BaseClaim) {
	*bc = baseClaim
}

func (bc BaseClaim) Valid() error {
	if time.Now().After(time.Unix(bc.ExpiresAt, 0)) {
		return ErrExpired
	}
	return jwt.StandardClaims(bc).Valid()
}

type Tokenizable interface {
	SetBaseClaim(BaseClaim)
	Valid() error
}

func (s *Service) NewToken(payload Tokenizable, validityDuration time.Duration) (*string, error) {

	// Different signing keys might require different types for the key
	key, ok := s.signingKey()
	if !ok {
		return nil, ErrMissingPrivateKey
	}

	payload.SetBaseClaim(
		BaseClaim{
			ExpiresAt: time.Now().Add(validityDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    s.config.Issuer,
		},
	)
	token := jwt.NewWithClaims(s.signingMethod, payload)

	tokenString, err := token.SignedString(key)
	if err != nil {
		return nil, err
	}

	return &tokenString, nil
}

func (s *Service) Validate(dest Tokenizable, token string) error {

	key := s.validationKey()

	jwtToken, err := jwt.ParseWithClaims(token, dest, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != s.signingMethod.Alg() {
			return nil, ErrSigningMethodMismatch
		}
		return key, nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse jwt: %w", err)
	}

	if !jwtToken.Valid {
		return ErrInvalid
	}

	return nil
}
