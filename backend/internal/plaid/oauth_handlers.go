package plaid

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dayboard/backend/internal/auth"
	"dayboard/backend/internal/db"
	"dayboard/backend/internal/store"
)

// OAuthHandlers handles Plaid OAuth flows and transaction sync
type OAuthHandlers struct {
	db           *db.DB
	plaidService *PlaidService
}

// NewOAuthHandlers creates new Plaid OAuth handlers
func NewOAuthHandlers(database *db.DB) *OAuthHandlers {
	return &OAuthHandlers{
		db:           database,
		plaidService: NewPlaidService(),
	}
}

// CreateLinkToken creates a Plaid Link token for the frontend
func (h *OAuthHandlers) CreateLinkToken(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	linkTokenResp, err := h.plaidService.CreateLinkToken(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create link token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"link_token": linkTokenResp.LinkToken,
		"expiration": linkTokenResp.Expiration,
	})
}

// ExchangePublicToken exchanges a public token for an access token
func (h *OAuthHandlers) ExchangePublicToken(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		PublicToken string `json:"public_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Exchange public token for access token
	accessTokenResp, err := h.plaidService.ExchangePublicToken(c.Request.Context(), req.PublicToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange public token"})
		return
	}

	// Store access token in database (encrypted in production)
	err = h.storeAccessToken(c.Request.Context(), userID, accessTokenResp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store access token"})
		return
	}

	// Sync initial transactions and accounts
	err = h.syncAccountsAndTransactions(c.Request.Context(), userID, accessTokenResp.AccessToken)
	if err != nil {
		// Log error but don't fail the request - can retry sync later
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank account connected successfully",
		"item_id": accessTokenResp.ItemID,
	})
}

// SyncTransactions manually triggers a transaction sync
func (h *OAuthHandlers) SyncTransactions(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get stored access token
	accessToken, err := h.getAccessToken(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No bank account connected"})
		return
	}

	// Sync transactions and detect subscriptions
	err = h.syncAccountsAndTransactions(c.Request.Context(), userID, accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transactions synced successfully"})
}

// GetConnectedAccounts returns the user's connected bank accounts
func (h *OAuthHandlers) GetConnectedAccounts(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get stored access token
	accessToken, err := h.getAccessToken(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"accounts": []interface{}{}})
		return
	}

	// Get accounts from Plaid
	accounts, err := h.plaidService.GetAccounts(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"accounts": accounts})
}

// Helper functions

func (h *OAuthHandlers) storeAccessToken(ctx context.Context, userID uuid.UUID, tokenResp *AccessTokenResponse) error {
	// In production, encrypt the access token before storing
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (user_id, provider, access_token_enc, scopes, expiry)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, provider) 
		DO UPDATE SET 
			access_token_enc = EXCLUDED.access_token_enc
	`, userID, "plaid",
		[]byte(tokenResp.AccessToken), // Should be encrypted
		[]string{"transactions"},
		time.Now().Add(365*24*time.Hour)) // Plaid tokens don't expire like OAuth tokens

	return err
}

func (h *OAuthHandlers) getAccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	var accessToken []byte

	err := h.db.QueryRowContext(ctx, `
		SELECT access_token_enc 
		FROM oauth_tokens 
		WHERE user_id = $1 AND provider = $2
	`, userID, "plaid").Scan(&accessToken)

	if err != nil {
		return "", err
	}

	// In production, decrypt the token
	return string(accessToken), nil
}

func (h *OAuthHandlers) syncAccountsAndTransactions(ctx context.Context, userID uuid.UUID, accessToken string) error {
	// Get transactions from Plaid
	transactions, err := h.plaidService.GetTransactions(ctx, accessToken)
	if err != nil {
		return err
	}

	// Store raw transactions
	for _, txn := range transactions {
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO transactions (user_id, source, ext_id, txn_date, merchant, amount_cents, category, raw)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (user_id, source, ext_id) DO NOTHING
		`, userID, "plaid", txn.ID, txn.Date, txn.MerchantName,
			int(txn.Amount*100), txn.Category, nil) // Convert to cents

		if err != nil {
			return err
		}
	}

	// Detect recurring subscriptions
	subscriptions := h.plaidService.DetectRecurringTransactions(transactions)

	// Store detected subscriptions
	for _, sub := range subscriptions {
		subscription := store.Subscription{
			ID:          uuid.New(),
			Merchant:    sub.MerchantName,
			AmountCents: int(sub.Amount * 100), // Convert to cents
			CadenceDays: frequencyToDays(sub.Frequency),
			NextDue:     &sub.NextDue,
			Source:      "plaid",
			IsActive:    true,
		}

		_, err := store.CreateSubscription(ctx, h.db, userID, subscription)
		if err != nil {
			// Log error but continue with other subscriptions
			continue
		}
	}

	return nil
}

func frequencyToDays(frequency string) int {
	switch frequency {
	case "weekly":
		return 7
	case "monthly":
		return 30
	case "quarterly":
		return 90
	case "yearly":
		return 365
	default:
		return 30 // Default to monthly
	}
}
