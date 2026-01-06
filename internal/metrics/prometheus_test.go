package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricVariables(t *testing.T) {
	t.Run("NamespacesTotal", func(t *testing.T) {
		assert.NotNil(t, NamespacesTotal)
	})

	t.Run("TablesTotal", func(t *testing.T) {
		assert.NotNil(t, TablesTotal)
	})

	t.Run("ViewsTotal", func(t *testing.T) {
		assert.NotNil(t, ViewsTotal)
	})

	t.Run("CommitsTotal", func(t *testing.T) {
		assert.NotNil(t, CommitsTotal)
	})

	t.Run("TransactionsTotal", func(t *testing.T) {
		assert.NotNil(t, TransactionsTotal)
	})

	t.Run("ActiveLocks", func(t *testing.T) {
		assert.NotNil(t, ActiveLocks)
	})

	t.Run("ScanPlansTotal", func(t *testing.T) {
		assert.NotNil(t, ScanPlansTotal)
	})

	t.Run("DBConnectionsActive", func(t *testing.T) {
		assert.NotNil(t, DBConnectionsActive)
	})

	t.Run("DBConnectionsIdle", func(t *testing.T) {
		assert.NotNil(t, DBConnectionsIdle)
	})

	t.Run("DBConnectionsTotal", func(t *testing.T) {
		assert.NotNil(t, DBConnectionsTotal)
	})

	t.Run("LeaderStatus", func(t *testing.T) {
		assert.NotNil(t, LeaderStatus)
	})

	t.Run("BackgroundTaskRuns", func(t *testing.T) {
		assert.NotNil(t, BackgroundTaskRuns)
	})
}

func TestRecordNamespaceCreated(t *testing.T) {
	// Get initial value
	initial := testutil.ToFloat64(NamespacesTotal)

	RecordNamespaceCreated()

	after := testutil.ToFloat64(NamespacesTotal)
	assert.Equal(t, initial+1, after)
}

func TestRecordNamespaceDropped(t *testing.T) {
	// Ensure there's at least one namespace
	RecordNamespaceCreated()
	initial := testutil.ToFloat64(NamespacesTotal)

	RecordNamespaceDropped()

	after := testutil.ToFloat64(NamespacesTotal)
	assert.Equal(t, initial-1, after)
}

func TestRecordTableCreated(t *testing.T) {
	namespace := "test-ns-table-created"

	RecordTableCreated(namespace)

	value := testutil.ToFloat64(TablesTotal.WithLabelValues(namespace))
	assert.Equal(t, float64(1), value)
}

func TestRecordTableDropped(t *testing.T) {
	namespace := "test-ns-table-dropped"

	// Create first
	RecordTableCreated(namespace)
	initial := testutil.ToFloat64(TablesTotal.WithLabelValues(namespace))

	// Then drop
	RecordTableDropped(namespace)

	after := testutil.ToFloat64(TablesTotal.WithLabelValues(namespace))
	assert.Equal(t, initial-1, after)
}

func TestRecordTableCommit(t *testing.T) {
	namespace := "test-ns-commit"
	table := "test-table-commit"

	initial := testutil.ToFloat64(CommitsTotal.WithLabelValues(namespace, table))

	RecordTableCommit(namespace, table)

	after := testutil.ToFloat64(CommitsTotal.WithLabelValues(namespace, table))
	assert.Equal(t, initial+1, after)
}

func TestRecordViewCreated(t *testing.T) {
	namespace := "test-ns-view-created"

	RecordViewCreated(namespace)

	value := testutil.ToFloat64(ViewsTotal.WithLabelValues(namespace))
	assert.Equal(t, float64(1), value)
}

func TestRecordViewDropped(t *testing.T) {
	namespace := "test-ns-view-dropped"

	// Create first
	RecordViewCreated(namespace)
	initial := testutil.ToFloat64(ViewsTotal.WithLabelValues(namespace))

	// Then drop
	RecordViewDropped(namespace)

	after := testutil.ToFloat64(ViewsTotal.WithLabelValues(namespace))
	assert.Equal(t, initial-1, after)
}

