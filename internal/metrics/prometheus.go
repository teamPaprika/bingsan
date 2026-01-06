package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// NamespacesTotal tracks the total number of namespaces.
	NamespacesTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "namespaces_total",
		Help:      "Total number of namespaces in the catalog",
	})

	// TablesTotal tracks the total number of tables per namespace.
	TablesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "tables_total",
		Help:      "Total number of tables per namespace",
	}, []string{"namespace"})

	// ViewsTotal tracks the total number of views per namespace.
	ViewsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "views_total",
		Help:      "Total number of views per namespace",
	}, []string{"namespace"})

	// CommitsTotal tracks the total number of table commits.
	CommitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iceberg",
		Name:      "commits_total",
		Help:      "Total number of table commits",
	}, []string{"namespace", "table"})

	// TransactionsTotal tracks the total number of multi-table transactions.
	TransactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iceberg",
		Name:      "transactions_total",
		Help:      "Total number of multi-table transactions",
	}, []string{"status"})

	// ActiveLocks tracks the number of currently held locks.
	ActiveLocks = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "active_locks",
		Help:      "Number of currently held catalog locks",
	})

	// ScanPlansTotal tracks the total number of scan plans submitted.
	ScanPlansTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iceberg",
		Name:      "scan_plans_total",
		Help:      "Total number of scan plans submitted",
	}, []string{"status"})

	// DBConnectionsActive tracks the number of active database connections.
	DBConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "db_connections_active",
		Help:      "Number of active database connections",
	})

	// DBConnectionsIdle tracks the number of idle database connections.
	DBConnectionsIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "db_connections_idle",
		Help:      "Number of idle database connections",
	})

	// DBConnectionsTotal tracks the total number of database connections.
	DBConnectionsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "db_connections_total",
		Help:      "Total number of database connections in pool",
	})

	// LeaderStatus tracks whether this node is the leader for a task.
	LeaderStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "iceberg",
		Name:      "leader_status",
		Help:      "Whether this node is the leader for a task (1=leader, 0=follower)",
	}, []string{"task"})

	// BackgroundTaskRuns tracks the total number of background task executions.
	BackgroundTaskRuns = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "iceberg",
		Name:      "background_task_runs_total",
		Help:      "Total number of background task executions",
	}, []string{"task", "status"})
)

// RecordNamespaceCreated increments the namespace counter.
func RecordNamespaceCreated() {
	NamespacesTotal.Inc()
}

// RecordNamespaceDropped decrements the namespace counter.
func RecordNamespaceDropped() {
	NamespacesTotal.Dec()
}

// RecordTableCreated increments the table counter for a namespace.
func RecordTableCreated(namespace string) {
	TablesTotal.WithLabelValues(namespace).Inc()
}

// RecordTableDropped decrements the table counter for a namespace.
func RecordTableDropped(namespace string) {
	TablesTotal.WithLabelValues(namespace).Dec()
}

// RecordTableCommit increments the commit counter.
func RecordTableCommit(namespace, table string) {
	CommitsTotal.WithLabelValues(namespace, table).Inc()
}

// RecordViewCreated increments the view counter for a namespace.
func RecordViewCreated(namespace string) {
	ViewsTotal.WithLabelValues(namespace).Inc()
}

// RecordViewDropped decrements the view counter for a namespace.
func RecordViewDropped(namespace string) {
	ViewsTotal.WithLabelValues(namespace).Dec()
}

// RecordTransactionCommit increments the transaction counter.
func RecordTransactionCommit(status string) {
	TransactionsTotal.WithLabelValues(status).Inc()
}

// RecordScanPlan increments the scan plan counter.
func RecordScanPlan(status string) {
	ScanPlansTotal.WithLabelValues(status).Inc()
}

// UpdateDBPoolStats updates database connection pool metrics.
func UpdateDBPoolStats(active, idle, total int) {
	DBConnectionsActive.Set(float64(active))
	DBConnectionsIdle.Set(float64(idle))
	DBConnectionsTotal.Set(float64(total))
}

// SetLeaderStatus sets the leader status for a task.
func SetLeaderStatus(task string, status float64) {
	LeaderStatus.WithLabelValues(task).Set(status)
}

// IncBackgroundTaskRuns increments the background task run counter.
func IncBackgroundTaskRuns(task, status string) {
	BackgroundTaskRuns.WithLabelValues(task, status).Inc()
}
