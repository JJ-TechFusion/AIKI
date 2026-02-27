package serp

import (
	"aiki/internal/domain"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const serpAPIBaseURL = "https://serpapi.com/search"
const cacheTTL = 24 * time.Hour

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// serpAPIResponse is the raw response from SerpApi Google Jobs
type serpAPIResponse struct {
	JobsResults []struct {
		JobID              string `json:"job_id"`
		Title              string `json:"title"`
		CompanyName        string `json:"company_name"`
		Location           string `json:"location"`
		Description        string `json:"description"`
		ShareLink          string `json:"share_link"`
		DetectedExtensions struct {
			PostedAt string `json:"posted_at"`
			Salary   string `json:"salary"`
		} `json:"detected_extensions"`
		ViaText string `json:"via"`
	} `json:"jobs_results"`
}

// FetchJobs calls SerpApi Google Jobs search based on job title and experience level
func (c *Client) FetchJobs(jobTitle, experienceLevel, location string) ([]domain.SerpJob, error) {
	query := buildQuery(jobTitle, experienceLevel)

	params := url.Values{}
	params.Set("engine", "google_jobs")
	params.Set("q", query)
	params.Set("api_key", c.apiKey)
	params.Set("num", "20")
	if location != "" {
		params.Set("location", location)
	}

	reqURL := fmt.Sprintf("%s?%s", serpAPIBaseURL, params.Encode())

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("serp api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("serp api returned status %d", resp.StatusCode)
	}

	var result serpAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode serp api response: %w", err)
	}

	now := time.Now()
	jobs := make([]domain.SerpJob, 0, len(result.JobsResults))
	for _, j := range result.JobsResults {
		jobs = append(jobs, domain.SerpJob{
			ExternalID:  j.JobID,
			Title:       j.Title,
			CompanyName: j.CompanyName,
			Location:    j.Location,
			Description: j.Description,
			Link:        j.ShareLink,
			Platform:    j.ViaText,
			PostedAt:    j.DetectedExtensions.PostedAt,
			Salary:      j.DetectedExtensions.Salary,
			FetchedAt:   now,
		})
	}

	return jobs, nil
}

// buildQuery constructs a natural language search query from profile data
func buildQuery(jobTitle, experienceLevel string) string {
	if experienceLevel == "" {
		return jobTitle
	}

	levelMap := map[string]string{
		"beginner":  "entry level",
		"junior":    "junior",
		"mid":       "mid level",
		"senior":    "senior",
		"lead":      "lead",
		"manager":   "manager",
		"executive": "executive",
	}

	level, ok := levelMap[experienceLevel]
	if !ok {
		level = experienceLevel
	}

	return fmt.Sprintf("%s %s jobs", level, jobTitle)
}
