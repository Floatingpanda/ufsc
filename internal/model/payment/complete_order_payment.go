package service

import "github.com/gofrs/uuid"

func (s *Service) CompleteOrderPayment(orderID int64, params, reference string) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	query := `
    INSERT INTO order_payments (id, order_id, message, reference)
	VALUES ($1, $2, $3, $4)`

	_, err = s.db.db.Exec(query, id, orderID, params, reference)
	return err
}
