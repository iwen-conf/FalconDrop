package ftpserver

import "log/slog"

type ftpLogger struct{}

func (l *ftpLogger) Print(sessionID string, message interface{}) {
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
