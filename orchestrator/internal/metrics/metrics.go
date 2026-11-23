package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	JobsSubmitted = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fleet_jobs_submitted_total", Help: "Jobs accepted by the orchestrator."}, []string{"type"})
	JobsFinished  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fleet_jobs_finished_total", Help: "Jobs reaching a terminal state."}, []string{"type", "status"})
	QueueDepth    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fleet_queue_depth", Help: "Jobs currently queued."})
	ActiveWorkers = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fleet_active_workers", Help: "Workers with recent heartbeats."})
	JobDuration   = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "fleet_job_duration_seconds", Help: "Job execution duration.", Buckets: prometheus.DefBuckets}, []string{"type"})
)

func init() {
	prometheus.MustRegister(JobsSubmitted, JobsFinished, QueueDepth, ActiveWorkers, JobDuration)
}
