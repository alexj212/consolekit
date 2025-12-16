package consolekit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

// FileInfoWithTimestamp holds file info and timestamp for sorting
type FileInfoWithTimestamp struct {
	Name      string
	FullPath  string
	Size      int64
	Timestamp time.Time
	IsDir     bool
}

// ByTimestampDesc sorts files by timestamp in descending order
type ByTimestampDesc []FileInfoWithTimestamp

func (a ByTimestampDesc) Len() int           { return len(a) }
func (a ByTimestampDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimestampDesc) Less(i, j int) bool { return a[i].Timestamp.Before(a[j].Timestamp) }

// ListFiles lists files and directories in the specified directory
// If extension is provided, only files with that extension are included (directories always included)
func ListFiles(dir, extension string) ([]FileInfoWithTimestamp, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, err
	}

	// Get absolute path for full path display
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	f, err := os.Open(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s, err: %v", dir, err)
	}
	files, err := f.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s, err: %v", dir, err)
	}
	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s, err: %v", dir, err)
	}

	var result []FileInfoWithTimestamp

	// Loop through entries and include both directories and files
	for _, file := range files {
		fullPath := filepath.Join(absDir, file.Name())

		if file.IsDir() {
			// Always include directories
			result = append(result, FileInfoWithTimestamp{
				Name:      file.Name(),
				FullPath:  fullPath,
				Size:      0,
				Timestamp: file.ModTime(),
				IsDir:     true,
			})
		} else if extension == "" || filepath.Ext(file.Name()) == extension {
			// Include files matching extension filter (or all files if no filter)
			result = append(result, FileInfoWithTimestamp{
				Name:      file.Name(),
				FullPath:  fullPath,
				Size:      file.Size(),
				Timestamp: file.ModTime(),
				IsDir:     false,
			})
		}
	}

	// Sort: directories first, then by name
	slices.SortFunc(result, func(a, b FileInfoWithTimestamp) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})

	return result, nil
}
