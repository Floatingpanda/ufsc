package service

import (
	"database/sql"
)

type SubscriptionAdd struct {
	PlanID         int64
	OrganizationID int64
	Reference      string
	IsDiscounted   bool
	DiscountID     int64
}

type subscriptionPlan struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Role     string `db:"role"`
	Period   int64  `db:"period"`
	Interval int64  `db:"interval"`
	Price    int64  `db:"price"`
	Tax      int64  `db:"tax"`
	Currency string `db:"currency"`
}

func (s *Service) subscriptionPlanByID(id int64) (*subscriptionPlan, error) {
	query := `
	SELECT p.id AS id,
           p.name,
           p.role,
           p.period,
           p.interval,
           p.price,
		   p.tax,
           p.currency
      FROM plans AS p
     WHERE p.id = $1
       AND p.active`

	var p subscriptionPlan
	return &p, s.db.db.Get(&p, query, id)
}

type subscriptionDiscount struct {
	ID        int64
	Code      string
	Amount    int64
	IsPercent bool `db:"is_percent"`
}

func (s *Service) subscriptionDiscountByID(id int64) (*subscriptionDiscount, error) {
	query := `
    SELECT d.id AS id,
           d.code AS code,
           d.amount AS amount,
		   d.is_percent as is_percent
      FROM discounts AS d
     WHERE d.id = $1
       AND d.valid_to >= CURRENT_TIMESTAMP`

	var d subscriptionDiscount
	err := s.db.db.Get(&d, query, id)
	return &d, err
}

func (s *Service) AddSubscription(sub *SubscriptionAdd) (int64, error) {
	plan, err := s.subscriptionPlanByID(sub.PlanID)
	if err != nil {
		return 0, err
	}

	discountPercent := float64(0)
	discountID := sql.NullInt64{}
	if sub.IsDiscounted {
		discount, err := s.subscriptionDiscountByID(sub.DiscountID)
		if err != nil {
			return 0, err
		}

		discountPercent = float64(discount.Amount)
		if !discount.IsPercent && plan.Price > 0 {
			discountPercent = float64(discount.Amount) / float64(plan.Price) * 100
			if discountPercent > 100 {
				discountPercent = 100
			}
		}

		discountID.Int64 = discount.ID
		discountID.Valid = true
	}

	cost := calculateCost(float64(plan.Price), float64(plan.Tax), discountPercent)

	query := `
	INSERT INTO subscriptions (
           plan_id,
           organization_id,
           discount_id,
		   reference,
           amount,
           tax_amount,
           discount_amount)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
 RETURNING id`

	var id int64
	if err := s.db.db.Get(&id, query,
		plan.ID,
		sub.OrganizationID,
		discountID,
		sub.Reference,
		cost.Total,
		cost.Tax,
		cost.Discount); err != nil {
		return id, err
	}

	return id, nil
}
