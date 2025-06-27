package jobs

import (
	"context"
	"errors"
	"log/slog"

	"delivery/internal/core/application/usecases/commands"

	"github.com/robfig/cron/v3"
)

// CourierAssignmentJob manages the scheduled assignment of couriers to orders.
// Runs every second to match pending orders with available couriers.
type CourierAssignmentJob struct {
	handler commands.AssignCourierCommandHandler
	cron    *cron.Cron
	logger  *slog.Logger
}

// NewCourierAssignmentJob creates a new job for assigning couriers.
// Uses AssignCourierCommandHandler to process courier assignments every second.
func NewCourierAssignmentJob(handler commands.AssignCourierCommandHandler, logger *slog.Logger) *CourierAssignmentJob {
	return &CourierAssignmentJob{
		handler: handler,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With("component", "courier_assignment_job"),
	}
}

// Start begins the courier assignment job to run every second.
func (j *CourierAssignmentJob) Start() error {
	_, err := j.cron.AddFunc("* * * * * *", func() {
		ctx := context.Background()
		cmd := commands.NewAssignCourierCommand()

		if err := j.handler.Handle(ctx, cmd); err != nil {
			// Only log errors that are not expected business scenarios
			if !errors.Is(err, commands.ErrNoOrderFound) && !errors.Is(err, commands.ErrNoFreeCouriersFound) {
				j.logger.ErrorContext(ctx, "Courier assignment job failed", "error", err)
			}
		}
	})

	if err != nil {
		return err
	}

	j.cron.Start()
	j.logger.InfoContext(context.Background(), "Courier assignment job started (running every second)")
	return nil
}

// Stop stops the courier assignment job.
func (j *CourierAssignmentJob) Stop() {
	j.cron.Stop()
	j.logger.InfoContext(context.Background(), "Courier assignment job stopped")
}
