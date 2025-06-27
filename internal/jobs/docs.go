// Package jobs provides scheduled background tasks for the delivery system.
//
// This package implements cron-based jobs using github.com/robfig/cron/v3
// to handle periodic operations required for the delivery service.
//
// # Available Jobs
//
// 1. CourierAssignmentJob - Runs every second to assign pending orders to available couriers
// 2. CourierMovementJob - Runs every second to move couriers toward their destinations and complete deliveries
//
// # Usage
//
// Jobs are managed through JobManager which provides a unified interface:
//
//	// Create job manager with required handlers
//	jobManager := jobs.NewJobManager(moveCouriersHandler, assignCourierHandler, logger)
//
//	// Start all jobs
//	if err := jobManager.StartAll(); err != nil {
//		log.Fatal("Failed to start jobs:", err)
//	}
//
//	// Stop all jobs when shutting down
//	defer jobManager.StopAll()
//
// # Scheduling
//
// Both jobs use the cron expression "* * * * * *" which means they run every second.
// This frequency ensures real-time responsiveness for order processing and courier movement.
//
// # Error Handling
//
// - Assignment job ignores expected business errors (no orders, no couriers)
// - Movement job logs all errors as they indicate system issues
// - Failed job starts will stop any already running jobs
package jobs
