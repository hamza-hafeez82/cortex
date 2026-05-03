package walker

// FileNode represents a single file in the scanned repository.
type FileNode struct {
	Path      string   // relative path from repo root
	AbsPath   string   // absolute path on disk
	Name      string   // filename only
	Extension string   // file extension including dot (e.g. ".go")
	Language  string   // detected language (e.g. "Go", "JavaScript")
	Size      int64    // file size in bytes
	LineCount int      // total number of lines
	Lines     []string // raw line content (populated lazily)
}

// RepoMap is the in-memory representation of a scanned repository.
// Built once during Stage 1 and shared across all subsequent stages.
type RepoMap struct {
	Root       string                 // absolute path to repo root
	Files      []*FileNode            // all scanned source files
	ByPath     map[string]*FileNode   // indexed by relative path
	ByLanguage map[string][]*FileNode // indexed by language name
	TechStack  []string               // detected technologies
	TotalLines int                    // sum of all file line counts
	TotalFiles int                    // total number of scanned files
}

// NewRepoMap creates an empty RepoMap for the given root.
func NewRepoMap(root string) *RepoMap {
	return &RepoMap{
		Root:       root,
		ByPath:     make(map[string]*FileNode),
		ByLanguage: make(map[string][]*FileNode),
	}
}

// AddFile registers a FileNode into the RepoMap indexes.
func (r *RepoMap) AddFile(f *FileNode) {
	r.Files = append(r.Files, f)
	r.ByPath[f.Path] = f
	r.ByLanguage[f.Language] = append(r.ByLanguage[f.Language], f)
	r.TotalLines += f.LineCount
	r.TotalFiles++
}

// Languages returns all detected languages in the repository.
func (r *RepoMap) Languages() []string {
	langs := make([]string, 0, len(r.ByLanguage))
	for lang := range r.ByLanguage {
		langs = append(langs, lang)
	}
	return langs
}
