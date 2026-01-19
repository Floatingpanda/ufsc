package service

type UpdateOrderCurrencyRequest struct {
	GameID int64 `db:"game_id"`
	// TaxAmount float64 `db:"tax_amount"`
	// DiscountID     *int64  `db:"discount_id"`
	// DiscountAmount float64 `db:"discount_amount"`
	ProductCost float64 `db:"product_cost"`
	ProductTax  float64 `db:"product_tax"`
	Currency    string  `db:"currency"`
}

func (s *Service) UpdateOrderCurrency(r UpdateOrderCurrencyRequest) error {

	cost := calculateCost(r.ProductCost, r.ProductTax, 0)

	query := `
    UPDATE orders SET
		amount = $2,
		tax_amount = $3,
		discount_id = $4,
		discount_amount = $5,
		product_cost = $6,
		product_tax = $7,
		currency = $8
    WHERE game_id = $1
	AND STATUS = 'CREATED'
	RETURNING id
	`
	var updatedId string
	err := s.db.db.Get(&updatedId, query,
		r.GameID,
		cost.Total,
		cost.Tax,
		nil,
		0,
		r.ProductCost,
		r.ProductTax,
		r.Currency,
	)
	if err != nil {
		return err
	}

	return nil
}
