package service

import "math"

type Cost struct {
	Total    float64
	Discount float64
	Tax      float64
}

func calculateCost(price, taxPercentage, discountPercentage float64) Cost {
	costInCents := price * 100
	discount := costInCents * (discountPercentage / 100.0)
	discount = math.Round(discount)

	total := costInCents - discount

	quotient := taxPercentage/100 + 1
	wotax := math.Round(total / quotient)
	taxamount := total - wotax

	return Cost{
		Total:    total / 100,
		Discount: discount / 100,
		Tax:      taxamount / 100,
	}
}

type OrderAdd struct {
	GameID          int64
	Quantity        int
	ReferenceUser   string
	ReferenceNumber string
	ProductName     string
	ProductCost     float64
	ProductTax      float64
	ProductCategory string
	Currency        string
}

func (s *Service) AddOrder(o *OrderAdd) (int64, error) {
	discount := 0.0

	if o.Currency == "" {
		o.Currency = "SEK"
	}

	cost := calculateCost(o.ProductCost, o.ProductTax, discount)

	query := `
    INSERT INTO orders (
           game_id,
           quantity,
           amount,
           tax_amount,
           discount_amount,
           product_name,
           product_cost,
           product_tax,
		   product_category,
           reference_user,
           reference_number,
		   status,
		   currency
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	RETURNING id`

	var resultID int64
	err := s.db.db.Get(&resultID, query,
		o.GameID,
		o.Quantity,
		cost.Total,
		cost.Tax,
		cost.Discount,
		o.ProductName,
		o.ProductCost,
		o.ProductTax,
		o.ProductCategory,
		o.ReferenceUser,
		o.ReferenceNumber,
		"CREATED",
		o.Currency,
	)

	return resultID, err
}