func TestRecordTransactionCommit(t *testing.T) {
	status := "success"

	initial := testutil.ToFloat64(TransactionsTotal.WithLabelValues(status))

	RecordTransactionCommit(status)

	after := testutil.ToFloat64(TransactionsTotal.WithLabelValues(status))
	assert.Equal(t, initial+1, after)
}

func TestRecordScanPlan(t *testing.T) {
	status := "submitted"

	initial := testutil.ToFloat64(ScanPlansTotal.WithLabelValues(status))

	RecordScanPlan(status)

	after := testutil.ToFloat64(ScanPlansTotal.WithLabelValues(status))
	assert.Equal(t, initial+1, after)
}

func TestUpdateDBPoolStats(t *testing.T) {
	UpdateDBPoolStats(10, 5, 15)

	assert.Equal(t, float64(10), testutil.ToFloat64(DBConnectionsActive))
	assert.Equal(t, float64(5), testutil.ToFloat64(DBConnectionsIdle))
	assert.Equal(t, float64(15), testutil.ToFloat64(DBConnectionsTotal))
}

func TestSetLeaderStatus(t *testing.T) {
	task := "test-task"

	SetLeaderStatus(task, 1.0)
	assert.Equal(t, float64(1), testutil.ToFloat64(LeaderStatus.WithLabelValues(task)))

	SetLeaderStatus(task, 0.0)
	assert.Equal(t, float64(0), testutil.ToFloat64(LeaderStatus.WithLabelValues(task)))
}

func TestIncBackgroundTaskRuns(t *testing.T) {
	task := "cleanup"
	status := "success"

	initial := testutil.ToFloat64(BackgroundTaskRuns.WithLabelValues(task, status))

	IncBackgroundTaskRuns(task, status)

	after := testutil.ToFloat64(BackgroundTaskRuns.WithLabelValues(task, status))
	assert.Equal(t, initial+1, after)
}

func TestMetricDescriptions(t *testing.T) {
	t.Run("NamespacesTotal has description", func(t *testing.T) {
		ch := make(chan *prometheus.Desc, 1)
		NamespacesTotal.Describe(ch)
		desc := <-ch
		assert.Contains(t, desc.String(), "iceberg_namespaces_total")
	})

	t.Run("TablesTotal has description", func(t *testing.T) {
		ch := make(chan *prometheus.Desc, 1)
		TablesTotal.Describe(ch)
		desc := <-ch
		assert.Contains(t, desc.String(), "iceberg_tables_total")
	})

	t.Run("CommitsTotal has description", func(t *testing.T) {
		ch := make(chan *prometheus.Desc, 1)
		CommitsTotal.Describe(ch)
		desc := <-ch
		assert.Contains(t, desc.String(), "iceberg_commits_total")
	})
}

func TestMetricLabels(t *testing.T) {
	t.Run("TablesTotal uses namespace label", func(t *testing.T) {
		ns1 := "namespace-1"
		ns2 := "namespace-2"

		RecordTableCreated(ns1)
		RecordTableCreated(ns2)
		RecordTableCreated(ns2)

		v1 := testutil.ToFloat64(TablesTotal.WithLabelValues(ns1))
		v2 := testutil.ToFloat64(TablesTotal.WithLabelValues(ns2))

		// ns2 should have more tables than ns1
		assert.Less(t, v1, v2)
	})

	t.Run("CommitsTotal uses namespace and table labels", func(t *testing.T) {
		ns := "commit-ns"
		t1 := "table1"
		t2 := "table2"

		RecordTableCommit(ns, t1)
		RecordTableCommit(ns, t2)
		RecordTableCommit(ns, t2)
		RecordTableCommit(ns, t2)

		v1 := testutil.ToFloat64(CommitsTotal.WithLabelValues(ns, t1))
		v2 := testutil.ToFloat64(CommitsTotal.WithLabelValues(ns, t2))

		assert.Less(t, v1, v2)
	})

	t.Run("TransactionsTotal uses status label", func(t *testing.T) {
		RecordTransactionCommit("success")
		RecordTransactionCommit("success")
		RecordTransactionCommit("failure")

		success := testutil.ToFloat64(TransactionsTotal.WithLabelValues("success"))
		failure := testutil.ToFloat64(TransactionsTotal.WithLabelValues("failure"))

		assert.Greater(t, success, failure)
	})
}
