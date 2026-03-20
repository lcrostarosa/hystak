package model

import "testing"

func TestServerDef_Target_SSE(t *testing.T) {
	s := ServerDef{
		Transport: TransportSSE,
		URL:       "https://sse.example.com/events",
		Command:   "should-not-be-returned",
	}
	got := s.Target()
	if got != s.URL {
		t.Errorf("Target() = %q, want %q", got, s.URL)
	}
}

func TestServerDef_Target_HTTP(t *testing.T) {
	s := ServerDef{
		Transport: TransportHTTP,
		URL:       "https://api.example.com/mcp",
		Command:   "should-not-be-returned",
	}
	got := s.Target()
	if got != s.URL {
		t.Errorf("Target() = %q, want %q", got, s.URL)
	}
}

func TestServerDef_Target_Stdio(t *testing.T) {
	s := ServerDef{
		Transport: TransportStdio,
		Command:   "npx",
		URL:       "should-not-be-returned",
	}
	got := s.Target()
	if got != s.Command {
		t.Errorf("Target() = %q, want %q", got, s.Command)
	}
}
