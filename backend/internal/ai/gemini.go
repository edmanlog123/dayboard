package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// GeminiService handles Gemini AI API operations
type GeminiService struct {
	apiKey  string
	baseURL string
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content represents the content of a message
type Content struct {
	Parts []Part `json:"parts"`
}

// Part represents a part of the content
type Part struct {
	Text string `json:"text"`
}

// GeminiResponse represents the response from Gemini API
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content Content `json:"content"`
}

// NewGeminiService creates a new Gemini AI service
func NewGeminiService() *GeminiService {
	return &GeminiService{
		apiKey:  os.Getenv("GEMINI_API_KEY"),
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent",
	}
}

// GenerateAdvice generates career advice using Gemini AI
func (s *GeminiService) GenerateAdvice(ctx context.Context, query string, userContext map[string]interface{}) (string, error) {
	if s.apiKey == "" {
		// Return demo response if no API key
		return s.getDemoResponse(query), nil
	}

	// Build context-aware prompt
	prompt := s.buildPrompt(query, userContext)

	request := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?key=%s", s.baseURL, s.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error: %s", resp.Status)
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// buildPrompt creates a context-aware prompt for the AI
func (s *GeminiService) buildPrompt(query string, userContext map[string]interface{}) string {
	var contextInfo strings.Builder

	// Add user context if available
	if profile, ok := userContext["profile"].(map[string]interface{}); ok {
		if state, ok := profile["state"].(string); ok {
			contextInfo.WriteString(fmt.Sprintf("User is located in %s. ", state))
		}
		if hourly, ok := profile["hourly_cents"].(int); ok {
			contextInfo.WriteString(fmt.Sprintf("User earns $%.2f/hour. ", float64(hourly)/100))
		}
	}

	if subscriptions, ok := userContext["subscriptions"].([]interface{}); ok {
		contextInfo.WriteString(fmt.Sprintf("User has %d active subscriptions. ", len(subscriptions)))
	}

	// Build the full prompt
	prompt := fmt.Sprintf(`You are a career advisor for college students and recent graduates. 
You specialize in internships, job searching, salary negotiation, and financial planning.

Context: %s

User Question: %s

Please provide specific, actionable advice. If the question is about:
- Salary negotiation: Include specific tactics and market rates
- Interview prep: Provide concrete tips and common questions
- Financial planning: Give practical budgeting advice for students
- Career decisions: Consider location, cost of living, and growth opportunities

Keep your response concise but informative (2-3 paragraphs max).`,
		contextInfo.String(), query)

	return prompt
}

// getDemoResponse returns demo responses when no API key is available
func (s *GeminiService) getDemoResponse(query string) string {
	query = strings.ToLower(query)

	if strings.Contains(query, "salary") || strings.Contains(query, "negotiation") {
		return `For salary negotiation, research market rates on Glassdoor and Levels.fyi first. When negotiating, focus on your value-add and use phrases like "Based on my research, the market rate for this role is..." Start with a number 10-15% above your target. Also negotiate beyond base salary - consider signing bonuses, equity, PTO, and professional development budgets.

For internships, many companies have fixed pay scales, but you can still negotiate start date, return offer terms, or additional mentorship opportunities. Remember that your first offer sets the baseline for future negotiations in your career.`
	}

	if strings.Contains(query, "interview") {
		return `Prepare for behavioral questions using the STAR method (Situation, Task, Action, Result). Practice your elevator pitch and have specific examples ready that showcase problem-solving, leadership, and technical skills.

For technical interviews, solve problems out loud to show your thinking process. For product/business cases, clarify assumptions first and structure your response. Always prepare thoughtful questions about the role, team, and company culture - this shows genuine interest and helps you evaluate if it's the right fit.`
	}

	if strings.Contains(query, "budget") || strings.Contains(query, "financial") {
		return `As a student, follow the 50/30/20 rule adapted for your situation: 50% for needs (tuition, rent, food), 30% for wants (entertainment, dining out), and 20% for savings/emergency fund. Track your subscriptions - they add up quickly!

For internships in expensive cities, factor in housing, transportation, and food costs when evaluating offers. A higher salary in SF might net less than a lower salary in Austin after cost of living adjustments. Use your DayBoard app to track daily expenses and see how location impacts your take-home pay.`
	}

	if strings.Contains(query, "location") || strings.Contains(query, "city") {
		return `When choosing between cities for internships or jobs, consider total compensation vs. cost of living. Texas has no state income tax, making a $70k salary equivalent to ~$77k in California. Factor in housing costs, transportation, and quality of life.

For tech roles, consider emerging hubs like Austin, Denver, or Atlanta alongside traditional markets. You'll often get more bang for your buck and better work-life balance while still accessing great opportunities and professional networks.`
	}

	// Default response
	return `I'd be happy to help with your career question! For the most personalized advice, I'd recommend providing more context about your situation, career goals, and specific challenges you're facing.

I can help with salary negotiation, interview preparation, financial planning, career decisions, and job search strategies. Feel free to ask about specific companies, roles, or situations you're navigating. The more details you provide, the better I can tailor my advice to your unique circumstances.`
}
