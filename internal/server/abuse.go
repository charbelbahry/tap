// Owner: Role A. See ROLE_A_PLAN.md step 9.

package server

import "time"

// window counts events in a fixed time period.
type window struct {
	start time.Time
	n     int
}

// hit records one event and returns the count in the current period.
func (w *window) hit(period time.Duration) int {
	if time.Since(w.start) > period {
		w.start, w.n = time.Now(), 0
	}
	w.n++
	return w.n
}

// admit counts a new connection and reports whether there is room for it.
func (s *Server) admit(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := s.recent[ip]
	if w == nil {
		w = &window{}
		s.recent[ip] = w
	}
	if w.hit(time.Minute) == rapidConnLimit+1 {
		s.log.Warn("rapid_connections", "ip", ip)
	}
	if s.conns >= maxClients {
		s.log.Warn("server_full", "ip", ip)
		return false
	}
	s.conns++
	s.log.Info("connect", "ip", ip)
	return true
}
