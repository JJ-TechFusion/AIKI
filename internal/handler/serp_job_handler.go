package handler

import (
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type SerpJobHandler struct {
	serpService service.SerpJobService
}

func NewSerpJobHandler(serpService service.SerpJobService) *SerpJobHandler {
	return &SerpJobHandler{serpService: serpService}
}

// GetRecommendedJobs godoc
// @Summary      Get recommended jobs
// @Description  Fetches jobs from SerpApi based on user profile (job title + experience level). Optional query param `location` (e.g. Austin, TX or United States) is passed to SerpApi as the job location filter. Returns cached results if fetched within 24 hours for the same location.
// @Tags         job-search
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=domain.JobSearchResult}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Param        location query string false "Job location filter (e.g. Seattle, WA) — forwarded to SerpApi Google Jobs"
// @Router       /jobs/recommended [get]
func (h *SerpJobHandler) GetRecommendedJobs(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	location := strings.TrimSpace(c.QueryParam("location"))
	result, err := h.serpService.GetJobsForUser(c.Request().Context(), userID, location)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "jobs retrieved", result)
}

// SaveJobToTracker godoc
// @Summary      Save a recommended job to tracker
// @Description  Saves a job from the recommended list into the user's job tracker
// @Tags         job-search
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Cache job ID"
// @Success      201 {object} response.Response{data=domain.Job}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /jobs/recommended/{id}/save [post]
func (h *SerpJobHandler) SaveJobToTracker(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	idParam := c.Param("id")
	cacheID, err := strconv.Atoi(idParam)
	if err != nil {
		return response.ValidationError(c, "invalid job id")
	}

	job, err := h.serpService.SaveJobToTracker(c.Request().Context(), userID, int32(cacheID))
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusCreated, "job saved to tracker", job)
}

// ApplyRecommendedJob godoc
// @Summary      Record an application and get the apply URL
// @Description  Creates or updates a tracker row with status applied, links the Serp cache row, and returns apply_url (Google Jobs share link) for opening in an in-app browser or external browser.
// @Tags         job-search
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Cache job ID"
// @Param        request body domain.DirectApplyRequest false "Optional notes"
// @Success      200 {object} response.Response{data=domain.DirectApplyResult}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /jobs/recommended/{id}/apply [post]
func (h *SerpJobHandler) ApplyRecommendedJob(c echo.Context) error {
	userID, ok := c.Get("user_id").(int32)
	if !ok {
		return response.Error(c, domain.ErrUnauthorized)
	}

	idParam := c.Param("id")
	cacheID, err := strconv.Atoi(idParam)
	if err != nil {
		return response.ValidationError(c, "invalid job id")
	}

	var req domain.DirectApplyRequest
	if err := c.Bind(&req); err != nil {
		return response.ValidationError(c, "invalid request body")
	}

	result, err := h.serpService.ApplyRecommendedJob(c.Request().Context(), userID, int32(cacheID), req.Notes)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, http.StatusOK, "apply recorded", result)
}
