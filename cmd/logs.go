package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/pomdtr/smallweb/utils"
	"github.com/spf13/cobra"
)

func NewCmdLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "logs",
		Aliases:           []string{"log"},
		Short:             "View app logs",
		Args:              cobra.MatchAll(cobra.MaximumNArgs(1)),
		ValidArgsFunction: completeApp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := filepath.Join(xdg.CacheHome, "smallweb", "http.log")
			if _, err := os.Stat(logPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("log file does not exist: %s", logPath)
				}

				return err
			}

			// Open the log file
			f, err := os.Open(logPath)
			if err != nil {
				return err
			}
			defer f.Close()

			// Get the last 10 lines of the file
			lines, err := tailFile(f, 10)
			if err != nil {
				return err
			}
			for _, line := range lines {
				msg, err := formatLine(line)
				if err != nil {
					return fmt.Errorf("failed to format log line: %w", err)
				}
				fmt.Println(msg)
			}

			// Stream new lines as they are added
			reader := bufio.NewReader(f)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						time.Sleep(1 * time.Second)
						continue
					}
					return err
				}

				msg, err := formatLine(line)
				if err != nil {
					return fmt.Errorf("failed to format log line: %w", err)
				}

				fmt.Println(msg)
			}
		},
	}

	return cmd
}

func formatLine(line string) (string, error) {
	var log utils.HttpLog
	if err := json.Unmarshal([]byte(line), &log); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s %s %s %d %d", log.Time.Format(time.RFC3339), log.Request.Method, log.Request.Url, log.Response.Status, log.Response.Bytes), nil
}

// tailFile reads the last `n` lines from the file, optimized for large files
func tailFile(f *os.File, n int) ([]string, error) {
	const bufferSize = 256
	var lines []string
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Start reading from the end of the file
	offset := stat.Size()
	buf := make([]byte, bufferSize)
	var currentLine strings.Builder
	lineCount := 0

	for offset > 0 && lineCount < n {
		// Move the offset backwards by `bufferSize` each time
		readSize := int64(bufferSize)
		if offset < readSize {
			readSize = offset
		}

		offset -= readSize
		f.Seek(offset, io.SeekStart)
		read, err := f.Read(buf[:readSize])
		if err != nil {
			return nil, err
		}

		// Process buffer from end to start
		for i := read - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				if currentLine.Len() > 0 {
					// Add line in reverse order to the start of `lines`
					lines = append([]string{reverseString(currentLine.String())}, lines...)
					lineCount++
					currentLine.Reset()
					if lineCount >= n {
						break
					}
				}
			} else {
				currentLine.WriteByte(buf[i])
			}
		}
	}

	// Append any remaining line if it exists and we still need lines
	if currentLine.Len() > 0 && lineCount < n {
		lines = append([]string{reverseString(currentLine.String())}, lines...)
	}

	// seek to the end of the file
	f.Seek(0, io.SeekEnd)

	return lines, nil
}

// reverseString reverses a string
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
