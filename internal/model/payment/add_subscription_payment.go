package service

import (
	"github.com/gofrs/uuid"
	"github.com/upforsports/match/service/worldline"
)

type SubscriptionPaymentRequest struct {
	SubscriptionID int64
	Amount         float64
	Currency       worldline.Currency
	Reference      string
}

type SubscriptionPaymentResponse struct {
	RedirectURL string
}

func (s *Service) AddSubscriptionPayment(payment SubscriptionPaymentRequest) (*SubscriptionPaymentResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	query := `
    INSERT INTO subscription_payments (id, subscription_id, message, reference)
	VALUES ($1, $2, $3, $4)`

	request := s.worldline.NewSubscription(payment.SubscriptionID, payment.Amount, payment.Currency)

	order, err := s.worldline.CreateOrder(request)

	// log error
	if err != nil {
		if _, dberr := s.db.db.Exec(query,
			id,
			payment.SubscriptionID,
			err.Error(),
			payment.Reference); dberr != nil {
			return nil, dberr
		}

		return nil, err
	}

	// log redirect
	_, err = s.db.db.Exec(query,
		id,
		payment.SubscriptionID,
		order.URL,
		payment.Reference)

	return &SubscriptionPaymentResponse{order.URL}, err
}
