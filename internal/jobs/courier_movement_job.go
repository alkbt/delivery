package jobs

import (
	"context"
	"log/slog"

	"delivery/internal/core/application/usecases/commands"

	"github.com/robfig/cron/v3"
)

// CourierMovementJob manages the scheduled movement of couriers.
// Runs every second to update courier positions and complete deliveries.
type CourierMovementJob struct {
	handler commands.MoveCouriersCommandHandler
	cron    *cron.Cron
	logger  *slog.Logger
}

// NewCourierMovementJob creates a new job for moving couriers.
// Uses MoveCouriersCommandHandler to process courier movements every second.
func NewCourierMovementJob(handler commands.MoveCouriersCommandHandler, logger *slog.Logger) *CourierMovementJob {
	return &CourierMovementJob{
		handler: handler,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With("component", "courier_movement_job"),
	}
}

// Start begins the courier movement job to run every second.
func (j *CourierMovementJob) Start() error {
	_, err := j.cron.AddFunc("* * * * * *", func() {
		ctx := context.Background()
		cmd := commands.NewMoveCouriersCommand()

		if err := j.handler.Handle(ctx, cmd); err != nil {
			j.logger.ErrorContext(ctx, "Courier movement job failed", "error", err)
		}
	})

	if err != nil {
		return err
	}

	j.cron.Start()
	j.logger.InfoContext(context.Background(), "Courier movement job started (running every second)")
	return nil
}

// Stop stops the courier movement job.
func (j *CourierMovementJob) Stop() {
	j.cron.Stop()
	j.logger.InfoContext(context.Background(), "Courier movement job stopped")
}
