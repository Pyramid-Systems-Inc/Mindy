package dataman

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type SearchHistory struct {
	dataDir  string
	Searches []SearchEntry `json:"searches"`
	MaxItems int           `json:"-"`
}

type SearchEntry struct {
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
	Results   int      `json:"results"`
}

type SavedSearch struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSearchHistory(dataDir string, maxItems int) *SearchHistory {
	if maxItems <= 0 {
		maxItems = 100
	}
	return &SearchHistory{
		dataDir:  dataDir,
		Searches: []SearchEntry{},
		MaxItems: maxItems,
	}
}

func (sh *SearchHistory) Load() error {
	path := filepath.Join(sh.dataDir, "search_history.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, sh)
}

func (sh *SearchHistory) Save() error {
	path := filepath.Join(sh.dataDir, "search_history.json")
	data, err := json.MarshalIndent(sh, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (sh *SearchHistory) Add(query string, results int) {
	entry := SearchEntry{
		Query:     query,
		Timestamp: time.Now(),
		Results:   results,
	}

	for i, e := range sh.Searches {
		if e.Query == query {
			sh.Searches = append(sh.Searches[:i], sh.Searches[i+1:]...)
			break
		}
	}

	sh.Searches = append([]SearchEntry{entry}, sh.Searches...)

	if len(sh.Searches) > sh.MaxItems {
		sh.Searches = sh.Searches[:sh.MaxItems]
	}

	sh.Save()
}

func (sh *SearchHistory) GetRecent(limit int) []SearchEntry {
	if limit <= 0 || limit > len(sh.Searches) {
		limit = len(sh.Searches)
	}
	return sh.Searches[:limit]
}

func (sh *SearchHistory) Clear() {
	sh.Searches = []SearchEntry{}
	sh.Save()
}

func (sh *SearchHistory) GetFilePath() string {
	return filepath.Join(sh.dataDir, "search_history.json")
}

type SavedSearches struct {
	dataDir string
	Saved   []SavedSearch `json:"saved"`
}

func NewSavedSearches(dataDir string) *SavedSearches {
	return &SavedSearches{
		dataDir: dataDir,
		Saved:   []SavedSearch{},
	}
}

func (ss *SavedSearches) Load() error {
	path := filepath.Join(ss.dataDir, "saved_searches.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, ss)
}

func (ss *SavedSearches) Save() error {
	path := filepath.Join(ss.dataDir, "saved_searches.json")
	data, err := json.MarshalIndent(ss, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (ss *SavedSearches) Add(name, query string) (*SavedSearch, error) {
	search := SavedSearch{
		ID:        generateID(),
		Name:      name,
		Query:     query,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ss.Saved = append(ss.Saved, search)
	if err := ss.Save(); err != nil {
		return nil, err
	}
	return &search, nil
}

func (ss *SavedSearches) Update(id, name, query string) (*SavedSearch, error) {
	for i, s := range ss.Saved {
		if s.ID == id {
			ss.Saved[i].Name = name
			ss.Saved[i].Query = query
			ss.Saved[i].UpdatedAt = time.Now()
			if err := ss.Save(); err != nil {
				return nil, err
			}
			return &ss.Saved[i], nil
		}
	}
	return nil, nil
}

func (ss *SavedSearches) Delete(id string) error {
	for i, s := range ss.Saved {
		if s.ID == id {
			ss.Saved = append(ss.Saved[:i], ss.Saved[i+1:]...)
			return ss.Save()
		}
	}
	return nil
}

func (ss *SavedSearches) Get(id string) *SavedSearch {
	for _, s := range ss.Saved {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

func (ss *SavedSearches) GetAll() []SavedSearch {
	return ss.Saved
}

func (ss *SavedSearches) GetFilePath() string {
	return filepath.Join(ss.dataDir, "saved_searches.json")
}

func generateID() string {
	return time.Now().Format("20060102150405") + randomSuffix()
}

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(b)
}
