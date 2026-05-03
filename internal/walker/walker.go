package walker

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/hamza-hafeez82/cortex/internal/parser"
)

const (
	// DefaultWorkers is the default number of parallel file-reading goroutines.
	// Scales to available CPU cores, capped at 16 to avoid I/O thrashing.
	DefaultWorkers = 16

	// MaxFileSize is the maximum file size we will read into memory (10 MB).
	// Files larger than this are recorded but their lines are not loaded.
	MaxFileSize = 10 * 1024 * 1024
)

// Options configures the walker behavior.
type Options struct {
	// Workers is the number of parallel goroutines reading file contents.
	// Defaults to min(NumCPU, DefaultWorkers).
	Workers int

	// LoadLines controls whether file line content is loaded into memory.
	// Set to false for a fast metadata-only pass.
	LoadLines bool

	// MaxFileSizeBytes overrides the default MaxFileSize limit.
	MaxFileSizeBytes int64
}

// DefaultOptions returns sensible defaults for a full scan.
func DefaultOptions() Options {
	workers := runtime.NumCPU()
	if workers > DefaultWorkers {
		workers = DefaultWorkers
	}
	return Options{
		Workers:          workers,
		LoadLines:        true,
		MaxFileSizeBytes: MaxFileSize,
	}
}

// Walker walks a repository root and produces a RepoMap.
type Walker struct {
	opts Options
}

// New creates a Walker with the given options.
func New(opts Options) *Walker {
	if opts.Workers <= 0 {
		opts.Workers = DefaultOptions().Workers
	}
	if opts.MaxFileSizeBytes <= 0 {
		opts.MaxFileSizeBytes = MaxFileSize
	}
	return &Walker{opts: opts}
}

// Walk traverses root, collecting all scannable files into a RepoMap.
// It uses a goroutine pool to read file contents concurrently.
func (w *Walker) Walk(root string) (*RepoMap, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root path: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("accessing root path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", absRoot)
	}

	// Phase 1: collect all candidate file paths (single-threaded directory walk)
	paths, err := w.collectPaths(absRoot)
	if err != nil {
		return nil, fmt.Errorf("collecting paths: %w", err)
	}

	// Phase 2: read file contents concurrently
	nodes := w.readFiles(absRoot, paths)

	// Phase 3: build the RepoMap
	repo := NewRepoMap(absRoot)
	for _, node := range nodes {
		repo.AddFile(node)
	}

	return repo, nil
}

// collectPaths does a depth-first walk of root and returns all file paths
// that pass the ignore rules. This is intentionally single-threaded because
// directory enumeration is inherently sequential on most filesystems.
func (w *Walker) collectPaths(root string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable paths instead of aborting the whole scan
			return nil
		}

		// Skip ignored directories entirely (do not descend)
		if d.IsDir() {
			if ShouldIgnorePath(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks to avoid infinite loops
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		name := d.Name()
		ext := filepath.Ext(name)

		if ShouldIgnoreFile(name, ext) {
			return nil
		}

		paths = append(paths, path)
		return nil
	})

	return paths, err
}

// work is the unit of work dispatched to each reader goroutine.
type work struct {
	absPath string
	relPath string
}

// readFiles fans out file reading across a goroutine pool and collects results.
func (w *Walker) readFiles(root string, paths []string) []*FileNode {
	jobs := make(chan work, len(paths))
	results := make(chan *FileNode, len(paths))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < w.opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				node := w.readFile(job.absPath, job.relPath)
				if node != nil {
					results <- node
				}
			}
		}()
	}

	// Enqueue jobs
	for _, absPath := range paths {
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			continue
		}
		jobs <- work{absPath: absPath, relPath: filepath.ToSlash(rel)}
	}
	close(jobs)

	// Close results once all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	nodes := make([]*FileNode, 0, len(paths))
	for node := range results {
		nodes = append(nodes, node)
	}

	return nodes
}

// readFile reads a single file and returns a populated FileNode.
func (w *Walker) readFile(absPath, relPath string) *FileNode {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil
	}

	name := filepath.Base(absPath)
	ext := strings.ToLower(filepath.Ext(name))
	lang := parser.DetectLanguage(name, ext)

	node := &FileNode{
		Path:      relPath,
		AbsPath:   absPath,
		Name:      name,
		Extension: ext,
		Language:  lang,
		Size:      info.Size(),
	}

	// Skip content loading for oversized files
	if info.Size() > w.opts.MaxFileSizeBytes {
		return node
	}

	if w.opts.LoadLines {
		lines, err := readLines(absPath)
		if err == nil {
			node.Lines = lines
			node.LineCount = len(lines)
		}
	}

	return node
}

// readLines opens a file and returns its content split by line.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Increase default scanner buffer for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
