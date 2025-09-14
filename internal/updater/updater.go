package updater

import (
	"fmt"
	"time"

	"ip-updater/internal/config"
	"ip-updater/internal/logger"
	"ip-updater/pkg/dns"
	"ip-updater/pkg/fileupdate"
)

type Updater struct {
	config     *config.Config
	logger     *logger.Logger
	dnsManager *dns.DNSManager
}

func New(cfg *config.Config, log *logger.Logger) *Updater {
	dnsManager := dns.NewDNSManager()
	dnsManager.InitializeProviders()

	return &Updater{
		config:     cfg,
		logger:     log,
		dnsManager: dnsManager,
	}
}

func (u *Updater) UpdateAll(newIP string) error {
	var errors []string

	// Update DNS records
	if err := u.UpdateDNS(newIP); err != nil {
		errors = append(errors, err.Error())
	}

	// Update configuration files
	if err := u.UpdateFiles(newIP); err != nil {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("some updates failed: %v", errors)
	}

	return nil
}

func (u *Updater) UpdateDNS(newIP string) error {
	var errors []string

	// Update DNS records
	for _, dnsUpdater := range u.config.DNSUpdaters {
		if err := u.updateDNSWithRetry(dnsUpdater, newIP); err != nil {
			errMsg := fmt.Sprintf("DNS update failed for %s: %v", dnsUpdater.Name, err)
			u.logger.Error(errMsg)
			errors = append(errors, errMsg)
		} else {
			u.logger.Infof("Successfully updated DNS records for %s", dnsUpdater.Name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("DNS updates failed: %v", errors)
	}

	return nil
}

func (u *Updater) UpdateFiles(newIP string) error {
	var errors []string

	// Update configuration files
	for _, fileUpdater := range u.config.FileUpdaters {
		if err := u.updateFileWithRetry(fileUpdater, newIP); err != nil {
			errMsg := fmt.Sprintf("File update failed for %s: %v", fileUpdater.Name, err)
			u.logger.Error(errMsg)
			errors = append(errors, errMsg)
		} else {
			u.logger.Infof("Successfully updated file %s", fileUpdater.Name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("File updates failed: %v", errors)
	}

	return nil
}

func (u *Updater) updateDNSWithRetry(dnsUpdater config.DNSUpdater, newIP string) error {
	maxRetries := u.config.Retry.MaxRetries
	if maxRetries == -1 {
		maxRetries = 999999 // Set a very high number for "infinite" retries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			u.logger.Warnf("Retrying DNS update for %s (attempt %d)", dnsUpdater.Name, attempt+1)
			time.Sleep(time.Duration(u.config.Retry.Interval) * time.Second)
		}

		err := u.dnsManager.UpdateDNSRecord(dnsUpdater, newIP)
		if err == nil {
			return nil
		}

		u.logger.Errorf("DNS update attempt %d failed for %s: %v", attempt+1, dnsUpdater.Name, err)

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("DNS update failed after %d attempts", maxRetries+1)
}

func (u *Updater) updateFileWithRetry(fileUpdater config.FileUpdater, newIP string) error {
	updater := fileupdate.New(
		fileUpdater.FilePath,
		fileUpdater.Format,
		fileUpdater.KeyPath,
		fileUpdater.Backup,
	)
	updater.SetLogger(u.logger)

	// Validate file first
	if err := updater.ValidateFile(); err != nil {
		return fmt.Errorf("file validation failed: %w", err)
	}

	maxRetries := u.config.Retry.MaxRetries
	if maxRetries == -1 {
		maxRetries = 999999 // Set a very high number for "infinite" retries
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			u.logger.Warnf("Retrying file update for %s (attempt %d)", fileUpdater.Name, attempt+1)
			time.Sleep(time.Duration(u.config.Retry.Interval) * time.Second)
		}

		err := updater.UpdateIP(newIP)
		if err == nil {
			return nil
		}

		u.logger.Errorf("File update attempt %d failed for %s: %v", attempt+1, fileUpdater.Name, err)

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("file update failed after %d attempts", maxRetries+1)
}

func isNonRetryableError(err error) bool {
	// Define errors that shouldn't be retried
	errorString := err.Error()

	nonRetryableErrors := []string{
		"invalid credentials",
		"unauthorized",
		"file not found",
		"permission denied",
		"invalid format",
		"unsupported",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if containsIgnoreCase(errorString, nonRetryable) {
			return true
		}
	}

	return false
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    len(s) > len(substr) &&
		    (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}