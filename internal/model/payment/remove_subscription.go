package service

func (s *Service) RemoveSubscription(id int64) error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM subscription_payments WHERE subscription_id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
