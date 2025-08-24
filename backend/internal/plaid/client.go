package plaid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// PlaidService handles Plaid API operations
type PlaidService struct {
	clientID string
	secret   string
	env      string
	baseURL  string
}

// LinkTokenResponse represents the response from creating a link token
type LinkTokenResponse struct {
	LinkToken  string    `json:"link_token"`
	Expiration time.Time `json:"expiration"`
	RequestID  string    `json:"request_id"`
}

// AccessTokenResponse represents the response from exchanging public token
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
	RequestID   string `json:"request_id"`
}

// Account represents a Plaid account
type Account struct {
	ID           string  `json:"account_id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	Balance      float64 `json:"balance"`
	CurrencyCode string  `json:"iso_currency_code"`
}

// Transaction represents a Plaid transaction
type Transaction struct {
	ID             string    `json:"transaction_id"`
	AccountID      string    `json:"account_id"`
	Amount         float64   `json:"amount"`
	Date           time.Time `json:"date"`
	Name           string    `json:"name"`
	MerchantName   string    `json:"merchant_name"`
	Category       []string  `json:"category"`
	Pending        bool      `json:"pending"`
	PaymentChannel string    `json:"payment_channel"`
}

// NewPlaidService creates a new Plaid service
func NewPlaidService() *PlaidService {
	env := os.Getenv("PLAID_ENV")
	if env == "" {
		env = "sandbox"
	}

	var baseURL string
	switch env {
	case "sandbox":
		baseURL = "https://sandbox.plaid.com"
	case "development":
		baseURL = "https://development.plaid.com"
	case "production":
		baseURL = "https://production.plaid.com"
	default:
		baseURL = "https://sandbox.plaid.com"
	}

	return &PlaidService{
		clientID: os.Getenv("PLAID_CLIENT_ID"),
		secret:   os.Getenv("PLAID_SECRET"),
		env:      env,
		baseURL:  baseURL,
	}
}

// CreateLinkToken creates a link token for Plaid Link
func (s *PlaidService) CreateLinkToken(ctx context.Context, userID string) (*LinkTokenResponse, error) {
	payload := map[string]interface{}{
		"client_id":     s.clientID,
		"secret":        s.secret,
		"client_name":   "DayBoard",
		"country_codes": []string{"US"},
		"language":      "en",
		"user": map[string]string{
			"client_user_id": userID,
		},
		"products":                       []string{"transactions"},
		"required_if_supported_products": []string{"transactions"},
		"redirect_uri":                   os.Getenv("PLAID_REDIRECT_URI"),
	}

	var result LinkTokenResponse
	_, err := s.makeRequest(ctx, "/link/token/create", payload, &result)
	return &result, err
}

// ExchangePublicToken exchanges a public token for an access token
func (s *PlaidService) ExchangePublicToken(ctx context.Context, publicToken string) (*AccessTokenResponse, error) {
	payload := map[string]interface{}{
		"client_id":    s.clientID,
		"secret":       s.secret,
		"public_token": publicToken,
	}

	var result AccessTokenResponse
	_, err := s.makeRequest(ctx, "/link/token/exchange", payload, &result)
	return &result, err
}

// GetAccounts retrieves accounts for an access token
func (s *PlaidService) GetAccounts(ctx context.Context, accessToken string) ([]Account, error) {
	payload := map[string]interface{}{
		"client_id":    s.clientID,
		"secret":       s.secret,
		"access_token": accessToken,
	}

	var response struct {
		Accounts []struct {
			ID       string `json:"account_id"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			Subtype  string `json:"subtype"`
			Balances struct {
				Available float64 `json:"available"`
				Current   float64 `json:"current"`
				ISO       string  `json:"iso_currency_code"`
			} `json:"balances"`
		} `json:"accounts"`
		RequestID string `json:"request_id"`
	}

	_, err := s.makeRequest(ctx, "/accounts/get", payload, &response)
	if err != nil {
		return nil, err
	}

	var accounts []Account
	for _, acc := range response.Accounts {
		accounts = append(accounts, Account{
			ID:           acc.ID,
			Name:         acc.Name,
			Type:         acc.Type,
			Subtype:      acc.Subtype,
			Balance:      acc.Balances.Current,
			CurrencyCode: acc.Balances.ISO,
		})
	}

	return accounts, nil
}

