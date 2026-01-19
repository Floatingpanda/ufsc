package service

func (s *Service) CompleteOrder(id int64) error {
	query := `
	UPDATE orders
	   SET status = 'COMPLETED',
	       updated_at = CURRENT_TIMESTAMP
     WHERE id = $1`

	_, err := s.db.db.Exec(query, id)
	return err
}
