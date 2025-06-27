package jobs

import (
	"fmt"
	"log/slog"

	"delivery/internal/core/application/usecases/commands"
)

// JobManager coordinates all scheduled jobs in the application.
// Provides a unified interface to start and stop all background jobs.
type JobManager struct {
	courierMovementJob   *CourierMovementJob
	courierAssignmentJob *CourierAssignmentJob
}

// NewJobManager creates a new job manager with all required jobs.
// Takes command handlers as dependencies to wire up the job execution.
func NewJobManager(
	moveCouriersHandler commands.MoveCouriersCommandHandler,
	assignCourierHandler commands.AssignCourierCommandHandler,
	logger *slog.Logger,
) *JobManager {
	return &JobManager{
		courierMovementJob:   NewCourierMovementJob(moveCouriersHandler, logger),
		courierAssignmentJob: NewCourierAssignmentJob(assignCourierHandler, logger),
	}
}

// StartAll starts all scheduled jobs.
// Returns an error if any job fails to start.
func (jm *JobManager) StartAll() error {
	if err := jm.courierAssignmentJob.Start(); err != nil {
		return fmt.Errorf("failed to start courier assignment job: %w", err)
	}

	if err := jm.courierMovementJob.Start(); err != nil {
		// Stop already started jobs if this one fails
		jm.courierAssignmentJob.Stop()
		return fmt.Errorf("failed to start courier movement job: %w", err)
	}

	return nil
}

// StopAll stops all scheduled jobs gracefully.
func (jm *JobManager) StopAll() {
	jm.courierMovementJob.Stop()
	jm.courierAssignmentJob.Stop()
}
