package model

import (
	"math"
	"upforschool/internal/pkg/worldline"

	"github.com/gofrs/uuid"
)

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

func (c *Core) AddOrder(o *OrderAdd) (int64, error) {
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
	err := c.db.Get(&resultID, query,
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

type OrderPaymentRequest struct {
	OrderID   int64
	Amount    float64
	Currency  worldline.Currency
	Reference string
}

type OrderPaymentResponse struct {
	RedirectURL string
}

func (c *Core) AddOrderPayment(payment OrderPaymentRequest) (*OrderPaymentResponse, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	query := `
    INSERT INTO order_payments (id, order_id, message, reference)
	VALUES ($1, $2, $3, $4)`

	request := c.worldline.NewOrder(payment.OrderID, payment.Amount, payment.Currency)

	order, err := c.worldline.CreateOrder(request)
	// log error
	if err != nil {
		if _, dberr := c.db.Exec(query,
			id,
			payment.OrderID,
			err.Error(),
			payment.Reference); dberr != nil {
			return nil, dberr
		}
		return nil, err
	}

	// log redirect
	_, err = c.db.Exec(query,
		id,
		payment.OrderID,
		order.URL,
		payment.Reference)

	return &OrderPaymentResponse{order.URL}, err
}

func (c *Core) CompleteOrder(id int64) error {
	query := `
	UPDATE orders
	   SET status = 'COMPLETED',
	       updated_at = CURRENT_TIMESTAMP
     WHERE id = $1`

	_, err := c.db.Exec(query, id)
	return err
}
