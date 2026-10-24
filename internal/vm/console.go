package vm

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"sync"
)

const (
	sentinelPrefix        = "__EXIT_CODE:"
	sentinelSuffix        = "__"
	maxSentinelHoldLength = 48
)

// ConsoleFilter is a streaming io.WriteCloser that filters the guest exit code sentinel
// from the microVM serial console stream while passing all other output unbuffered to dest.
type ConsoleFilter struct {
	dest     io.Writer
	exitCode int
	found    bool
	buf      []byte
	closed   bool
	mu       sync.Mutex
}

// NewConsoleFilter wraps dest with sentinel scanning and unbuffered streaming.
func NewConsoleFilter(dest io.Writer) *ConsoleFilter {
	return &ConsoleFilter{
		dest:     dest,
		exitCode: -1,
		found:    false,
		buf:      make([]byte, 0, 1024),
	}
}

// Write processes incoming serial output, capturing the exit code sentinel
// and forwarding all other bytes unbuffered to the underlying writer.
func (c *ConsoleFilter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}

	c.buf = append(c.buf, p...)

	for len(c.buf) > 0 {
		idx := bytes.Index(c.buf, []byte(sentinelPrefix))
		if idx != -1 {
			// Flush everything preceding the sentinel
			if idx > 0 {
				if _, err := c.dest.Write(c.buf[:idx]); err != nil {
					return len(p), err
				}
				c.buf = c.buf[idx:]
			}

			// Now look for sentinelSuffix after sentinelPrefix
			searchFrom := len(sentinelPrefix)
			suffixIdx := bytes.Index(c.buf[searchFrom:], []byte(sentinelSuffix))
			if suffixIdx != -1 {
				endIdx := searchFrom + suffixIdx
				codeStr := string(c.buf[searchFrom:endIdx])

				afterSentinel := endIdx + len(sentinelSuffix)

				// If buffer ends exactly at the sentinel suffix, wait for trailing newline
				if afterSentinel == len(c.buf) {
					break
				}

				// Consume sentinel and any immediate trailing newline
				if afterSentinel < len(c.buf) && c.buf[afterSentinel] == '\r' {
					afterSentinel++
				}
				if afterSentinel < len(c.buf) && c.buf[afterSentinel] == '\n' {
					afterSentinel++
				}

				// Parse exit code integer
				if code, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
					c.exitCode = code
					c.found = true
				}

				c.buf = c.buf[afterSentinel:]
				continue
			}

			// Suffix not yet found. If buffer is shorter than max expected sentinel, wait for more data
			if len(c.buf) < maxSentinelHoldLength {
				break
			}

			// If it exceeds max length without closing suffix, it was not a valid sentinel; flush it
			if _, err := c.dest.Write(c.buf[:len(sentinelPrefix)]); err != nil {
				return len(p), err
			}
			c.buf = c.buf[len(sentinelPrefix):]
			continue
		}

		// Prefix not found in buffer. Check if the buffer ends with a partial prefix of sentinelPrefix
		partialLen := findPartialPrefixMatch(c.buf, sentinelPrefix)
		if partialLen > 0 {
			flushLen := len(c.buf) - partialLen
			if flushLen > 0 {
				if _, err := c.dest.Write(c.buf[:flushLen]); err != nil {
					return len(p), err
				}
				c.buf = c.buf[flushLen:]
			}
			break
		}

		// No partial prefix match at all; flush entire buffer
		if _, err := c.dest.Write(c.buf); err != nil {
			return len(p), err
		}
		c.buf = c.buf[:0]
		break
	}

	return len(p), nil
}

// Close flushes any remaining buffered bytes and marks the filter closed.
func (c *ConsoleFilter) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	// Check if buffer contains a complete or trailing sentinel
	if len(c.buf) > 0 {
		idx := bytes.Index(c.buf, []byte(sentinelPrefix))
		if idx != -1 {
			// Write pre-sentinel
			if idx > 0 {
				if _, err := c.dest.Write(c.buf[:idx]); err != nil {
					return err
				}
				c.buf = c.buf[idx:]
			}
			searchFrom := len(sentinelPrefix)
			suffixIdx := bytes.Index(c.buf[searchFrom:], []byte(sentinelSuffix))
			if suffixIdx != -1 {
				endIdx := searchFrom + suffixIdx
				codeStr := string(c.buf[searchFrom:endIdx])
				if code, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
					c.exitCode = code
					c.found = true
				}
				afterSentinel := endIdx + len(sentinelSuffix)
				if afterSentinel < len(c.buf) && c.buf[afterSentinel] == '\r' {
					afterSentinel++
				}
				if afterSentinel < len(c.buf) && c.buf[afterSentinel] == '\n' {
					afterSentinel++
				}
				c.buf = c.buf[afterSentinel:]
			}
		}

		if len(c.buf) > 0 {
			_, err := c.dest.Write(c.buf)
			c.buf = nil
			return err
		}
	}
	return nil
}

// ExitCode returns the parsed exit code and a boolean indicating whether the sentinel was found.
func (c *ConsoleFilter) ExitCode() (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.exitCode, c.found
}

// findPartialPrefixMatch returns length of longest suffix of data that matches a prefix of target.
func findPartialPrefixMatch(data []byte, target string) int {
	targetBytes := []byte(target)
	maxCheck := len(targetBytes) - 1
	if len(data) < maxCheck {
		maxCheck = len(data)
	}

	for k := maxCheck; k > 0; k-- {
		if bytes.Equal(data[len(data)-k:], targetBytes[:k]) {
			return k
		}
	}
	return 0
}
