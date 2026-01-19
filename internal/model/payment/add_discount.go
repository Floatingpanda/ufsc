package service

import (
	"errors"
	"math"
)

type discount struct {
	ID         int64   `db:"id"`
	Code       string  `db:"code"`
	Amount     int64   `db:"amount"`
	IsCampaign bool    `db:"is_campaign"`
	IsPercent  bool    `db:"is_percent"`
	Currency   *string `db:"currency"`
	IsValid    bool    `db:"is_valid"`
	IsUsed     bool    `db:"is_used"`
}

func (d *discount) Usable() bool {
	if !d.IsValid {
		return false
	}

	if d.IsCampaign {
		return true
	}

	return !d.IsUsed
}

func (s *Service) discountByCode(code string) (*discount, error) {
	query := `
	SELECT d.id AS id,
           d.code AS code,
           d.amount AS amount,
           d.campaign AS is_campaign,
		   d.is_percent AS is_percent,
		   d.currency AS currency,
           d.valid_to >= CURRENT_TIMESTAMP AS is_valid,
           (count(o.id) > 0 or count(s.id) > 0) AS is_used
      FROM discounts AS d
       LEFT JOIN orders AS o
         ON o.discount_id = d.id
       LEFT JOIN subscriptions AS s
         ON s.discount_id = d.id
      WHERE d.code = $1
      GROUP by d.id`

	var d discount
	return &d, s.db.db.Get(&d, query, code)
}

type productInfo struct {
	ProductCost float64 `db:"product_cost"`
	ProductTax  float64 `db:"product_tax"`
	Currency    string  `db:"currency"`
}

func (s *Service) productInfoByGameID(gameID int64) (*productInfo, error) {
	query := `
	SELECT o.product_cost,
	       o.product_tax,
		   o.currency as currency
      FROM orders AS o
     WHERE o.game_id = $1`

	var p productInfo
	return &p, s.db.db.Get(&p, query, gameID)
}

func (s *Service) productInfoByOrderID(orderID int64) (*productInfo, error) {
	query := `
	SELECT o.product_cost,
	       o.product_tax,
		   o.currency as currency
      FROM orders AS o
     WHERE o.id = $1`

	var p productInfo
	return &p, s.db.db.Get(&p, query, orderID)
}

func (s *Service) AddDiscount(gameID int64, code string) error {
	d, err := s.discountByCode(code)
	if err != nil {
		return err
	}

	if !d.Usable() {
		return errors.New("expired discount code")
	}

	product, err := s.productInfoByGameID(gameID)
	if err != nil {
		return err
	}
	if !d.IsPercent && (d.Currency == nil || *d.Currency != product.Currency) {
		return errors.New("discount code not applicable to currency")
	}

	taxQuotent := (float64(product.ProductTax) / 100) + 1
	originalAmount := product.ProductCost

	var discountAmount float64
	if d.IsPercent {
		discount := float64(d.Amount) / 100
		discountAmount = originalAmount * discount
		discountAmount = math.Round(discountAmount*10) / 10
	} else {
		discountAmount = float64(d.Amount)
	}

	// Ensure the discounted amount doesn't go below zero
	if discountAmount > originalAmount {
		discountAmount = originalAmount
	}

	finalAmount := originalAmount - discountAmount
	finalAmount = math.Round(finalAmount*100) / 100 // Round to 2 decimal places

	amountWoTax := finalAmount / taxQuotent
	amountWoTax = math.Round(amountWoTax*100) / 100 // Round to 2 decimal places

	taxAmount := finalAmount - amountWoTax
	taxAmount = math.Round(taxAmount*10) / 10

	query := `
	UPDATE orders
	  SET discount_id = $1,
          amount = $3,
          tax_amount = $4,
		  discount_amount = $5
    WHERE game_id = $2
	  AND status != 'COMPLETED'
	`

	_, err = s.db.db.Exec(query, d.ID, gameID,
		finalAmount,
		taxAmount,
		discountAmount)
	return err
}

func (s *Service) AddDiscountToOrder(orderID int64, code string) error {
	d, err := s.discountByCode(code)
	if err != nil {
		return err
	}

	if !d.Usable() {
		return errors.New("expired discount code")
	}

	product, err := s.productInfoByOrderID(orderID)
	if err != nil {
		return err
	}
	if !d.IsPercent && (d.Currency == nil || *d.Currency != product.Currency) {
		return errors.New("discount code not applicable to currency")
	}

	taxQuotent := (float64(product.ProductTax) / 100) + 1
	originalAmount := product.ProductCost

	var discountAmount float64
	if d.IsPercent {
		discount := float64(d.Amount) / 100
		discountAmount = originalAmount * discount
		discountAmount = math.Round(discountAmount*10) / 10
	} else {
		discountAmount = float64(d.Amount)
	}

	// Ensure the discounted amount doesn't go below zero
	if discountAmount > originalAmount {
		discountAmount = originalAmount
	}

	finalAmount := originalAmount - discountAmount
	finalAmount = math.Round(finalAmount*100) / 100 // Round to 2 decimal places

	amountWoTax := finalAmount / taxQuotent
	amountWoTax = math.Round(amountWoTax*100) / 100 // Round to 2 decimal places

	taxAmount := finalAmount - amountWoTax
	taxAmount = math.Round(taxAmount*10) / 10

	query := `
	UPDATE orders
	  SET discount_id = $1,
          amount = $3,
          tax_amount = $4,
		  discount_amount = $5
    WHERE id = $2
	  AND status != 'COMPLETED'
	`
	_, err = s.db.db.Exec(query, d.ID, orderID,
		finalAmount,
		taxAmount,
		discountAmount)
	return err
}
