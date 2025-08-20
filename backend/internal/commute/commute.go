package commute

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// Estimate represents the output of a commute cost estimate. Distances and
// durations are included along with low/high cost estimates (in cents).
type Estimate struct {
	DistanceMiles    float64 `json:"distanceMiles"`
	DurationMinutes  float64 `json:"durationMinutes"`
	EstCostLowCents  int     `json:"estCostLowCents"`
	EstCostHighCents int     `json:"estCostHighCents"`
}

// estimateDistance calls the Google Distance Matrix API to compute the
// distance and duration between two addresses. It returns miles and
// minutes. The API key must be set via MAPS_API_KEY environment
// variable. This function is blocking and should be called from a
// goroutine or asynchronous context if latency is a concern.
func estimateDistance(ctx context.Context, origin, destination string) (float64, float64, error) {
	apiKey := os.Getenv("MAPS_API_KEY")
	if apiKey == "" {
		return 0, 0, fmt.Errorf("MAPS_API_KEY environment variable not set")
	}
	endpoint := "https://maps.googleapis.com/maps/api/distancematrix/json"
	params := url.Values{}
	params.Set("origins", origin)
	params.Set("destinations", destination)
	params.Set("units", "imperial")
	params.Set("key", apiKey)
	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	var dmResp struct {
		Rows []struct {
			Elements []struct {
				Distance struct {
					Value int    `json:"value"` // meters
					Text  string `json:"text"`
				} `json:"distance"`
				Duration struct {
					Value int    `json:"value"` // seconds
					Text  string `json:"text"`
				} `json:"duration"`
				Status string `json:"status"`
			} `json:"elements"`
		} `json:"rows"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dmResp); err != nil {
		return 0, 0, err
	}
	if dmResp.Status != "OK" || len(dmResp.Rows) == 0 || len(dmResp.Rows[0].Elements) == 0 {
		return 0, 0, fmt.Errorf("distance matrix API error: %s", dmResp.Status)
	}
	elem := dmResp.Rows[0].Elements[0]
	if elem.Status != "OK" {
		return 0, 0, fmt.Errorf("distance matrix element error: %s", elem.Status)
	}
	// Convert meters to miles and seconds to minutes.
	miles := float64(elem.Distance.Value) * 0.000621371
	minutes := float64(elem.Duration.Value) / 60.0
	return miles, minutes, nil
}

// EstimateCommute calculates the commute cost between origin and destination
// given a surge factor. The cost is computed based on a simple model:
// base fare + per-mile * miles + per-minute * minutes. The cost model
// parameters should be stored in a DB table (city_cost_models) and loaded
// by the caller. For demonstration, this function accepts the cost
// parameters directly.
func EstimateCommute(ctx context.Context, origin, destination string, baseCents, perMileCents, perMinCents int, surge float64) (*Estimate, error) {
	miles, minutes, err := estimateDistance(ctx, origin, destination)
	if err != nil {
		return nil, err
	}
	low := float64(baseCents) + float64(perMileCents)*miles + float64(perMinCents)*minutes
	high := low * surge
	return &Estimate{
		DistanceMiles:    miles,
		DurationMinutes:  minutes,
		EstCostLowCents:  int(low),
		EstCostHighCents: int(high),
	}, nil
}
