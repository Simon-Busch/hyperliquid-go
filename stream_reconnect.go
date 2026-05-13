package hyperliquid

import (
	"context"
	"math/rand"
	"time"
)

// handleDisconnect tears down the current connection and schedules a
// reconnect. Safe to call repeatedly; no-op after Close.
func (s *Stream) handleDisconnect() {
	if s.closed.Load() {
		return
	}

	s.connected.Store(false)

	s.connMu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	s.connMu.Unlock()

	s.scheduleReconnect()
}

// scheduleReconnect schedules an asynchronous reconnection attempt with
// exponential backoff and jitter. Subsequent failures double the delay
// until maxReconnectWait.
func (s *Stream) scheduleReconnect() {
	if s.closed.Load() {
		return
	}

	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()

	if s.reconnectTimer != nil {
		s.reconnectTimer.Stop()
	}

	s.reconnectAttempts++
	attempts := s.reconnectAttempts

	if s.maxReconnectAttempts > 0 && attempts > s.maxReconnectAttempts {
		s.warnf("Max reconnection attempts (%d) reached, giving up", s.maxReconnectAttempts)
		return
	}

	backoff := s.reconnectWait * time.Duration(1<<(attempts-1))
	if backoff > maxReconnectWait {
		backoff = maxReconnectWait
	}
	jitter := time.Duration(float64(backoff) * 0.2 * (2*rand.Float64() - 1))
	delay := backoff + jitter
	if delay < time.Second {
		delay = time.Second
	}

	s.warnf("Reconnection attempt %d in %v...", attempts, delay)

	s.reconnectTimer = time.AfterFunc(delay, func() {
		if s.closed.Load() || s.connected.Load() {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		err := s.Connect(ctx)
		cancel()

		if err != nil {
			s.warnf("Reconnection attempt %d failed: %v", attempts, err)
			s.scheduleReconnect()
		} else {
			s.warnf("Reconnection successful after %d attempts", attempts)
		}
	})
}
