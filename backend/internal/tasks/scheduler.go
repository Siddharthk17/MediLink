package tasks

import "github.com/hibiken/asynq"

const (
	TaskCleanupExpiredTokens = "tasks:cleanup:tokens"
	TaskESReindexMissed      = "tasks:es:reindex"
	TaskExpireStaleJobs      = "tasks:jobs:expire"
	TaskRevokeExpiredConsents = "tasks:consent:expire"
	TaskDailyStatsSnapshot   = "tasks:stats:snapshot"
)

func RegisterPeriodicTasks(scheduler *asynq.Scheduler) {
	scheduler.Register("@every 6h", asynq.NewTask(TaskCleanupExpiredTokens, nil))
	scheduler.Register("@every 2h", asynq.NewTask(TaskESReindexMissed, nil))
	scheduler.Register("@every 15m", asynq.NewTask(TaskExpireStaleJobs, nil))
	scheduler.Register("@every 1h", asynq.NewTask(TaskRevokeExpiredConsents, nil))
	scheduler.Register("30 20 * * *", asynq.NewTask(TaskDailyStatsSnapshot, nil)) // 02:00 IST
}
