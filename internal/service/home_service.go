package service

import (
	"aiki/internal/domain"
	"aiki/internal/pkg/response"
	"aiki/internal/repository"
	"context"
	"fmt"
	"time"
)

//go:generate mockgen -source=home_service.go -destination=mocks/mock_home_service.go -package=mocks

type HomeService interface {
	// Home screen aggregated data
	GetHomeScreenData(ctx context.Context, userID int32) (*domain.HomeScreenData, error)

	// Focus sessions
	StartSession(ctx context.Context, userID int32, req *domain.StartSessionRequest) (*domain.FocusSession, error)
	PauseSession(ctx context.Context, userID int32, sessionID int32, elapsed int32) (*domain.FocusSession, error)
	ResumeSession(ctx context.Context, userID int32, sessionID int32) (*domain.FocusSession, error)
	EndSession(ctx context.Context, userID int32, sessionID int32, elapsed int32, completed bool) (*domain.FocusSession, error)
	GetActiveSession(ctx context.Context, userID int32) (*domain.FocusSession, error)
	GetSessionHistory(ctx context.Context, userID int32, limit, offset int32) ([]domain.FocusSession, error)

	// Streak
	GetStreak(ctx context.Context, userID int32) (*domain.Streak, error)

	// Badges
	GetUserBadges(ctx context.Context, userID int32) ([]domain.UserBadge, error)
	GetAllBadges(ctx context.Context) ([]domain.BadgeDefinition, error)

	// Progress
	GetProgressSummary(ctx context.Context, userID int32, period string) (*domain.ProgressSummary, error)
}

type homeService struct {
	homeRepo     repository.HomeRepository
	notifService NotificationService
}

// NewHomeService now requires a NotificationService for firing notifications.
func NewHomeService(homeRepo repository.HomeRepository, notifService NotificationService) HomeService {
	return &homeService{homeRepo: homeRepo, notifService: notifService}
}

// ─────────────────────────────────────────
// Home screen aggregated data
// ─────────────────────────────────────────

func (s *homeService) GetHomeScreenData(ctx context.Context, userID int32) (*domain.HomeScreenData, error) {
	streak, err := s.homeRepo.GetStreak(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeSession, err := s.homeRepo.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())

	weeklyProgress, err := s.homeRepo.GetProgressSummary(ctx, userID, weekStart, now)
	if err != nil {
		return nil, err
	}
	weeklyProgress.Period = "weekly"

	badges, err := s.homeRepo.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	recentBadges := badges
	if len(recentBadges) > 6 {
		recentBadges = recentBadges[:6]
	}

	totalBadges, err := s.homeRepo.GetUserBadgeCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &domain.HomeScreenData{
		Streak:         streak,
		ActiveSession:  activeSession,
		WeeklyProgress: *weeklyProgress,
		RecentBadges:   recentBadges,
		TotalBadges:    totalBadges,
	}, nil
}

// ─────────────────────────────────────────
// Focus Sessions
// ─────────────────────────────────────────

func (s *homeService) StartSession(ctx context.Context, userID int32, req *domain.StartSessionRequest) (*domain.FocusSession, error) {
	existing, err := s.homeRepo.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, response.ErrSessionAlreadyActive
	}
	return s.homeRepo.CreateSession(ctx, userID, req.DurationSeconds)
}

func (s *homeService) PauseSession(ctx context.Context, userID int32, sessionID int32, elapsed int32) (*domain.FocusSession, error) {
	session, err := s.homeRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrUnauthorized
	}
	if session.Status != domain.SessionStatusActive {
		return nil, response.ErrInvalidSessionStatus
	}
	return s.homeRepo.UpdateSession(ctx, sessionID, elapsed, domain.SessionStatusPaused, nil)
}

func (s *homeService) ResumeSession(ctx context.Context, userID int32, sessionID int32) (*domain.FocusSession, error) {
	session, err := s.homeRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrUnauthorized
	}
	if session.Status != domain.SessionStatusPaused {
		return nil, response.ErrInvalidSessionStatus
	}
	return s.homeRepo.UpdateSession(ctx, sessionID, session.ElapsedSeconds, domain.SessionStatusActive, nil)
}

func (s *homeService) EndSession(ctx context.Context, userID int32, sessionID int32, elapsed int32, completed bool) (*domain.FocusSession, error) {
	session, err := s.homeRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	status := domain.SessionStatusAbandoned
	if completed {
		status = domain.SessionStatusCompleted
	}

	now := time.Now()
	updatedSession, err := s.homeRepo.UpdateSession(ctx, sessionID, elapsed, status, &now)
	if err != nil {
		return nil, err
	}

	// Run side effects in a goroutine so the response is returned immediately
	if completed {
		go s.handleSessionCompleted(context.Background(), userID, elapsed, now)
	}

	return updatedSession, nil
}

