package dns

import "errors"

var (
	ErrProviderNotFound   = errors.New("DNS provider not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRecordNotFound     = errors.New("DNS record not found")
	ErrUpdateFailed       = errors.New("failed to update DNS record")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidDomain      = errors.New("invalid domain")
	ErrInvalidRecordType  = errors.New("invalid record type")
)