// GetTransactions retrieves transactions for the last 30 days
func (s *PlaidService) GetTransactions(ctx context.Context, accessToken string) ([]Transaction, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // Last 30 days

	payload := map[string]interface{}{
		"client_id":    s.clientID,
		"secret":       s.secret,
		"access_token": accessToken,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
		"count":        500,
		"offset":       0,
	}

	var response struct {
		Accounts []struct {
			ID   string `json:"account_id"`
			Name string `json:"name"`
		} `json:"accounts"`
		Transactions []struct {
			ID             string   `json:"transaction_id"`
			AccountID      string   `json:"account_id"`
			Amount         float64  `json:"amount"`
			Date           string   `json:"date"`
			Name           string   `json:"name"`
			MerchantName   string   `json:"merchant_name"`
			Category       []string `json:"category"`
			Pending        bool     `json:"pending"`
			PaymentChannel string   `json:"payment_channel"`
		} `json:"transactions"`
		TotalTransactions int    `json:"total_transactions"`
		RequestID         string `json:"request_id"`
	}

	_, err := s.makeRequest(ctx, "/transactions/get", payload, &response)
	if err != nil {
		return nil, err
	}

	var transactions []Transaction
	for _, txn := range response.Transactions {
		date, _ := time.Parse("2006-01-02", txn.Date)
		transactions = append(transactions, Transaction{
			ID:             txn.ID,
			AccountID:      txn.AccountID,
			Amount:         txn.Amount,
			Date:           date,
			Name:           txn.Name,
			MerchantName:   txn.MerchantName,
			Category:       txn.Category,
			Pending:        txn.Pending,
			PaymentChannel: txn.PaymentChannel,
		})
	}

	return transactions, nil
}

// DetectRecurringTransactions analyzes transactions to find recurring subscriptions
func (s *PlaidService) DetectRecurringTransactions(transactions []Transaction) []RecurringSubscription {
	// Group transactions by merchant and amount
	groups := make(map[string][]Transaction)

	for _, txn := range transactions {
		// Skip pending transactions and income
		if txn.Pending || txn.Amount < 0 {
			continue
		}

		// Create a key based on merchant name and amount
		key := fmt.Sprintf("%s_%.2f", strings.ToLower(txn.MerchantName), txn.Amount)
		groups[key] = append(groups[key], txn)
	}

	var subscriptions []RecurringSubscription

	for _, txns := range groups {
		// Need at least 2 transactions to detect a pattern
		if len(txns) < 2 {
			continue
		}

		// Check if transactions occur at regular intervals
		if isRecurring(txns) {
			subscription := RecurringSubscription{
				MerchantName: txns[0].MerchantName,
				Amount:       txns[0].Amount,
				Frequency:    determineFrequency(txns),
				LastCharge:   txns[0].Date,
				NextDue:      predictNextDue(txns),
				Category:     txns[0].Category,
			}
			subscriptions = append(subscriptions, subscription)
		}
	}

	return subscriptions
}

// RecurringSubscription represents a detected recurring subscription
type RecurringSubscription struct {
	MerchantName string    `json:"merchant_name"`
	Amount       float64   `json:"amount"`
	Frequency    string    `json:"frequency"` // "monthly", "weekly", etc.
	LastCharge   time.Time `json:"last_charge"`
	NextDue      time.Time `json:"next_due"`
	Category     []string  `json:"category"`
}

// Helper function to make HTTP requests to Plaid API
func (s *PlaidService) makeRequest(ctx context.Context, endpoint string, payload interface{}, result interface{}) (interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+endpoint, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plaid API error: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper functions for recurring transaction detection
func isRecurring(transactions []Transaction) bool {
	if len(transactions) < 2 {
		return false
	}

	// Sort transactions by date
	for i := 0; i < len(transactions)-1; i++ {
		for j := i + 1; j < len(transactions); j++ {
			if transactions[i].Date.Before(transactions[j].Date) {
				transactions[i], transactions[j] = transactions[j], transactions[i]
			}
		}
	}

	// Check if intervals between transactions are consistent
	intervals := make([]int, 0)
	for i := 1; i < len(transactions); i++ {
		days := int(transactions[i-1].Date.Sub(transactions[i].Date).Hours() / 24)
		intervals = append(intervals, days)
	}

	// Check if intervals are similar (within 5 days tolerance)
	if len(intervals) < 1 {
		return false
	}

	avgInterval := 0
	for _, interval := range intervals {
		avgInterval += interval
	}
	avgInterval /= len(intervals)

	for _, interval := range intervals {
		if abs(interval-avgInterval) > 5 {
			return false
		}
	}

	return true
}

func determineFrequency(transactions []Transaction) string {
	if len(transactions) < 2 {
		return "unknown"
	}

	// Calculate average interval
	totalDays := int(transactions[0].Date.Sub(transactions[len(transactions)-1].Date).Hours() / 24)
	avgDays := totalDays / (len(transactions) - 1)

	if avgDays <= 8 {
		return "weekly"
	} else if avgDays <= 35 {
		return "monthly"
	} else if avgDays <= 95 {
		return "quarterly"
	} else {
		return "yearly"
	}
}

func predictNextDue(transactions []Transaction) time.Time {
	if len(transactions) < 2 {
		return time.Now()
	}

	// Calculate average interval
	totalDays := int(transactions[0].Date.Sub(transactions[len(transactions)-1].Date).Hours() / 24)
	avgDays := totalDays / (len(transactions) - 1)

	return transactions[0].Date.AddDate(0, 0, avgDays)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