// handleSessionCompleted updates streak, progress, badges and fires notifications.
// Runs in a goroutine — errors are non-fatal.
func (s *homeService) handleSessionCompleted(ctx context.Context, userID int32, focusSeconds int32, completedAt time.Time) {
	// 1. Update daily progress
	today := time.Date(completedAt.Year(), completedAt.Month(), completedAt.Day(), 0, 0, 0, 0, completedAt.Location())
	_ = s.homeRepo.UpsertDailyProgress(ctx, userID, today, focusSeconds, 1)

	// 2. Update streak
	streak, err := s.homeRepo.GetStreak(ctx, userID)
	newStreakVal := int32(1)
	if err == nil {
		newStreakVal = s.calculateNewStreak(streak, today)
		updatedStreak, err := s.homeRepo.UpsertStreak(ctx, userID, newStreakVal, max32(newStreakVal, streak.LongestStreak), &today)
		if err == nil {
			streak = updatedStreak
		}
	}

	// 3. Notify session completed
	s.notifService.NotifySessionCompleted(ctx, userID, focusSeconds)

	// 4. Notify streak milestone if applicable
	s.notifService.NotifyStreakMilestone(ctx, userID, newStreakVal)

	// 5. Check badges and notify for each new one earned
	s.checkAndAwardBadges(ctx, userID, streak)
}

func (s *homeService) calculateNewStreak(streak *domain.Streak, today time.Time) int32 {
	if streak.LastSessionDate == nil {
		return 1
	}
	lastDate := *streak.LastSessionDate
	lastDate = time.Date(lastDate.Year(), lastDate.Month(), lastDate.Day(), 0, 0, 0, 0, lastDate.Location())
	diff := int(today.Sub(lastDate).Hours() / 24)
	switch diff {
	case 0:
		return streak.CurrentStreak
	case 1:
		return streak.CurrentStreak + 1
	default:
		return 1
	}
}

func (s *homeService) checkAndAwardBadges(ctx context.Context, userID int32, streak *domain.Streak) {
	definitions, err := s.homeRepo.GetAllBadgeDefinitions(ctx)
	if err != nil {
		return
	}

	now := time.Now()
	allTime, _ := s.homeRepo.GetProgressSummary(ctx, userID, time.Time{}, now)

	earnedBadges, _ := s.homeRepo.GetUserBadges(ctx, userID)
	earnedMap := make(map[int32]bool, len(earnedBadges))
	for _, b := range earnedBadges {
		earnedMap[b.BadgeID] = true
	}

	for _, def := range definitions {
		if earnedMap[def.ID] {
			continue
		}

		var earned bool
		switch def.CriteriaType {
		case "streak":
			if streak != nil && streak.CurrentStreak >= def.CriteriaValue {
				earned = true
			}
		case "sessions":
			if allTime != nil && int32(allTime.SessionsCompleted) >= def.CriteriaValue {
				earned = true
			}
		case "focus_time":
			if allTime != nil && int32(allTime.TotalFocusSeconds) >= def.CriteriaValue {
				earned = true
			}
		}

		if earned {
			if err := s.homeRepo.AwardBadge(ctx, userID, def.ID); err == nil {
				s.notifService.NotifyBadgeEarned(ctx, userID, def.Name)
			}
		}
	}
}

// ─────────────────────────────────────────
// Remaining methods
// ─────────────────────────────────────────

func (s *homeService) GetActiveSession(ctx context.Context, userID int32) (*domain.FocusSession, error) {
	return s.homeRepo.GetActiveSession(ctx, userID)
}

func (s *homeService) GetSessionHistory(ctx context.Context, userID int32, limit, offset int32) ([]domain.FocusSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.homeRepo.GetUserSessionHistory(ctx, userID, limit, offset)
}

func (s *homeService) GetStreak(ctx context.Context, userID int32) (*domain.Streak, error) {
	return s.homeRepo.GetStreak(ctx, userID)
}

func (s *homeService) GetUserBadges(ctx context.Context, userID int32) ([]domain.UserBadge, error) {
	return s.homeRepo.GetUserBadges(ctx, userID)
}

func (s *homeService) GetAllBadges(ctx context.Context) ([]domain.BadgeDefinition, error) {
	return s.homeRepo.GetAllBadgeDefinitions(ctx)
}

