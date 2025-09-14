package fileupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

type FileUpdater struct {
	FilePath string
	Format   string
	KeyPath  string
	Backup   bool
	Logger   Logger
}

type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
}

func New(filePath, format, keyPath string, backup bool) *FileUpdater {
	return &FileUpdater{
		FilePath: filePath,
		Format:   format,
		KeyPath:  keyPath,
		Backup:   backup,
	}
}

func (fu *FileUpdater) SetLogger(logger Logger) {
	fu.Logger = logger
}

func (fu *FileUpdater) UpdateIP(newIP string) error {
	if fu.Logger != nil {
		fu.Logger.Infof("📁 文件更新开始 - 文件: %s, 格式: %s, 键路径: %s", fu.FilePath, fu.Format, fu.KeyPath)
	}

	// Check current value first
	currentValue, err := fu.GetCurrentValue()
	if err == nil {
		if fu.Logger != nil {
			fu.Logger.Infof("✅ 获取到当前文件键值: %s:%s = '%s'", fu.FilePath, fu.KeyPath, currentValue)
		}

		// Process the new IP value considering current value's mask
		processedIP := fu.processIPWithMask(currentValue, newIP)
		if currentValue == processedIP {
			if fu.Logger != nil {
				fu.Logger.Infof("✔️ 文件键值未变化，跳过更新: %s:%s = '%s'", fu.FilePath, fu.KeyPath, currentValue)
			}
			return nil
		}

		if fu.Logger != nil {
			fu.Logger.Infof("📝 文件键值需要更新: %s:%s 从 '%s' 更新为 '%s'", fu.FilePath, fu.KeyPath, currentValue, processedIP)
		}
		newIP = processedIP
	} else {
		if fu.Logger != nil {
			fu.Logger.Warnf("⚠️ 无法获取当前文件键值 %s:%s: %v", fu.FilePath, fu.KeyPath, err)
			fu.Logger.Infof("🔄 尝试直接更新文件键值...")
		}
	}

	// Create backup if enabled
	if fu.Backup {
		if err := fu.createBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	var updateErr error
	switch strings.ToLower(fu.Format) {
	case "json":
		updateErr = fu.updateJSON(newIP)
	case "yaml", "yml":
		updateErr = fu.updateYAML(newIP)
	case "toml":
		updateErr = fu.updateTOML(newIP)
	case "ini":
		updateErr = fu.updateINI(newIP)
	default:
		updateErr = fmt.Errorf("unsupported file format: %s", fu.Format)
	}

	if updateErr != nil {
		if fu.Logger != nil {
			fu.Logger.Warnf("❌ 文件更新失败: %s:%s: %v", fu.FilePath, fu.KeyPath, updateErr)
		}
		return updateErr
	}

	if fu.Logger != nil {
		fu.Logger.Infof("✅ 文件更新成功: %s:%s = '%s'", fu.FilePath, fu.KeyPath, newIP)
	}

	return nil
}

func (fu *FileUpdater) createBackup() error {
	backupPath := fu.FilePath + ".backup"

	src, err := os.Open(fu.FilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (fu *FileUpdater) updateJSON(newIP string) error {
	// Read and prepare data
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return err
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}

	if err := fu.setNestedValue(jsonData, fu.KeyPath, newIP); err != nil {
		return err
	}

	updatedData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write to minimize file lock time
	return fu.atomicWrite(fu.FilePath, updatedData)
}

func (fu *FileUpdater) updateYAML(newIP string) error {
	// Read and prepare data
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return err
	}

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return err
	}

	if err := fu.setNestedValue(yamlData, fu.KeyPath, newIP); err != nil {
		return err
	}

	updatedData, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}

	// Atomic write to minimize file lock time
	return fu.atomicWrite(fu.FilePath, updatedData)
}

func (fu *FileUpdater) updateTOML(newIP string) error {
	// Read and prepare data
	var tomlData map[string]interface{}
	if _, err := toml.DecodeFile(fu.FilePath, &tomlData); err != nil {
		return err
	}

	if err := fu.setNestedValue(tomlData, fu.KeyPath, newIP); err != nil {
		return err
	}

	// Prepare buffer to minimize file lock time
	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(tomlData); err != nil {
		return err
	}

	// Atomic write to minimize file lock time
	return fu.atomicWrite(fu.FilePath, []byte(buf.String()))
}

func (fu *FileUpdater) updateINI(newIP string) error {
	// Read and prepare data
	cfg, err := ini.Load(fu.FilePath)
	if err != nil {
		return err
	}

	parts := strings.Split(fu.KeyPath, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid key path for INI format: %s (expected: section/key)", fu.KeyPath)
	}

	sectionName := parts[0]
	keyName := parts[1]

	section, err := cfg.GetSection(sectionName)
	if err != nil {
		// Create section if it doesn't exist
		section, err = cfg.NewSection(sectionName)
		if err != nil {
			return err
		}
	}

	section.Key(keyName).SetValue(newIP)

	// Prepare buffer to minimize file lock time
	var buf strings.Builder
	if _, err := cfg.WriteTo(&buf); err != nil {
		return err
	}

	// Atomic write to minimize file lock time
	return fu.atomicWrite(fu.FilePath, []byte(buf.String()))
}

func (fu *FileUpdater) setNestedValue(data map[string]interface{}, keyPath string, value interface{}) error {
	keys := strings.Split(keyPath, "/")

	current := data
	for i, key := range keys[:len(keys)-1] {
		if current[key] == nil {
			current[key] = make(map[string]interface{})
		}

		next, ok := current[key].(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid path at key %s (step %d)", key, i+1)
		}
		current = next
	}

	finalKey := keys[len(keys)-1]
	current[finalKey] = value

	return nil
}

