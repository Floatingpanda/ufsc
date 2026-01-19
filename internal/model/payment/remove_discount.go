package service

import "math"

// RemoveDiscount for game.
func (s *Service) RemoveDiscount(gameID int64) error {
	p, err := s.productInfoByGameID(gameID)
	if err != nil {
		return err
	}

	taxQuotent := (float64(p.ProductTax) / 100) + 1

	amount := p.ProductCost
	amountWoTax := amount / taxQuotent
	amountWoTax = math.Round(amountWoTax*100) / 100

	taxAmount := amount - amountWoTax
	taxAmount = math.Round(taxAmount*10) / 10

	query := `
    UPDATE orders
       SET discount_id = NULL,
           amount = $2,
           tax_amount = $3,
           discount_amount = 0
     WHERE game_id = $1
       AND status != 'COMPLETED'`

	_, err = s.db.db.Exec(query, gameID, amount, taxAmount)
	return err
}
