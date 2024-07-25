package startstop

type Status int

const (
	Uninitialized Status = iota
	Initializing
	Healthy
	Unhealthy
	ShuttingDown
	Stopped
)

func (cs Status) String() string {
	statusStrings := map[Status]string{
		Uninitialized: "uninitialized",
		Initializing:  "initializing",
		Healthy:       "healthy",
		Unhealthy:     "unhealthy",
		ShuttingDown:  "shutting_down",
		Stopped:       "stopped",
	}

	return statusStrings[cs]
}
