package dataman

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DataManager struct {
	dataDir string
}

type ExportOptions struct {
	IncludeBlobs  bool
	IncludeGraph  bool
	IncludeTFIDF  bool
	IncludeHistory bool
	OutputPath    string
}

type ImportOptions struct {
	ImportPath    string
	Merge         bool
}

type BatchDeleteOptions struct {
	PathPattern   string
	FileType      string
	OlderThanDays int
	DryRun        bool
}

type BatchReindexOptions struct {
	PathPattern   string
	FileType      string
	OlderThanDays int
}

func NewDataManager(dataDir string) *DataManager {
	return &DataManager{dataDir: dataDir}
}

func (dm *DataManager) Export(opts *ExportOptions) error {
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(dm.dataDir, fmt.Sprintf("mindy_backup_%s.zip", time.Now().Format("20060102_150405")))
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	if opts.IncludeBlobs {
		if err := dm.addDirToZip(writer, filepath.Join(dm.dataDir, "blobs"), "blobs"); err != nil {
			return fmt.Errorf("failed to export blobs: %w", err)
		}
	}

	if opts.IncludeGraph {
		if err := dm.addDirToZip(writer, filepath.Join(dm.dataDir, "graph"), "graph"); err != nil {
			return fmt.Errorf("failed to export graph: %w", err)
		}
	}

	if opts.IncludeTFIDF {
		if err := dm.addDirToZip(writer, filepath.Join(dm.dataDir, "tfidf"), "tfidf"); err != nil {
			return fmt.Errorf("failed to export tfidf: %w", err)
		}
		if err := dm.addDirToZip(writer, filepath.Join(dm.dataDir, "vector"), "vector"); err != nil {
			return fmt.Errorf("failed to export vector: %w", err)
		}
	}

	if err := dm.addFileToZip(writer, filepath.Join(dm.dataDir, "file_tracker.json"), "file_tracker.json"); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to export file_tracker: %w", err)
		}
	}

	if opts.IncludeHistory {
		if err := dm.addFileToZip(writer, filepath.Join(dm.dataDir, "search_history.json"), "search_history.json"); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to export search_history: %w", err)
			}
		}
		if err := dm.addFileToZip(writer, filepath.Join(dm.dataDir, "saved_searches.json"), "saved_searches.json"); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to export saved_searches: %w", err)
			}
		}
	}

	if err := dm.addMetadataJSON(writer); err != nil {
		return fmt.Errorf("failed to add metadata: %w", err)
	}

	return writer.Close()
}

func (dm *DataManager) Import(opts *ImportOptions) error {
	reader, err := zip.OpenReader(opts.ImportPath)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer reader.Close()

	if !opts.Merge {
		if err := dm.clearDataDir(); err != nil {
			return fmt.Errorf("failed to clear data directory: %w", err)
		}
	}

	for _, file := range reader.File {
		if err := dm.extractFile(file); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	return nil
}

func (dm *DataManager) BatchDelete(opts *BatchDeleteOptions) (int, error) {
	trackerPath := filepath.Join(dm.dataDir, "file_tracker.json")
	data, err := os.ReadFile(trackerPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read file tracker: %w", err)
	}

	var tracker struct {
		Files map[string]interface{} `json:"files"`
	}
	if err := json.Unmarshal(data, &tracker); err != nil {
		return 0, fmt.Errorf("failed to parse file tracker: %w", err)
	}

	var toDelete []string
	cutoffTime := time.Now().AddDate(0, 0, -opts.OlderThanDays).Unix()

	for path, info := range tracker.Files {
		infoMap, ok := info.(map[string]interface{})
		if !ok {
			continue
		}

		if opts.OlderThanDays > 0 {
			if indexedAt, ok := infoMap["indexed_at"].(float64); ok {
				if int64(indexedAt) > cutoffTime {
					continue
				}
			}
		}

		if opts.PathPattern != "" {
			if !strings.Contains(path, opts.PathPattern) {
				continue
			}
		}

		if opts.FileType != "" {
			if !strings.HasSuffix(path, "."+opts.FileType) {
				continue
			}
		}

		toDelete = append(toDelete, path)
	}

	if opts.DryRun {
		return len(toDelete), nil
	}

	for _, path := range toDelete {
		delete(tracker.Files, path)
	}

	if len(toDelete) > 0 {
		newData, _ := json.Marshal(tracker)
		os.WriteFile(trackerPath, newData, 0644)
	}

	return len(toDelete), nil
}

func (dm *DataManager) BatchReindex(opts *BatchReindexOptions) ([]string, error) {
	trackerPath := filepath.Join(dm.dataDir, "file_tracker.json")
	data, err := os.ReadFile(trackerPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file tracker: %w", err)
	}

	var tracker struct {
		Files map[string]interface{} `json:"files"`
	}
	if err := json.Unmarshal(data, &tracker); err != nil {
		return nil, fmt.Errorf("failed to parse file tracker: %w", err)
	}

	var toReindex []string

	for path := range tracker.Files {
		if opts.PathPattern != "" {
			if !strings.Contains(path, opts.PathPattern) {
				continue
			}
		}

		if opts.FileType != "" {
			if !strings.HasSuffix(path, "."+opts.FileType) {
				continue
			}
		}

		toReindex = append(toReindex, path)
	}

	return toReindex, nil
}

func (dm *DataManager) Reset() error {
	return dm.clearDataDir()
}

func (dm *DataManager) clearDataDir() error {
	entries, err := os.ReadDir(dm.dataDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == "blobs" || entry.Name() == "graph" || 
		   entry.Name() == "tfidf" || entry.Name() == "vector" ||
		   entry.Name() == "file_tracker.json" || entry.Name() == "search_history.json" ||
		   entry.Name() == "saved_searches.json" {
			os.RemoveAll(filepath.Join(dm.dataDir, entry.Name()))
		}
	}
	return nil
}

func (dm *DataManager) addDirToZip(writer *zip.Writer, sourceDir, prefix string) error {
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		zipPath := filepath.Join(prefix, relPath)
		return dm.addFileToZip(writer, path, zipPath)
	})
}

func (dm *DataManager) addFileToZip(writer *zip.Writer, sourcePath, zipPath string) error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return err
	}

	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	w, err := writer.Create(zipPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)
	return err
}

func (dm *DataManager) addMetadataJSON(writer *zip.Writer) error {
	metadata := map[string]interface{}{
		"version":      "2.0.0",
		"exported_at": time.Now().Format(time.RFC3339),
		"data_dir":    dm.dataDir,
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	w, err := writer.Create("metadata.json")
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func (dm *DataManager) extractFile(file *zip.File) error {
	// Skip metadata
	if file.Name == "metadata.json" {
		return nil
	}

	outPath := filepath.Join(dm.dataDir, file.Name)

	if file.FileInfo().IsDir() {
		return os.MkdirAll(outPath, 0755)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
