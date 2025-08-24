package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"dayboard/backend/internal/ai"
	"dayboard/backend/internal/auth"
	"dayboard/backend/internal/commute"
	"dayboard/backend/internal/db"
	"dayboard/backend/internal/estimate"
	"dayboard/backend/internal/google"
	"dayboard/backend/internal/plaid"
	"dayboard/backend/internal/store"
)

// In-memory demo data (used only when DEMO_MODE is enabled)
var (
	demoSubs         []store.Subscription
	demoEvents       []store.Event
	demoProfile      store.Profile
	demoCommutes     []CommuteEntry
	demoEmails       EmailSummary
	demoStateTax     []StateTaxComparison
	demoHousing      []HousingComparison
	demoCampusEvents []CampusEvent
	demoSeeded       bool
)

type CommuteEntry struct {
	ID        uuid.UUID `json:"id"`
	Date      time.Time `json:"date"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	CostCents int       `json:"costCents"`
	Method    string    `json:"method"`
}

type EmailSummary struct {
	UnreadCount int      `json:"unreadCount"`
	TopSubjects []string `json:"topSubjects"`
}

type StateTaxComparison struct {
	State       string  `json:"state"`
	TaxRate     float64 `json:"taxRate"`
	NetPayCents int     `json:"netPayCents"`
}

type HousingComparison struct {
	City              string `json:"city"`
	AvgRentCents      int    `json:"avgRentCents"`
	NetAfterRentCents int    `json:"netAfterRentCents"`
}

type CampusEvent struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Date     time.Time `json:"date"`
	Location string    `json:"location"`
	Category string    `json:"category"`
}

// main is the entrypoint for the DayBoard backend. It sets up the HTTP router
// and starts listening on the port specified in the PORT environment variable.
func main() {
	// Determine the port to listen on. Default to 8080 if not set.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Demo mode allows running without a database or external API keys.
	demoMode := strings.EqualFold(os.Getenv("DEMO_MODE"), "true") || os.Getenv("DEMO_MODE") == "1"

	// Use Gin in release mode for production. Gin automatically logs requests.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Register health check endpoint for uptime monitoring.
	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Mount API routes under /api/v1.
	api := router.Group("/api/v1")

	// Initialize JWT manager and auth handlers (works in both demo and production mode)
	jwtManager := auth.NewJWTManager()

	// Auth routes
	authGroup := api.Group("/auth")

	// Demo auth endpoints that return mock responses
	authGroup.POST("/signup", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{
			"token": "demo_jwt_token_for_testing",
			"user": gin.H{
				"id":    "demo-user-123",
				"email": "demo@dayboard.app",
				"name":  "Demo User",
			},
		})
	})
	authGroup.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"token": "demo_jwt_token_for_testing",
			"user": gin.H{
				"id":    "demo-user-123",
				"email": "demo@dayboard.app",
				"name":  "Demo User",
			},
		})
	})

	if demoMode {
		// Seed demo data once at startup
		if !demoSeeded {
			seedDemoData()
			demoSeeded = true
		}

		// In demo mode, serve persistent dummy data so the app is fully usable without
		// DATABASE_URL, MAPS_API_KEY, or other external credentials.
		api.GET("/agenda/today", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoEvents)
		})

		api.POST("/agenda/today", func(c *gin.Context) {
			var req store.Event
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.ID == uuid.Nil {
				req.ID = uuid.New()
			}
			demoEvents = append(demoEvents, req)
			c.JSON(http.StatusCreated, req)
		})

		api.GET("/subs", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoSubs)
		})

		api.POST("/subs", func(c *gin.Context) {
			var req store.Subscription
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			req.ID = uuid.New()
			if req.Source == "" {
				req.Source = "manual"
			}
			req.IsActive = true
			demoSubs = append(demoSubs, req)
			c.JSON(http.StatusCreated, req)
		})

		// Demo: accept delete requests and return success so client can simulate removal.
		api.DELETE("/subs/:id", func(c *gin.Context) {
			idStr := c.Param("id")
			for i, s := range demoSubs {
				if s.ID.String() == idStr {
					demoSubs = append(demoSubs[:i], demoSubs[i+1:]...)
					c.Status(http.StatusNoContent)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})

		api.GET("/profile", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoProfile)
		})

		api.POST("/profile", func(c *gin.Context) {
			var prof store.Profile
			if err := c.BindJSON(&prof); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			demoProfile = prof
			c.JSON(http.StatusCreated, prof)
		})

		// Email summary endpoint
		api.GET("/email/summary", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoEmails)
		})

		// Commute entries
		api.GET("/commute/entries", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoCommutes)
		})

		api.POST("/commute/entries", func(c *gin.Context) {
			var req CommuteEntry
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.ID == uuid.Nil {
				req.ID = uuid.New()
			}
			if req.Date.IsZero() {
				req.Date = time.Now().UTC()
			}
			demoCommutes = append(demoCommutes, req)
			c.JSON(http.StatusCreated, req)
		})

		// Today's burn calculation
		api.GET("/daily/burn", func(c *gin.Context) {
			today := time.Now().UTC()
			var totalCents int

			// Add subscriptions due today
			for _, sub := range demoSubs {
				if sub.NextDue != nil && isSameDay(*sub.NextDue, today) {
					totalCents += sub.AmountCents
				}
			}

			// Add commute costs for today
			for _, commute := range demoCommutes {
				if isSameDay(commute.Date, today) {
					totalCents += commute.CostCents
				}
			}

			// Add food cost if it's an office day (simplified: assume today is office day)
			totalCents += demoProfile.FoodCostCents

			c.JSON(http.StatusOK, gin.H{
				"totalCents": totalCents,
				"breakdown": gin.H{
					"subscriptions": getSubsDueToday(),
					"commutes":      getCommutesToday(),
					"food":          demoProfile.FoodCostCents,
				},
			})
		})

		// Finance comparison endpoints
		api.GET("/finance/state-comparison", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoStateTax)
		})

		api.GET("/finance/housing-comparison", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoHousing)
		})

		// Campus events endpoint
		api.GET("/campus/events", func(c *gin.Context) {
			c.JSON(http.StatusOK, demoCampusEvents)
		})

		// AI advice endpoint (demo responses)
		api.POST("/ai/advice", func(c *gin.Context) {
			var req struct {
				Query string `json:"query"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Demo AI responses based on query keywords
			advice := "I'm a demo AI assistant. "
			if strings.Contains(strings.ToLower(req.Query), "salary") {
				advice += "For internship salary negotiation: Research market rates, highlight your skills, and be confident but respectful. Consider the total compensation package including benefits and learning opportunities."
			} else if strings.Contains(strings.ToLower(req.Query), "interview") {
				advice += "For interviews: Practice coding problems, prepare STAR method stories, research the company, ask thoughtful questions, and follow up professionally."
			} else {
				advice += "I can help with internship advice, salary negotiation, interview tips, and financial planning. What specific area would you like guidance on?"
			}
			c.JSON(http.StatusOK, gin.H{"advice": advice})
		})

		api.POST("/estimate/taxes", func(c *gin.Context) {
			// Parse payload {incomeCents,state,filingStatus,payFreq,termWeeks}
			var body struct {
				IncomeCents  int    `json:"incomeCents"`
				State        string `json:"state"`
				FilingStatus string `json:"filingStatus"`
				PayFreq      string `json:"payFreq"`
				TermWeeks    int    `json:"termWeeks"`
			}
			if err := c.BindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// Very simple demo tax model: std deduction + flat rates.
			stdDeduction := 1385000 // $13,850.00 in cents
			taxable := body.IncomeCents - stdDeduction
			if taxable < 0 {
				taxable = 0
			}
			federal := taxable * 22 / 100 // 22%
			state := taxable * 5 / 100    // 5%
			fica := body.IncomeCents * 765 / 10000
			totalTax := federal + state + fica
			netAnnual := body.IncomeCents - totalTax
			checks := 0
			switch body.PayFreq {
			case "weekly":
				checks = body.TermWeeks
			case "biweekly":
				checks = body.TermWeeks / 2
			case "monthly":
				checks = body.TermWeeks / 4
			default:
				checks = body.TermWeeks / 2
			}
			perPay := 0
			if checks > 0 {
				perPay = netAnnual / checks
			}
			c.JSON(http.StatusOK, gin.H{
				"federalCents":        federal,
				"stateCents":          state,
				"ficaCents":           fica,
				"perPaycheckNetCents": perPay,
				"termNetCents":        netAnnual,
			})
		})

		api.GET("/commute/estimate", func(c *gin.Context) {
			// Provide a fixed demo estimate without calling external APIs.
			surge := 1.0
			if s := c.Query("surge"); s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil {
					surge = v
				}
			}
			miles := 3.2
			minutes := 14.0
			baseCents := 200
			perMileCents := 150
			perMinCents := 25
			low := float64(baseCents) + float64(perMileCents)*miles + float64(perMinCents)*minutes
			high := low * surge
			c.JSON(http.StatusOK, gin.H{
				"distanceMiles":    miles,
				"durationMinutes":  minutes,
				"estCostLowCents":  int(low),
				"estCostHighCents": int(high),
			})
		})
	} else {
		// Initialize DB connection. Fatal if cannot connect.
		database := db.New()
		defer database.Close()

		// Initialize auth handlers for production
		authHandlers := auth.NewAuthHandlers(database, jwtManager)
		authGroup.POST("/signup", authHandlers.Signup)
		authGroup.POST("/login", authHandlers.Login)
		authGroup.GET("/profile", auth.AuthMiddleware(jwtManager), authHandlers.GetProfile)
		authGroup.POST("/refresh", authHandlers.RefreshToken)

		// Initialize OAuth handlers
		googleHandlers := google.NewOAuthHandlers(database)
		plaidHandlers := plaid.NewOAuthHandlers(database)
		geminiService := ai.NewGeminiService()

		// Google Calendar OAuth routes
		googleGroup := api.Group("/google", auth.AuthMiddleware(jwtManager))
		googleGroup.GET("/auth", googleHandlers.InitiateGoogleAuth)
		googleGroup.GET("/callback", googleHandlers.HandleGoogleCallback)
		googleGroup.POST("/sync", googleHandlers.SyncCalendarEvents)

		// Plaid OAuth routes
		plaidGroup := api.Group("/plaid", auth.AuthMiddleware(jwtManager))
		plaidGroup.POST("/link-token", plaidHandlers.CreateLinkToken)
		plaidGroup.POST("/exchange", plaidHandlers.ExchangePublicToken)
		plaidGroup.POST("/sync", plaidHandlers.SyncTransactions)
		plaidGroup.GET("/accounts", plaidHandlers.GetConnectedAccounts)

		// AI Assistant route with real Gemini integration
		api.POST("/ai/advice", auth.OptionalAuthMiddleware(jwtManager), func(c *gin.Context) {
			var req struct {
				Query string `json:"query" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Get user context for personalized advice
			userContext := make(map[string]interface{})
			if userID, exists := auth.GetUserIDFromContext(c); exists {
				// Get user profile for context
				if profile, err := store.GetProfile(c.Request.Context(), database, userID); err == nil && profile != nil {
					userContext["profile"] = map[string]interface{}{
						"state":        profile.State,
						"hourly_cents": profile.HourlyCents,
					}
				}
				// Get subscriptions for context
				if subs, err := store.GetSubscriptions(c.Request.Context(), database, userID); err == nil {
					userContext["subscriptions"] = subs
				}
			}

			advice, err := geminiService.GenerateAdvice(c.Request.Context(), req.Query, userContext)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate advice"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"advice": advice})
		})

		api.GET("/agenda/today", func(c *gin.Context) {
			// In a production system you'd derive the user ID from the
			// authenticated session. For demonstration we read a query param.
			userParam := c.Query("user_id")
			userID := uuid.Nil
			if userParam != "" {
				if uid, err := uuid.Parse(userParam); err == nil {
					userID = uid
				}
			}
			// Determine start and end of today in UTC based on the server's time.
			now := time.Now().UTC()
			y, m, d := now.Date()
			loc := now.Location()
			startOfDay := time.Date(y, m, d, 0, 0, 0, 0, loc)
			endOfDay := startOfDay.Add(24 * time.Hour)
			events, err := store.GetTodayEvents(c.Request.Context(), database, userID, startOfDay, endOfDay)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// Transform events into response objects. Gin will marshal the
			// time.Time fields as RFC3339 strings.
			c.JSON(http.StatusOK, events)
		})

		api.GET("/subs", func(c *gin.Context) {
			userParam := c.Query("user_id")
			userID := uuid.Nil
			if uid, err := uuid.Parse(userParam); err == nil {
				userID = uid
			}
			subs, err := store.GetSubscriptions(c.Request.Context(), database, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, subs)
		})

		api.POST("/subs", func(c *gin.Context) {
			userParam := c.Query("user_id")
			userID := uuid.Nil
			if uid, err := uuid.Parse(userParam); err == nil {
				userID = uid
			}
			var req store.Subscription
			if err := c.BindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			sub, err := store.CreateSubscription(c.Request.Context(), database, userID, req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, sub)
		})

		// TODO: Implement real delete in DB. For demo, return 204.
		api.DELETE("/subs/:id", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		api.POST("/estimate/taxes", func(c *gin.Context) {
			// Parse payload {incomeCents,state,filingStatus,payFreq,termWeeks}
			var body struct {
				IncomeCents  int    `json:"incomeCents"`
				State        string `json:"state"`
				FilingStatus string `json:"filingStatus"`
				PayFreq      string `json:"payFreq"`
				TermWeeks    int    `json:"termWeeks"`
			}
			if err := c.BindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			// Use current year for taxes. In production you might allow specifying.
			year := time.Now().Year()
			res, err := estimate.EstimateTaxes(c.Request.Context(), database, body.IncomeCents, body.State, body.FilingStatus, year, body.PayFreq, body.TermWeeks)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		api.GET("/commute/estimate", func(c *gin.Context) {
			origin := c.Query("from")
			destination := c.Query("to")
			// Example surge parameter, default to 1.0 (no surge)
			surge := 1.0
			if s := c.Query("surge"); s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil {
					surge = v
				}
			}
			// For demonstration, fetch cost model from DB based on city. Here
			// we simply hardcode a generic model. In production, you would
			// select by city/state.
			baseCents := 200    // $2 base fare
			perMileCents := 150 // $1.50 per mile
			perMinCents := 25   // $0.25 per minute
			est, err := commute.EstimateCommute(c.Request.Context(), origin, destination, baseCents, perMileCents, perMinCents, surge)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, est)
		})

		api.GET("/profile", func(c *gin.Context) {
			userParam := c.Query("user_id")
			userID := uuid.Nil
			if uid, err := uuid.Parse(userParam); err == nil {
				userID = uid
			}
			prof, err := store.GetProfile(c.Request.Context(), database, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if prof == nil {
				c.JSON(http.StatusOK, gin.H{})
				return
			}
			c.JSON(http.StatusOK, prof)
		})

		api.POST("/profile", func(c *gin.Context) {
			userParam := c.Query("user_id")
			userID := uuid.Nil
			if uid, err := uuid.Parse(userParam); err == nil {
				userID = uid
			}
			var prof store.Profile
			if err := c.BindJSON(&prof); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			prof.UserID = userID
			if err := store.UpsertProfile(c.Request.Context(), database, prof); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, prof)
		})
	}

	// Start listening and serving requests. If an error occurs, log and exit.
	if err := router.Run(fmt.Sprintf(":" + port)); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func seedDemoData() {
	now := time.Now().UTC()

	// Seed events
	start := now.Add(30 * time.Minute)
	end := start.Add(45 * time.Minute)
	demoEvents = []store.Event{
		{ID: uuid.New(), Start: start, End: end, Title: "Standup", JoinURL: "https://meet.google.com/xyz-standup", Location: "Remote"},
		{ID: uuid.New(), Start: end.Add(90 * time.Minute), End: end.Add(150 * time.Minute), Title: "Project Sync", JoinURL: "https://zoom.us/j/123456789", Location: "Remote"},
	}

	// Seed subscriptions
	next := now.Add(24 * time.Hour)
	next2 := now.Add(6 * 24 * time.Hour)
	demoSubs = []store.Subscription{
		{ID: uuid.New(), Merchant: "Spotify", AmountCents: 999, CadenceDays: 30, NextDue: ptrTime(next), Source: "manual", IsActive: true},
		{ID: uuid.New(), Merchant: "Notion", AmountCents: 800, CadenceDays: 30, NextDue: ptrTime(next2), Source: "manual", IsActive: true},
		{ID: uuid.New(), Merchant: "Netflix", AmountCents: 1599, CadenceDays: 30, NextDue: ptrTime(now), Source: "plaid", IsActive: true}, // Due today
	}

	// Seed profile
	hourly := 2500
	hours := 40
	startDate := now.AddDate(0, -1, 0)
	demoProfile = store.Profile{
		UserID:        uuid.Nil,
		HomeAddr:      "123 Main St, Indianapolis, IN",
		OfficeAddr:    "456 Company Rd, Indianapolis, IN",
		City:          "Indianapolis",
		State:         "IN",
		HourlyCents:   &hourly,
		HoursPerWeek:  &hours,
		PayFreq:       "biweekly",
		StartDate:     &startDate,
		InOfficeDays:  3,
		FoodCostCents: 1200, // $12 lunch
	}

	// Seed commute entries
	demoCommutes = []CommuteEntry{
		{ID: uuid.New(), Date: now, From: "Home", To: "Office", CostCents: 1250, Method: "Uber"},
	}

	// Seed email summary
	demoEmails = EmailSummary{
		UnreadCount: 7,
		TopSubjects: []string{"Weekly Team Update", "Action Required: Submit Timesheet", "Lunch & Learn Tomorrow"},
	}

	// Seed state tax comparisons (demo data for popular internship states)
	baseIncome := 52000 * 100 // $52k annual
	demoStateTax = []StateTaxComparison{
		{State: "CA", TaxRate: 9.3, NetPayCents: int(float64(baseIncome) * 0.677)}, // High tax
		{State: "TX", TaxRate: 0.0, NetPayCents: int(float64(baseIncome) * 0.765)}, // No state tax
		{State: "NY", TaxRate: 6.5, NetPayCents: int(float64(baseIncome) * 0.705)},
		{State: "WA", TaxRate: 0.0, NetPayCents: int(float64(baseIncome) * 0.765)}, // No state tax
		{State: "IN", TaxRate: 3.23, NetPayCents: int(float64(baseIncome) * 0.735)},
	}

	// Seed housing comparisons (popular tech cities)
	demoHousing = []HousingComparison{
		{City: "San Francisco, CA", AvgRentCents: 350000, NetAfterRentCents: int(float64(baseIncome)*0.677) - 350000},
		{City: "Austin, TX", AvgRentCents: 180000, NetAfterRentCents: int(float64(baseIncome)*0.765) - 180000},
		{City: "Seattle, WA", AvgRentCents: 220000, NetAfterRentCents: int(float64(baseIncome)*0.765) - 220000},
		{City: "Indianapolis, IN", AvgRentCents: 120000, NetAfterRentCents: int(float64(baseIncome)*0.735) - 120000},
		{City: "Raleigh, NC", AvgRentCents: 140000, NetAfterRentCents: int(float64(baseIncome)*0.725) - 140000},
	}

	// Seed campus events
	demoCampusEvents = []CampusEvent{
		{ID: uuid.New(), Title: "Career Fair", Date: now.Add(48 * time.Hour), Location: "Student Union", Category: "Career"},
		{ID: uuid.New(), Title: "Basketball vs State", Date: now.Add(72 * time.Hour), Location: "Arena", Category: "Sports"},
		{ID: uuid.New(), Title: "Tech Talk: AI in Finance", Date: now.Add(120 * time.Hour), Location: "Engineering Building", Category: "Academic"},
		{ID: uuid.New(), Title: "Spring Concert", Date: now.Add(168 * time.Hour), Location: "Outdoor Stage", Category: "Entertainment"},
	}
}

func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func getSubsDueToday() []store.Subscription {
	today := time.Now().UTC()
	var result []store.Subscription
	for _, sub := range demoSubs {
		if sub.NextDue != nil && isSameDay(*sub.NextDue, today) {
			result = append(result, sub)
		}
	}
	return result
}

func getCommutesToday() []CommuteEntry {
	today := time.Now().UTC()
	var result []CommuteEntry
	for _, commute := range demoCommutes {
		if isSameDay(commute.Date, today) {
			result = append(result, commute)
		}
	}
	return result
}
