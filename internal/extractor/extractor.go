package extractor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Extractor struct{}

func New() *Extractor {
	return &Extractor{}
}

func (e *Extractor) Extract(path string, content []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".pdf":
		return e.ExtractPDF(content)
	case ".docx":
		return e.ExtractDOCX(content)
	case ".doc":
		return e.ExtractDOC(content)
	case ".txt", ".md", ".markdown":
		return string(content), nil
	case ".html", ".htm":
		return e.StripHTML(string(content)), nil
	case ".json":
		return string(content), nil
	case ".xml":
		return string(content), nil
	case ".csv":
		return e.ExtractCSV(content)
	case ".log":
		return string(content), nil
	default:
		return string(content), nil
	}
}

func (e *Extractor) ExtractPDF(content []byte) (string, error) {
	text := extractPDFText(content)
	if text == "" {
		return string(content), nil
	}
	return text, nil
}

func extractPDFText(content []byte) string {
	var text strings.Builder
	reader := bytes.NewReader(content)
	
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n == 0 {
			break
		}
		if err != nil && err != io.EOF {
			break
		}
		
		chunk := string(buf[:n])
		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
			cleaned := cleanPDFLine(line)
			if len(cleaned) > 2 {
				text.WriteString(cleaned)
				text.WriteString("\n")
			}
		}
		
		if err == io.EOF {
			break
		}
	}
	
	return text.String()
}

func cleanPDFLine(line string) string {
	var result strings.Builder
	for _, c := range line {
		if c >= 32 && c < 127 {
			result.WriteRune(c)
		} else if c == '\n' || c == '\r' || c == '\t' {
			result.WriteRune(' ')
		}
	}
	return strings.TrimSpace(result.String())
}

func (e *Extractor) ExtractDOCX(content []byte) (string, error) {
	reader := bytes.NewReader(content)
	zipReader, err := zip.NewReader(reader, int64(len(content)))
	if err != nil {
		return string(content), err
	}

	var text strings.Builder
	
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				continue
			}
			defer rc.Close()
			
			data, err := io.ReadAll(rc)
			if err != nil {
				continue
			}
			
			decoder := xml.NewDecoder(bytes.NewReader(data))
			inText := false
			var currentTag string
			
			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}
				
				switch t := token.(type) {
				case xml.StartElement:
					currentTag = t.Name.Local
					if currentTag == "t" || currentTag == "p" {
						inText = true
					}
				case xml.EndElement:
					if t.Name.Local == "p" {
						text.WriteString("\n")
					}
					inText = false
				case xml.CharData:
					if inText && currentTag == "t" {
						text.WriteString(string(t))
					}
				}
			}
			break
		}
	}
	
	if text.Len() == 0 {
		return string(content), nil
	}
	
	return text.String(), nil
}

func (e *Extractor) ExtractDOC(content []byte) (string, error) {
	return string(content), fmt.Errorf("DOC format not fully supported, raw content returned")
}

func (e *Extractor) ExtractCSV(content []byte) (string, error) {
	lines := strings.Split(string(content), "\n")
	var text strings.Builder
	
	for _, line := range lines {
		fields := strings.Split(line, ",")
		for _, field := range fields {
			cleaned := strings.TrimSpace(field)
			if len(cleaned) > 0 {
				text.WriteString(cleaned)
				text.WriteString(" ")
			}
		}
		text.WriteString("\n")
	}
	
	return text.String(), nil
}

func (e *Extractor) StripHTML(html string) string {
	var buf bytes.Buffer
	inTag := false
	for _, c := range html {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			buf.WriteRune(c)
		}
	}
	
	text := buf.String()
	text = strings.ReplaceAll(text, "\n ", "\n")
	text = strings.ReplaceAll(text, "  ", " ")
	
	return text
}

func (e *Extractor) GetContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".txt":
		return "text/plain"
	case ".md", ".markdown":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".csv":
		return "text/csv"
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	case ".log":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func (e *Extractor) GetFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".txt", ".md", ".markdown":
		return "text"
	case ".html", ".htm":
		return "html"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".csv":
		return "csv"
	case ".pdf":
		return "pdf"
	case ".docx", ".doc":
		return "word"
	case ".log":
		return "log"
	default:
		return "unknown"
	}
}

func ExtractMetadata(path string, content []byte) (map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	
	metadata := map[string]interface{}{
		"filename":   filepath.Base(path),
		"extension":  filepath.Ext(path),
		"size":       info.Size(),
		"modified":   info.ModTime().Unix(),
		"created":    info.ModTime().Unix(),
		"is_dir":     info.IsDir(),
		"file_type":  GetFileTypeSimple(path),
	}
	
	if size := info.Size(); size < 1024*1024 {
		metadata["size_display"] = fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		metadata["size_display"] = fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	} else {
		metadata["size_display"] = fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
	}
	
	return metadata, nil
}

func GetFileTypeSimple(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	types := map[string]string{
		".txt":     "text",
		".md":      "markdown",
		".markdown": "markdown",
		".html":    "html",
		".htm":     "html",
		".json":    "json",
		".xml":     "xml",
		".csv":     "csv",
		".pdf":     "pdf",
		".docx":    "word",
		".doc":     "word",
		".log":     "log",
	}
	
	if t, ok := types[ext]; ok {
		return t
	}
	return "unknown"
}
