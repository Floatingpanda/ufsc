package service

func (s *Service) ActivateSubscription(id, period int64) error {
	query := `
	UPDATE subscriptions
       SET activated_at = CURRENT_TIMESTAMP,
           starts_at = CURRENT_TIMESTAMP,
           ends_at = (CURRENT_TIMESTAMP + interval '1 month' * $2)
     WHERE id = $1`

	_, err := s.db.db.Exec(query, id, period)
	return err
}
