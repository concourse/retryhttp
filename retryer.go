package retryhttp

import (
	"errors"
	"net"
	"slices"
	"strings"
	"syscall"
)

//counterfeiter:generate . Retryer

type Retryer interface {
	IsRetryable(err error) bool
}

type DefaultRetryer struct{}

func (r *DefaultRetryer) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if netErr, ok := err.(net.Error); ok {
		if netErr.Temporary() || netErr.Timeout() {
			return true
		}
	}

	// Check if the error is in our predefined list of retryable errors
	var sysErr syscall.Errno
	if errors.As(err, &sysErr) {
		if slices.Contains(retryableSyscallErrors, sysErr) {
			return true
		}
	}

	// Fall back to string matching for other error types
	errMsg := strings.ToLower(err.Error())
	for _, msg := range retryableErrorMessages {
		if strings.Contains(errMsg, msg) {
			return true
		}
	}

	return false
}

// Syscall error codes that should trigger a retry
var retryableSyscallErrors = []syscall.Errno{
	syscall.ECONNREFUSED, // Connection refused - The server is not listening on the specified port or is unreachable
	syscall.ECONNRESET,   // Connection reset by peer - The server abruptly closed the connection without proper termination
	syscall.ETIMEDOUT,    // Operation timed out - The connection or request did not complete within the allotted time
	syscall.EPIPE,        // Broken pipe - Attempt to write to a socket that has been closed by the peer
}

// Error message substrings that should trigger a retry
var retryableErrorMessages = []string{
	"i/o timeout",
	"no such host",
	"handshake failure",
	"handshake timeout",
	"timeout awaiting response headers",
	"net/http: timeout awaiting response headers", // Specific to http package timeouts
	"unexpected eof",
	"connection reset by peer",
	"read: connection reset by peer",
	"read on closed response body",
	"broken pipe",
	"use of closed network connection",
}
