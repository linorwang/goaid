package workerpool

type Stats struct {
	Name             string
	Workers          int
	MinWorkers       int
	MaxWorkers       int
	Running          int64
	Queued           int
	Completed        int64
	Failed           int64
	Rejected         int64
	PanicCount       int64
	TaskPanicCount   int64
	WorkerPanicCount int64
}
