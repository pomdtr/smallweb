package utils

import (
	"bufio"
	"io"
	"log/slog"
)

func LogPipe(pipe io.ReadCloser, logger *slog.Logger) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		logger.Info(scanner.Text())
	}
}