func (s *homeService) GetProgressSummary(ctx context.Context, userID int32, period string) (*domain.ProgressSummary, error) {
	now := time.Now()
	var from time.Time

	switch period {
	case "monthly":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "yearly":
		from = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		period = "weekly"
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = now.AddDate(0, 0, -(weekday - 1))
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	}

	summary, err := s.homeRepo.GetProgressSummary(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}

	dailyProgress, err := s.homeRepo.GetDailyProgressRange(ctx, userID, from, now)
	if err != nil {
		return nil, err
	}
	summary.Period = period
	summary.ChartPoints = buildProgressChartPoints(period, from, dailyProgress)
	return summary, nil
}

// ─────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func buildProgressChartPoints(period string, from time.Time, dailyProgress []domain.DailyProgress) []domain.ProgressChartPoint {
	switch period {
	case "monthly":
		return buildMonthlyChartPoints(from, dailyProgress)
	case "yearly":
		return buildYearlyChartPoints(from, dailyProgress)
	default:
		return buildWeeklyChartPoints(from, dailyProgress)
	}
}

func buildWeeklyChartPoints(from time.Time, dailyProgress []domain.DailyProgress) []domain.ProgressChartPoint {
	pointsByDate := make(map[string]domain.DailyProgress, len(dailyProgress))
	for _, point := range dailyProgress {
		key := normalizeDate(point.Date).Format("2006-01-02")
		pointsByDate[key] = point
	}

	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	chartPoints := make([]domain.ProgressChartPoint, 0, 7)

	for index := 0; index < 7; index++ {
		day := normalizeDate(from.AddDate(0, 0, index))
		bucketKey := day.Format("2006-01-02")
		chartPoints = append(chartPoints, newChartPoint(labels[index], bucketKey, pointsByDate[bucketKey]))
	}

	return chartPoints
}

func buildMonthlyChartPoints(from time.Time, dailyProgress []domain.DailyProgress) []domain.ProgressChartPoint {
	year, month, _ := from.Date()
	location := from.Location()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
	weeksInMonth := (daysInMonth + 6) / 7

	focusByWeek := make(map[int]int64, weeksInMonth)
	sessionsByWeek := make(map[int]int64, weeksInMonth)

	for _, point := range dailyProgress {
		weekIndex := ((normalizeDate(point.Date).Day() - 1) / 7) + 1
		focusByWeek[weekIndex] += int64(point.TotalFocusSeconds)
		sessionsByWeek[weekIndex] += int64(point.SessionsCompleted)
	}

	chartPoints := make([]domain.ProgressChartPoint, 0, weeksInMonth)
	for weekIndex := 1; weekIndex <= weeksInMonth; weekIndex++ {
		totalFocusSeconds := focusByWeek[weekIndex]
		sessionsCompleted := sessionsByWeek[weekIndex]

		chartPoints = append(chartPoints, domain.ProgressChartPoint{
			Label:             fmt.Sprintf("Week %d", weekIndex),
			BucketKey:         fmt.Sprintf("%04d-%02d-W%d", year, month, weekIndex),
			TotalFocusSeconds: totalFocusSeconds,
			TotalFocusHours:   float64(totalFocusSeconds) / 3600.0,
			SessionsCompleted: sessionsCompleted,
		})
	}

	return chartPoints
}

func buildYearlyChartPoints(from time.Time, dailyProgress []domain.DailyProgress) []domain.ProgressChartPoint {
	year := from.Year()
	focusByMonth := make(map[time.Month]int64, 12)
	sessionsByMonth := make(map[time.Month]int64, 12)

	for _, point := range dailyProgress {
		month := normalizeDate(point.Date).Month()
		focusByMonth[month] += int64(point.TotalFocusSeconds)
		sessionsByMonth[month] += int64(point.SessionsCompleted)
	}

	chartPoints := make([]domain.ProgressChartPoint, 0, 12)
	for month := time.January; month <= time.December; month++ {
		totalFocusSeconds := focusByMonth[month]
		sessionsCompleted := sessionsByMonth[month]

		chartPoints = append(chartPoints, domain.ProgressChartPoint{
			Label:             month.String()[:3],
			BucketKey:         fmt.Sprintf("%04d-%02d", year, int(month)),
			TotalFocusSeconds: totalFocusSeconds,
			TotalFocusHours:   float64(totalFocusSeconds) / 3600.0,
			SessionsCompleted: sessionsCompleted,
		})
	}

	return chartPoints
}

func newChartPoint(label, bucketKey string, progress domain.DailyProgress) domain.ProgressChartPoint {
	totalFocusSeconds := int64(progress.TotalFocusSeconds)
	sessionsCompleted := int64(progress.SessionsCompleted)

	return domain.ProgressChartPoint{
		Label:             label,
		BucketKey:         bucketKey,
		TotalFocusSeconds: totalFocusSeconds,
		TotalFocusHours:   float64(totalFocusSeconds) / 3600.0,
		SessionsCompleted: sessionsCompleted,
	}
}

func normalizeDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}
