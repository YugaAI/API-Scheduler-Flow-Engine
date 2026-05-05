package request

type CreateScheduleRequest struct {
	CronExpression string `json:"cron_expression" binding:"required"`
}

type UpdateScheduleRequest struct {
	Enabled bool `json:"enabled"`
}
