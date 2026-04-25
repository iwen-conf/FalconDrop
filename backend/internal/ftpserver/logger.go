package ftpserver

import (
	"log/slog"
	"sync/atomic"
	"time"
)

type ftpLogger struct {
	stats *Status
}

func (l *ftpLogger) Print(sessionID string, message interface{}) {
	if l.stats != nil {
		if msg, ok := message.(string); ok {
			switch msg {
			case "Connection Established":
				atomic.AddInt64(&l.stats.ConnectedClients, 1)
				l.stats.UpdatedAt = time.Now().UTC()
			case "Connection Terminated":
				next := atomic.AddInt64(&l.stats.ConnectedClients, -1)
				if next < 0 {
					atomic.StoreInt64(&l.stats.ConnectedClients, 0)
				}
				l.stats.UpdatedAt = time.Now().UTC()
			}
		}
	}
	slog.Info("ftp", "sessionId", sessionID, "message", message)
}

func (l *ftpLogger) Printf(sessionID string, format string, v ...interface{}) {
	slog.Info("ftp", "sessionId", sessionID, "format", format, "args", v)
}

func (l *ftpLogger) PrintCommand(sessionID string, command string, params string) {
	if command == "PASS" {
		slog.Info("ftp command", "sessionId", sessionID, "command", command, "params", "****")
		return
	}
	slog.Info("ftp command", "sessionId", sessionID, "command", command, "params", params)
}

func (l *ftpLogger) PrintResponse(sessionID string, code int, message string) {
	slog.Info("ftp response", "sessionId", sessionID, "code", code, "message", message)
}