func (fu *FileUpdater) GetCurrentValue() (string, error) {
	switch strings.ToLower(fu.Format) {
	case "json":
		return fu.getCurrentValueJSON()
	case "yaml", "yml":
		return fu.getCurrentValueYAML()
	case "toml":
		return fu.getCurrentValueTOML()
	case "ini":
		return fu.getCurrentValueINI()
	default:
		return "", fmt.Errorf("unsupported file format: %s", fu.Format)
	}
}

func (fu *FileUpdater) getCurrentValueJSON() (string, error) {
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return "", err
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return "", err
	}

	value, err := fu.getNestedValue(jsonData, fu.KeyPath)
	if err != nil {
		return "", err
	}

	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("value is not a string")
}

func (fu *FileUpdater) getCurrentValueYAML() (string, error) {
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return "", err
	}

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return "", err
	}

	value, err := fu.getNestedValue(yamlData, fu.KeyPath)
	if err != nil {
		return "", err
	}

	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("value is not a string")
}

func (fu *FileUpdater) getCurrentValueTOML() (string, error) {
	var tomlData map[string]interface{}
	if _, err := toml.DecodeFile(fu.FilePath, &tomlData); err != nil {
		return "", err
	}

	value, err := fu.getNestedValue(tomlData, fu.KeyPath)
	if err != nil {
		return "", err
	}

	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("value is not a string")
}

func (fu *FileUpdater) getCurrentValueINI() (string, error) {
	cfg, err := ini.Load(fu.FilePath)
	if err != nil {
		return "", err
	}

	parts := strings.Split(fu.KeyPath, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid key path for INI format: %s (expected: section/key)", fu.KeyPath)
	}

	sectionName := parts[0]
	keyName := parts[1]

	section, err := cfg.GetSection(sectionName)
	if err != nil {
		return "", err
	}

	key := section.Key(keyName)
	return key.String(), nil
}

func (fu *FileUpdater) atomicWrite(filePath string, data []byte) error {
	// Create a temporary file in the same directory as the target file
	// This ensures it's on the same filesystem for atomic rename
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_"+filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tempPath := tempFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tempFile = nil // Prevent cleanup defer from trying to close again

	// Atomic rename - this minimizes the lock time to just the rename operation
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to atomic rename: %w", err)
	}

	return nil
}

func (fu *FileUpdater) processIPWithMask(currentValue, newIP string) string {
	// Check if current value contains a subnet mask
	cidrRegex := regexp.MustCompile(`^(.+?)(/\d+)$`)
	matches := cidrRegex.FindStringSubmatch(currentValue)

	if len(matches) == 3 {
		// Current value has a mask, preserve it
		currentIP := matches[1]
		mask := matches[2]

		// Validate current IP
		if net.ParseIP(currentIP) == nil {
			if fu.Logger != nil {
				fu.Logger.Warnf("Current IP value '%s' is not a valid IP format, but updating anyway", currentIP)
			}
		}

		// Validate new IP
		if net.ParseIP(newIP) == nil {
			if fu.Logger != nil {
				fu.Logger.Warnf("New IP value '%s' is not a valid IP format, but updating anyway", newIP)
			}
		}

		// Return new IP with preserved mask
		return newIP + mask
	} else {
		// No mask in current value, check if it's a valid IP
		if net.ParseIP(currentValue) == nil {
			if fu.Logger != nil {
				fu.Logger.Warnf("Current IP value '%s' is not a valid IP format, but updating anyway", currentValue)
			}
		}

		// Validate new IP
		if net.ParseIP(newIP) == nil {
			if fu.Logger != nil {
				fu.Logger.Warnf("New IP value '%s' is not a valid IP format, but updating anyway", newIP)
			}
		}

		return newIP
	}
}

func (fu *FileUpdater) getNestedValue(data map[string]interface{}, keyPath string) (interface{}, error) {
	keys := strings.Split(keyPath, "/")

	current := data
	for i, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid path at key %s (step %d)", key, i+1)
		}
		current = next
	}

	finalKey := keys[len(keys)-1]
	value, exists := current[finalKey]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", finalKey)
	}

	return value, nil
}

func (fu *FileUpdater) ValidateFile() error {
	// Check if file exists
	if _, err := os.Stat(fu.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", fu.FilePath)
	}

	// Validate format
	switch strings.ToLower(fu.Format) {
	case "json":
		return fu.validateJSON()
	case "yaml", "yml":
		return fu.validateYAML()
	case "toml":
		return fu.validateTOML()
	case "ini":
		return fu.validateINI()
	default:
		return fmt.Errorf("unsupported file format: %s", fu.Format)
	}
}

func (fu *FileUpdater) validateJSON() error {
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return err
	}

	var jsonData map[string]interface{}
	return json.Unmarshal(data, &jsonData)
}

func (fu *FileUpdater) validateYAML() error {
	data, err := os.ReadFile(fu.FilePath)
	if err != nil {
		return err
	}

	var yamlData map[string]interface{}
	return yaml.Unmarshal(data, &yamlData)
}

func (fu *FileUpdater) validateTOML() error {
	var tomlData map[string]interface{}
	_, err := toml.DecodeFile(fu.FilePath, &tomlData)
	return err
}

func (fu *FileUpdater) validateINI() error {
	_, err := ini.Load(fu.FilePath)
	return err
}