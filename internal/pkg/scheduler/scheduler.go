package scheduler

import (
	"aiki/internal/service"
	"context"
	"log"
	"time"
)

type Scheduler struct {
	notifService service.NotificationService
}

func NewScheduler(notifService service.NotificationService) *Scheduler {
	return &Scheduler{notifService: notifService}
}

// Start begins the background scheduler. Call this in a goroutine from main.go.
func (s *Scheduler) Start() {
	log.Println("✓ Notification scheduler started")

	go s.runAt(18, 0, "daily_reminder", func() {
		ctx := context.Background()
		s.notifService.SendDailyReminders(ctx)
	})

	go s.runAt(20, 0, "streak_warning", func() {
		ctx := context.Background()
		s.notifService.SendStreakWarnings(ctx)
	})
}

// runAt runs a job every day at the specified hour and minute (24hr).
func (s *Scheduler) runAt(hour, minute int, name string, job func()) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

		// If the time has already passed today, schedule for tomorrow
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}

		waitDuration := next.Sub(now)
		log.Printf("scheduler: next %s run in %s (at %s)", name, waitDuration.Round(time.Minute), next.Format("15:04"))

		time.Sleep(waitDuration)
		log.Printf("scheduler: running %s", name)
		job()
	}
}
