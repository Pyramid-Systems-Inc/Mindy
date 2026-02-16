package extractor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractor_ExtractText(t *testing.T) {
	ext := New()
	
	tests := []struct {
		name    string
		path    string
		content []byte
		wantLen int
	}{
		{
			name:    "plain text",
			path:    "test.txt",
			content: []byte("Hello World Test Content"),
			wantLen: 20,
		},
		{
			name:    "markdown",
			path:    "test.md",
			content: []byte("# Hello World\n\nThis is content."),
			wantLen: 15,
		},
		{
			name:    "json",
			path:    "test.json",
			content: []byte(`{"key": "value"}`),
			wantLen: 16,
		},
		{
			name:    "csv",
			path:    "test.csv",
			content: []byte("a,b,c\n1,2,3"),
			wantLen: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ext.Extract(tt.path, tt.content)
			if err != nil {
				t.Errorf("Extract() error = %v", err)
				return
			}
			if len(got) < tt.wantLen-5 {
				t.Errorf("Extract() too short: got %d, want at least %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestExtractor_ContentType(t *testing.T) {
	ext := New()
	
	tests := []struct {
		path    string
		want    string
	}{
		{"test.txt", "text/plain"},
		{"test.md", "text/markdown"},
		{"test.html", "text/html"},
		{"test.json", "application/json"},
		{"test.pdf", "application/pdf"},
		{"test.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"test.csv", "text/csv"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ext.GetContentType(tt.path)
			if got != tt.want {
				t.Errorf("GetContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractor_FileType(t *testing.T) {
	ext := New()
	
	tests := []struct {
		path string
		want string
	}{
		{"test.txt", "text"},
		{"test.md", "markdown"},
		{"test.pdf", "pdf"},
		{"test.docx", "word"},
		{"test.csv", "csv"},
		{"test.unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ext.GetFileType(tt.path)
			if got != tt.want {
				t.Errorf("GetFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractor_StripHTML(t *testing.T) {
	ext := New()
	
	html := "<html><body><h1>Title</h1><p>Hello World</p></body></html>"
	got := ext.StripHTML(html)
	
	if len(got) < 10 {
		t.Errorf("StripHTML() too short: got %d", len(got))
	}
	
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Error("StripHTML() should not contain HTML tags")
	}
}

func TestExtractMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	
	metadata, err := ExtractMetadata(testFile, []byte("test content"))
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}
	
	if metadata["filename"] != "test.txt" {
		t.Errorf("filename = %v, want test.txt", metadata["filename"])
	}
	
	if metadata["size"].(int64) == 0 {
		t.Error("size should not be 0")
	}
}

func TestExtractor_ExtractDOCX(t *testing.T) {
	ext := New()
	
	content := []byte{0x50, 0x4B, 0x03, 0x04}
	_, err := ext.Extract("test.docx", content)
	if err == nil {
		t.Log("DOCX extraction on invalid data handled gracefully")
	}
}
