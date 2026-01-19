package service

import (
	"github.com/gofrs/uuid"
	"github.com/upforsports/match/service/worldline"
)

type OrderPaymentRequest struct {
	OrderID   int64
	Amount    float64
	Currency  worldline.Currency
	Reference string
}

type OrderPaymentResponse struct {
	RedirectURL string
}

func (s *Service) AddOrderPayment(payment OrderPaymentRequest) (*OrderPaymentResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	query := `
    INSERT INTO order_payments (id, order_id, message, reference)
	VALUES ($1, $2, $3, $4)`

	request := s.worldline.NewOrder(payment.OrderID, payment.Amount, payment.Currency)

	order, err := s.worldline.CreateOrder(request)
	// log error
	if err != nil {
		if _, dberr := s.db.db.Exec(query,
			id,
			payment.OrderID,
			err.Error(),
			payment.Reference); dberr != nil {
			return nil, dberr
		}
		return nil, err
	}

	// log redirect
	_, err = s.db.db.Exec(query,
		id,
		payment.OrderID,
		order.URL,
		payment.Reference)

	return &OrderPaymentResponse{order.URL}, err
}
