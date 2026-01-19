package service

import "github.com/gofrs/uuid"

func (s *Service) CompleteSubscriptionPayment(subscriptionID int64, params, reference string) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}

	query := `
    INSERT INTO subscription_payments (id, subscription_id, message, reference)
	VALUES ($1, $2, $3, $4)`

	_, err = s.db.db.Exec(query, id, subscriptionID, params, reference)
	return err
}
