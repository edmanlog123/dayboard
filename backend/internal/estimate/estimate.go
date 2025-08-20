package estimate

import (
	"context"
	"fmt"

	"dayboard/backend/internal/db"
)

// TaxResult holds the computed tax amounts and net values for a given
// income, state and filing status. All monetary values are in cents.
type TaxResult struct {
	FederalCents        int `json:"federalCents"`
	StateCents          int `json:"stateCents"`
	FicaCents           int `json:"ficaCents"`
	PerPaycheckNetCents int `json:"perPaycheckNetCents"`
	TermNetCents        int `json:"termNetCents"`
}

// EstimateTaxes estimates U.S. federal, state, and FICA taxes for a given annual
// income (in cents). It looks up the progressive tax brackets stored in
// tax_tables_federal and tax_tables_state. FilingStatus must be either
// "single" or "married"; other values return an error. The year parameter
// allows supporting future/previous tax years. The result includes the
// after-tax take-home per paycheck over the given termWeeks.
func EstimateTaxes(ctx context.Context, d *db.DB, incomeCents int, state string, filingStatus string, year int, payFreq string, termWeeks int) (*TaxResult, error) {
	// Determine standard deduction based on filing status.
	var stdDeduction int
	switch filingStatus {
	case "single":
		row := d.QueryRowContext(ctx, `SELECT DISTINCT std_deduction_single FROM tax_tables_federal WHERE year = $1 LIMIT 1`, year)
		if err := row.Scan(&stdDeduction); err != nil {
			return nil, fmt.Errorf("failed to fetch std deduction: %w", err)
		}
	case "married":
		// Not implemented: add support for married filing jointly.
		return nil, fmt.Errorf("married filing jointly not yet supported")
	default:
		return nil, fmt.Errorf("unsupported filing status: %s", filingStatus)
	}

	taxableIncome := incomeCents - stdDeduction
	if taxableIncome < 0 {
		taxableIncome = 0
	}
	// Compute federal tax.
	var federalTax int
	rows, err := d.QueryContext(ctx, `
        SELECT bracket_low, bracket_high, rate_bps
        FROM tax_tables_federal WHERE year = $1
        ORDER BY bracket_low ASC
    `, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	remaining := taxableIncome
	for rows.Next() {
		var low, high, rateBps int
		if err := rows.Scan(&low, &high, &rateBps); err != nil {
			return nil, err
		}
		if remaining <= 0 {
			break
		}
		// Determine portion of income in this bracket.
		upperBound := high
		if high == 0 { // zero or null high implies no upper bound (top bracket)
			upperBound = taxableIncome
		}
		// Determine taxable amount in this bracket.
		segment := min(remaining, upperBound-low)
		federalTax += segment * rateBps / 10000 // rate_bps is basis points
		remaining -= segment
	}
	// Compute state tax. If state is unknown, assume zero.
	var stateTax int
	if state != "" {
		rows, err := d.QueryContext(ctx, `
            SELECT bracket_low, bracket_high, rate_bps
            FROM tax_tables_state WHERE year = $1 AND state = $2
            ORDER BY bracket_low ASC
        `, year, state)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		remaining = taxableIncome
		for rows.Next() {
			var low, high, rateBps int
			if err := rows.Scan(&low, &high, &rateBps); err != nil {
				return nil, err
			}
			if remaining <= 0 {
				break
			}
			upperBound := high
			if high == 0 {
				upperBound = taxableIncome
			}
			segment := min(remaining, upperBound-low)
			stateTax += segment * rateBps / 10000
			remaining -= segment
		}
	}
	// Estimate FICA (Social Security + Medicare) at 7.65% for simplicity.
	ficaTax := incomeCents * 765 / 10000
	// Determine number of paychecks in the term.
	var checks int
	switch payFreq {
	case "weekly":
		checks = termWeeks
	case "biweekly":
		checks = termWeeks / 2
	case "monthly":
		// Approximate 4 weeks per month. Multiply by termWeeks/4.
		checks = termWeeks / 4
	default:
		checks = termWeeks / 2
	}
	totalTax := federalTax + stateTax + ficaTax
	netAnnual := incomeCents - totalTax
	// Net per paycheck. Avoid division by zero.
	perPay := 0
	if checks > 0 {
		perPay = netAnnual / checks
	}
	result := &TaxResult{
		FederalCents:        federalTax,
		StateCents:          stateTax,
		FicaCents:           ficaTax,
		PerPaycheckNetCents: perPay,
		TermNetCents:        netAnnual,
	}
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
