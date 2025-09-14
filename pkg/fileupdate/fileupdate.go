package fileupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
}

func New(filePath, format, keyPath string, backup bool) *FileUpdater {
	return &FileUpdater{
		FilePath: filePath,
		Format:   format,
		KeyPath:  keyPath,
		Backup:   backup,
	}
}

func (fu *FileUpdater) UpdateIP(newIP string) error {
	// Create backup if enabled
	if fu.Backup {
		if err := fu.createBackup(); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	switch strings.ToLower(fu.Format) {
	case "json":
		return fu.updateJSON(newIP)
	case "yaml", "yml":
		return fu.updateYAML(newIP)
	case "toml":
		return fu.updateTOML(newIP)
	case "ini":
		return fu.updateINI(newIP)
	default:
		return fmt.Errorf("unsupported file format: %s", fu.Format)
	}
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

	return os.WriteFile(fu.FilePath, updatedData, 0644)
}

func (fu *FileUpdater) updateYAML(newIP string) error {
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

	return os.WriteFile(fu.FilePath, updatedData, 0644)
}

func (fu *FileUpdater) updateTOML(newIP string) error {
	var tomlData map[string]interface{}
	if _, err := toml.DecodeFile(fu.FilePath, &tomlData); err != nil {
		return err
	}

	if err := fu.setNestedValue(tomlData, fu.KeyPath, newIP); err != nil {
		return err
	}

	file, err := os.Create(fu.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(tomlData)
}

func (fu *FileUpdater) updateINI(newIP string) error {
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

	return cfg.SaveTo(fu.FilePath)
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