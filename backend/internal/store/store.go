package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"dayboard/backend/internal/db"
)

// Event represents a calendar event stored in the database. It mirrors the
// columns of the calendar_events table and is returned to the API caller.
type Event struct {
	ID       uuid.UUID `json:"id"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Title    string    `json:"title"`
	JoinURL  string    `json:"joinURL"`
	Location string    `json:"location"`
}

// Subscription represents a recurring payment. AmountCents and cadence
// determine the billing schedule. NextDue may be nil if unknown.
type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	Merchant    string     `json:"merchant"`
	AmountCents int        `json:"amountCents"`
	CadenceDays int        `json:"cadenceDays"`
	NextDue     *time.Time `json:"nextDue,omitempty"`
	Source      string     `json:"source"`
	IsActive    bool       `json:"isActive"`
}

// Profile holds user-specific settings used for tax and cost estimation.
// All monetary values are stored as cents to avoid floating point errors.
type Profile struct {
	UserID        uuid.UUID
	HomeAddr      string
	OfficeAddr    string
	City          string
	State         string
	HourlyCents   *int
	HoursPerWeek  *int
	StipendCents  *int
	PayFreq       string
	StartDate     *time.Time
	InOfficeDays  int
	FoodCostCents int
}

// GetTodayEvents returns all events for a user that start on the given day.
// The caller is responsible for passing startOfDay and endOfDay in UTC.
func GetTodayEvents(ctx context.Context, d *db.DB, userID uuid.UUID, startOfDay, endOfDay time.Time) ([]Event, error) {
	rows, err := d.QueryContext(ctx, `
        SELECT id, start_ts, end_ts, title, join_url, location
        FROM calendar_events
        WHERE user_id = $1
          AND start_ts >= $2
          AND start_ts < $3
        ORDER BY start_ts ASC
    `, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		var e Event
		var id string
		if err := rows.Scan(&id, &e.Start, &e.End, &e.Title, &e.JoinURL, &e.Location); err != nil {
			return nil, err
		}
		uid, _ := uuid.Parse(id)
		e.ID = uid
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetSubscriptions returns all active subscriptions for a user.
func GetSubscriptions(ctx context.Context, d *db.DB, userID uuid.UUID) ([]Subscription, error) {
	rows, err := d.QueryContext(ctx, `
        SELECT id, merchant, amount_cents, cadence_days, next_due, source, is_active
        FROM subscriptions
        WHERE user_id = $1 AND is_active = true
        ORDER BY next_due ASC NULLS LAST
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []Subscription
	for rows.Next() {
		var s Subscription
		var id string
		var nextDue pgtype.Date
		if err := rows.Scan(&id, &s.Merchant, &s.AmountCents, &s.CadenceDays, &nextDue, &s.Source, &s.IsActive); err != nil {
			return nil, err
		}
		s.ID, _ = uuid.Parse(id)
		if !nextDue.Time.IsZero() && nextDue.Valid {
			// pgtype.Date stores date in nextDue.Time
			t := nextDue.Time
			s.NextDue = &t
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// CreateSubscription inserts a new manual subscription for the user. Plaid-detected
// subscriptions should be inserted via separate routines. Returns the created
// subscription or an error.
func CreateSubscription(ctx context.Context, d *db.DB, userID uuid.UUID, s Subscription) (*Subscription, error) {
	// Basic validation
	if s.Merchant == "" || s.AmountCents <= 0 || s.CadenceDays <= 0 {
		return nil, errors.New("invalid subscription fields")
	}
	id := uuid.New()
	_, err := d.ExecContext(ctx, `
        INSERT INTO subscriptions (id, user_id, merchant, amount_cents, cadence_days, next_due, source, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, 'manual', true)
    `, id, userID, s.Merchant, s.AmountCents, s.CadenceDays, s.NextDue)
	if err != nil {
		return nil, err
	}
	s.ID = id
	s.Source = "manual"
	s.IsActive = true
	return &s, nil
}

// GetProfile retrieves the user's profile. If no profile exists, returns
// (nil, nil) to signal caller to create a default. Do not create default
// profiles automatically here to avoid unexpected writes.
func GetProfile(ctx context.Context, d *db.DB, userID uuid.UUID) (*Profile, error) {
	row := d.QueryRowContext(ctx, `
        SELECT home_addr, office_addr, city, state, hourly_cents, hours_per_week,
               stipend_cents, pay_freq, start_date, in_office_days, food_cost_cents
        FROM profiles WHERE user_id = $1
    `, userID)
	var p Profile
	p.UserID = userID
	var hourly, stipend sql.NullInt64
	var hours sql.NullInt32
	var start sql.NullTime
	if err := row.Scan(&p.HomeAddr, &p.OfficeAddr, &p.City, &p.State, &hourly, &hours, &stipend, &p.PayFreq, &start, &p.InOfficeDays, &p.FoodCostCents); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if hourly.Valid {
		v := int(hourly.Int64)
		p.HourlyCents = &v
	}
	if hours.Valid {
		v := int(hours.Int32)
		p.HoursPerWeek = &v
	}
	if stipend.Valid {
		v := int(stipend.Int64)
		p.StipendCents = &v
	}
	if start.Valid {
		t := start.Time
		p.StartDate = &t
	}
	return &p, nil
}

// UpsertProfile inserts or updates a user's profile. If a profile does not
// exist, one is created. Otherwise, the existing record is updated.
func UpsertProfile(ctx context.Context, d *db.DB, p Profile) error {
	_, err := d.ExecContext(ctx, `
        INSERT INTO profiles (
            user_id, home_addr, office_addr, city, state, hourly_cents,
            hours_per_week, stipend_cents, pay_freq, start_date,
            in_office_days, food_cost_cents
        ) VALUES (
            $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
        )
        ON CONFLICT (user_id) DO UPDATE SET
            home_addr = EXCLUDED.home_addr,
            office_addr = EXCLUDED.office_addr,
            city = EXCLUDED.city,
            state = EXCLUDED.state,
            hourly_cents = EXCLUDED.hourly_cents,
            hours_per_week = EXCLUDED.hours_per_week,
            stipend_cents = EXCLUDED.stipend_cents,
            pay_freq = EXCLUDED.pay_freq,
            start_date = EXCLUDED.start_date,
            in_office_days = EXCLUDED.in_office_days,
            food_cost_cents = EXCLUDED.food_cost_cents
    `, p.UserID, p.HomeAddr, p.OfficeAddr, p.City, p.State, p.HourlyCents,
		p.HoursPerWeek, p.StipendCents, p.PayFreq, p.StartDate,
		p.InOfficeDays, p.FoodCostCents)
	return err
}
