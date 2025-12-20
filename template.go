package consolekit

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
)

// TemplateManager manages script templates
type TemplateManager struct {
	templates    map[string]*template.Template
	embeddedFS   embed.FS
	templatesDir string
	mu           sync.RWMutex
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(templatesDir string, embeddedFS embed.FS) *TemplateManager {
	return &TemplateManager{
		templates:    make(map[string]*template.Template),
		embeddedFS:   embeddedFS,
		templatesDir: templatesDir,
	}
}

// LoadTemplate loads a template from file or embedded FS
func (tm *TemplateManager) LoadTemplate(name string) (*template.Template, error) {
	// Check if already loaded (with read lock)
	tm.mu.RLock()
	tmpl, exists := tm.templates[name]
	tm.mu.RUnlock()
	if exists {
		return tmpl, nil
	}

	// Try to load from embedded FS first
	content, err := tm.embeddedFS.ReadFile(name)
	if err == nil {
		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedded template %s: %w", name, err)
		}
		tm.mu.Lock()
		tm.templates[name] = tmpl
		tm.mu.Unlock()
		return tmpl, nil
	}

	// Try to load from templates directory
	if tm.templatesDir != "" {
		filePath := filepath.Join(tm.templatesDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("template not found: %s", name)
		}

		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		tm.mu.Lock()
		tm.templates[name] = tmpl
		tm.mu.Unlock()
		return tmpl, nil
	}

	return nil, fmt.Errorf("template not found: %s", name)
}

// ExecuteTemplate executes a template with the given data
func (tm *TemplateManager) ExecuteTemplate(name string, data map[string]interface{}) (string, error) {
	tmpl, err := tm.LoadTemplate(name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// ListTemplates lists all available templates
func (tm *TemplateManager) ListTemplates() ([]string, error) {
	templates := make([]string, 0)

	// List embedded templates
	entries, err := tm.embeddedFS.ReadDir(".")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
				templates = append(templates, entry.Name())
			}
		}
	}

	// List file system templates
	if tm.templatesDir != "" {
		entries, err := os.ReadDir(tm.templatesDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
					// Avoid duplicates
					name := entry.Name()
					found := false
					for _, t := range templates {
						if t == name {
							found = true
							break
						}
					}
					if !found {
						templates = append(templates, name)
					}
				}
			}
		}
	}

	return templates, nil
}

// GetTemplateContent returns the raw content of a template
func (tm *TemplateManager) GetTemplateContent(name string) (string, error) {
	// Try embedded FS first
	content, err := tm.embeddedFS.ReadFile(name)
	if err == nil {
		return string(content), nil
	}

	// Try file system
	if tm.templatesDir != "" {
		filePath := filepath.Join(tm.templatesDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("template not found: %s", name)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("template not found: %s", name)
}

// SaveTemplate saves a template to the file system
func (tm *TemplateManager) SaveTemplate(name string, content string) error {
	if tm.templatesDir == "" {
		return fmt.Errorf("templates directory not configured")
	}

	// Ensure templates directory exists
	if err := os.MkdirAll(tm.templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Ensure .tmpl extension
	if !strings.HasSuffix(name, ".tmpl") {
		name = name + ".tmpl"
	}

	filePath := filepath.Join(tm.templatesDir, name)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	return nil
}

// DeleteTemplate deletes a template from the file system
func (tm *TemplateManager) DeleteTemplate(name string) error {
	if tm.templatesDir == "" {
		return fmt.Errorf("templates directory not configured")
	}

	filePath := filepath.Join(tm.templatesDir, name)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Remove from cache
	delete(tm.templates, name)

	return nil
}

// ClearCache clears the template cache
func (tm *TemplateManager) ClearCache() {
	tm.templates = make(map[string]*template.Template)
}

// ParseVariables parses variables from command line format (key=value)
func ParseVariables(args []string) (map[string]interface{}, error) {
	vars := make(map[string]interface{})

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable format: %s (expected key=value)", arg)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		vars[key] = value
	}

	return vars, nil
}
