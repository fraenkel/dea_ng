package health_check

type HealthCheck interface {
	Destroy()
}

type HealthCheckCallback interface {
	Success()
	Failure()
}
