package sync_scheduler_test

import (
	"github.com/hariskhan14/sync-scheduler"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSyncScheduler_DoWithLockedForFullInterval(t *testing.T) {
	name := "test_job_full_interval"
	expiry := 2 * time.Second
	s7l := sync_scheduler.NewSyncScheduler(name, expiry, sync_scheduler.RedisOptions{
		Addr: "localhost:6379",
	})

	success := "eat"
	val := &success
	err := s7l.Every(1).Seconds().KeepLocked().Do(cycle, val)
	require.NoError(t, err)

	s7l.StartAsync()
	time.Sleep(8 * time.Second)

	require.Equal(t, "done", *val)
}

func cycle(val *string) {
	switch *val {
	case "eat":
		*val = "sleep"
	case "sleep":
		*val = "code"
	case "code":
		*val = "repeat"
	case "repeat":
		*val = "done"
	default:
		*val = "fail"
	}
}

func TestSyncScheduler_DoWithoutLockedForFullInterval(t *testing.T) {
	name := "test_job_without_full_interval"
	expiry := 30 * time.Second
	s7l := sync_scheduler.NewSyncScheduler(name, expiry, sync_scheduler.RedisOptions{
		Addr: "localhost:6379",
	})

	z := 0
	count := &z
	err := s7l.Every(1).Seconds().Do(incrementCounter, count)
	require.NoError(t, err)

	s7l.StartAsync()

	time.Sleep(10 * time.Second)

	require.True(t, *count > 5)
}

func incrementCounter(val *int) {
	*val = *val + 1
}
