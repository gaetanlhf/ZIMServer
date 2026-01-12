package services

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
	zimfs "github.com/gaetanlhf/ZIMServer/internal/zim/fs"
	"github.com/gaetanlhf/ZIMServer/internal/zim/index"
	zimreader "github.com/gaetanlhf/ZIMServer/internal/zim/reader"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

type Archive struct {
	Name     string
	Path     string
	Reader   *zimreader.ZIMReader
	FS       *zimfs.ZIMFS
	IndexMgr *index.Manager
	Metadata Metadata
}

type Metadata struct {
	Title        string
	Description  string
	Language     string
	LanguageCode string
	Creator      string
	Publisher    string
	Date         string
	Tags         string
	Category     string
	EntryCount   uint32
}

type LanguageInfo struct {
	Code string
	Name string
}

type ArchiveService struct {
	archives map[string]*Archive
	mu       sync.RWMutex
}

func NewArchiveService() *ArchiveService {
	return &ArchiveService{
		archives: make(map[string]*Archive),
	}
}

func (s *ArchiveService) LoadZIM(path string) error {
	reader, err := zimreader.NewReader(path)
	if err != nil {
		return fmt.Errorf("failed to open ZIM: %w", err)
	}

	baseName := filepath.Base(path)
	name := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	fs := zimfs.New(reader)

	indexMgr, err := index.NewManager(reader)
	if err != nil {
		log.Printf("No search index for %s: %v", name, err)
	}

	metadata := s.extractMetadata(reader, name)

	archive := &Archive{
		Name:     name,
		Path:     path,
		Reader:   reader,
		FS:       fs,
		IndexMgr: indexMgr,
		Metadata: metadata,
	}

	s.mu.Lock()
	s.archives[name] = archive
	s.mu.Unlock()

	return nil
}

func (s *ArchiveService) extractMetadata(reader *zimreader.ZIMReader, name string) Metadata {
	header := reader.GetHeader()

	metadata := Metadata{
		EntryCount: header.EntryCount,
	}

	keys := map[string]*string{
		"Title":       &metadata.Title,
		"Description": &metadata.Description,
		"Language":    &metadata.Language,
		"Creator":     &metadata.Creator,
		"Publisher":   &metadata.Publisher,
		"Date":        &metadata.Date,
		"Tags":        &metadata.Tags,
	}

	for key, ptr := range keys {
		if value, err := reader.GetMetadata(key); err == nil {
			*ptr = value
		}
	}

	if metadata.Title == "" {
		metadata.Title = name
	}

	if metadata.Language != "" {
		if strings.Contains(metadata.Language, ",") || strings.Contains(metadata.Language, ";") {
			metadata.LanguageCode = "MUL"
		} else {
			metadata.LanguageCode = utils.GetLanguageCode(metadata.Language)
		}
	}

	metadata.Category = extractMainCategory(metadata.Tags)

	return metadata
}

func extractMainCategory(tags string) string {
	if tags == "" {
		return ""
	}

	tagList := strings.Split(tags, ";")

	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if strings.HasPrefix(tag, "_category:") && !strings.HasPrefix(tag, "_category:_") {
			category := strings.TrimPrefix(tag, "_category:")
			return utils.CapitalizeFirst(category)
		}
	}

	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if tag != "" && !strings.HasPrefix(tag, "_") {
			return utils.CapitalizeFirst(tag)
		}
	}

	return ""
}

func (s *ArchiveService) UnloadZIM(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.archives[name]; !exists {
		return fmt.Errorf("archive not found: %s", name)
	}

	delete(s.archives, name)

	// Correction: Ajout de l'extension .zim
	zimFileName := name + ".zim"
	log.Printf("%sℹ%s Unloaded ZIM: %s%s%s", colorCyan, colorReset, colorCyan, zimFileName, colorReset)
	return nil
}

func (s *ArchiveService) GetArchive(name string) (*Archive, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	archive, exists := s.archives[name]
	return archive, exists
}

func (s *ArchiveService) ListArchives() []*Archive {
	s.mu.RLock()
	defer s.mu.RUnlock()

	archives := make([]*Archive, 0, len(s.archives))
	for _, archive := range s.archives {
		archives = append(archives, archive)
	}

	sort.Slice(archives, func(i, j int) bool {
		return strings.ToLower(archives[i].Metadata.Title) < strings.ToLower(archives[j].Metadata.Title)
	})

	return archives
}

func (s *ArchiveService) GetLanguages() []LanguageInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	langMap := make(map[string]string)
	for _, archive := range s.archives {
		if archive.Metadata.Language != "" {
			code := archive.Metadata.LanguageCode
			if code != "MUL" {
				if _, exists := langMap[code]; !exists {
					langMap[code] = utils.GetLanguageName(archive.Metadata.Language)
				}
			}
		}
	}

	languages := make([]LanguageInfo, 0, len(langMap))
	for code, name := range langMap {
		languages = append(languages, LanguageInfo{
			Code: code,
			Name: name,
		})
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Name < languages[j].Name
	})

	return languages
}

func (s *ArchiveService) GetCategories() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categoryMap := make(map[string]bool)
	for _, archive := range s.archives {
		if archive.Metadata.Tags != "" {
			tags := strings.Split(archive.Metadata.Tags, ";")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" && !strings.HasPrefix(tag, "_") {
					tag = strings.TrimPrefix(tag, "_category:")
					if !strings.HasPrefix(tag, "_") {
						categoryMap[utils.CapitalizeFirst(tag)] = true
					}
				}
			}
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}

	sort.Strings(categories)

	return categories
}
