package stats

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	stats := GetInstance()

	stats.IncBootstrap()
	stats.IncUDPError(true)
	stats.IncUDPError(false)
	stats.IncRtClearing()
	stats.IncNonUTF8()
	stats.IncDBError(false)
	stats.IncDBError(true)
	stats.IncLeech([8]byte{})

	ch := make(chan prometheus.Metric)
	go func() {
		stats.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}

	expectedCount := 9 // 8 counters + 1 extension counter
	if count != expectedCount {
		t.Errorf("Expected %d metrics, but got %d", expectedCount, count)
	}
}
