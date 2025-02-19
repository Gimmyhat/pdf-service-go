package gotenberg

import "time"

type mockStatsHandler struct {
	duration      time.Duration
	hasError      bool
	isHealthCheck bool
}

func (m *mockStatsHandler) TrackGotenbergRequest(duration time.Duration, hasError bool, isHealthCheck bool) {
	m.duration = duration
	m.hasError = hasError
	m.isHealthCheck = isHealthCheck
}
