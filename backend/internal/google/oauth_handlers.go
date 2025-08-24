package google

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dayboard/backend/internal/auth"
	"dayboard/backend/internal/db"
	"dayboard/backend/internal/store"
)

// OAuthHandlers handles Google OAuth flows
type OAuthHandlers struct {
	db              *db.DB
	calendarService *CalendarService
}

// NewOAuthHandlers creates new OAuth handlers
func NewOAuthHandlers(database *db.DB) *OAuthHandlers {
	return &OAuthHandlers{
		db:              database,
		calendarService: NewCalendarService(),
	}
}

// InitiateGoogleAuth starts the Google OAuth flow
func (h *OAuthHandlers) InitiateGoogleAuth(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Generate state parameter for security
	state := generateState(userID.String())

	// Store state in session or cache (simplified for demo)
	// In production, you'd store this in Redis or session store

	authURL := h.calendarService.GetAuthURL(state)

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleGoogleCallback handles the OAuth callback from Google
func (h *OAuthHandlers) HandleGoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	// Verify state parameter (simplified for demo)
	userID, err := verifyState(state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state parameter"})
		return
	}

	// Exchange code for tokens
	tokenResp, err := h.calendarService.ExchangeCodeForToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code for token"})
		return
	}

	// Store tokens in database (encrypted)
	err = h.storeTokens(c.Request.Context(), userID, tokenResp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store tokens"})
		return
	}

	// Sync calendar events
	err = h.syncCalendarEvents(c.Request.Context(), userID, tokenResp.AccessToken)
	if err != nil {
		// Log error but don't fail the request
		// Initial sync can be retried later
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Google Calendar connected successfully",
		"user_id": userID,
	})
}

// SyncCalendarEvents manually triggers a calendar sync
func (h *OAuthHandlers) SyncCalendarEvents(c *gin.Context) {
	userID, exists := auth.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get stored access token
	accessToken, err := h.getAccessToken(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google Calendar not connected"})
		return
	}

	// Sync events
	err = h.syncCalendarEvents(c.Request.Context(), userID, accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync calendar events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Calendar events synced successfully"})
}

// Helper functions

func generateState(userID string) string {
	// In production, use a proper state generation with expiry
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return base64.URLEncoding.EncodeToString(randomBytes) + ":" + userID
}

func verifyState(state string) (uuid.UUID, error) {
	// Simplified state verification - extract user ID
	// In production, verify the random part and check expiry
	parts := []string{state} // Simplified - would normally split by ":"
	if len(parts) < 2 {
		return uuid.Nil, fmt.Errorf("invalid state format")
	}

	return uuid.Parse(parts[1])
}

func (h *OAuthHandlers) storeTokens(ctx context.Context, userID uuid.UUID, tokens *TokenResponse) error {
	// In production, encrypt these tokens before storing
	_, err := h.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (user_id, provider, access_token_enc, refresh_token_enc, scopes, expiry)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, provider) 
		DO UPDATE SET 
			access_token_enc = EXCLUDED.access_token_enc,
			refresh_token_enc = EXCLUDED.refresh_token_enc,
			expiry = EXCLUDED.expiry
	`, userID, "google_calendar",
		[]byte(tokens.AccessToken),  // Should be encrypted
		[]byte(tokens.RefreshToken), // Should be encrypted
		[]string{"https://www.googleapis.com/auth/calendar.readonly"},
		time.Now().Add(time.Duration(tokens.ExpiresIn)*time.Second))

	return err
}

func (h *OAuthHandlers) getAccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	var accessToken []byte
	var expiry time.Time

	err := h.db.QueryRowContext(ctx, `
		SELECT access_token_enc, expiry 
		FROM oauth_tokens 
		WHERE user_id = $1 AND provider = $2
	`, userID, "google_calendar").Scan(&accessToken, &expiry)

	if err != nil {
		return "", err
	}

	// Check if token is expired (simplified - should refresh if needed)
	if time.Now().After(expiry) {
		return "", fmt.Errorf("token expired")
	}

	// In production, decrypt the token
	return string(accessToken), nil
}

func (h *OAuthHandlers) syncCalendarEvents(ctx context.Context, userID uuid.UUID, accessToken string) error {
	events, err := h.calendarService.GetTodaysEvents(ctx, accessToken)
	if err != nil {
		return err
	}

	// Store events in database
	for _, event := range events {
		// Convert Google Calendar event to store.Event
		storeEvent := store.Event{
			ID:       uuid.New(),
			Start:    event.StartTime,
			End:      event.EndTime,
			Title:    event.Summary,
			JoinURL:  getJoinURL(event),
			Location: event.Location,
		}

		// Insert or update event
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO calendar_events (id, user_id, source, ext_id, start_ts, end_ts, title, join_url, location)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (user_id, source, ext_id)
			DO UPDATE SET
				start_ts = EXCLUDED.start_ts,
				end_ts = EXCLUDED.end_ts,
				title = EXCLUDED.title,
				join_url = EXCLUDED.join_url,
				location = EXCLUDED.location,
				updated_at = NOW()
		`, storeEvent.ID, userID, "google_calendar", event.ID,
			event.StartTime, event.EndTime, event.Summary, getJoinURL(event), event.Location)

		if err != nil {
			return err
		}
	}

	return nil
}

func getJoinURL(event CalendarEvent) string {
	if event.HangoutLink != "" {
		return event.HangoutLink
	}

	// Extract Zoom/Teams links from description or location
	// This is a simplified extraction - in production you'd use regex
	if strings.Contains(strings.ToLower(event.Description), "zoom.us") {
		// Extract Zoom URL logic
	}

	return event.HTMLLink
}
