package ftpserver

import (
	"sync/atomic"
	"testing"
)

func TestFTPLoggerConnectionCounters(t *testing.T) {
	status := &Status{}
	l := &ftpLogger{stats: status}

	l.Print("a", "Connection Established")
	if got := atomic.LoadInt64(&status.ConnectedClients); got != 1 {
		t.Fatalf("expected connected clients = 1, got %d", got)
	}

	l.Print("a", "Connection Terminated")
	if got := atomic.LoadInt64(&status.ConnectedClients); got != 0 {
		t.Fatalf("expected connected clients = 0, got %d", got)
	}
}

func TestFTPLoggerConnectionCountersNoNegative(t *testing.T) {
	status := &Status{}
	l := &ftpLogger{stats: status}

	l.Print("a", "Connection Terminated")
	if got := atomic.LoadInt64(&status.ConnectedClients); got != 0 {
		t.Fatalf("expected connected clients floor at 0, got %d", got)
	}
}
