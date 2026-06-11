package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ppanda/chitragupta/pkg/types"
)

// Downloader handles parallel package downloads
type Downloader struct {
	cacheDir   string
	maxWorkers int
}

// NewDownloader creates downloader
func NewDownloader(cacheDir string, maxWorkers int) *Downloader {
	if maxWorkers == 0 {
		maxWorkers = 10
	}

	return &Downloader{
		cacheDir:   cacheDir,
		maxWorkers: maxWorkers,
	}
}

// Download fetches packages in parallel
func (d *Downloader) Download(packages []*types.Package, fetcher func(*types.Package) error) error {
	if fetcher == nil {
		return fmt.Errorf("fetcher cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan *types.Package, len(packages))
	// Buffer must accommodate all packages + all workers potentially sending errors on cancel
	results := make(chan error, len(packages)+d.maxWorkers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < d.maxWorkers; i++ {
		wg.Add(1)
		go d.worker(ctx, cancel, &wg, jobs, results, fetcher)
	}

	// Send jobs
	for _, pkg := range packages {
		jobs <- pkg
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)

	// Check results
	for err := range results {
		if err != nil {
			return err
		}
	}

	return nil
}

// worker processes download jobs
func (d *Downloader) worker(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, jobs <-chan *types.Package, results chan<- error, fetcher func(*types.Package) error) {
	defer wg.Done()

	for pkg := range jobs {
		select {
		case <-ctx.Done():
			results <- ctx.Err()
			return
		default:
			// Check cache first
			if d.isCached(pkg) {
				pkg.SetPath(d.cachePath(pkg))
				results <- nil
				continue
			}

			// Download
			if err := fetcher(pkg); err != nil {
				cancel() // Cancel context on first error
				results <- fmt.Errorf("failed to fetch %s: %w", pkg.Name, err)
				return
			}

			// Cache
			if err := d.cache(pkg); err != nil {
				cancel() // Cancel context on first error
				results <- fmt.Errorf("failed to cache %s: %w", pkg.Name, err)
				return
			}

			results <- nil
		}
	}
}

// isCached checks if package is in cache
func (d *Downloader) isCached(pkg *types.Package) bool {
	path := d.cachePath(pkg)
	_, err := os.Stat(path)
	return err == nil
}

// cache stores package in cache
func (d *Downloader) cache(pkg *types.Package) error {
	cachePath := d.cachePath(pkg)

	// Create cache directory
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}

	// Copy package to cache
	return copyDir(pkg.GetPath(), cachePath)
}

// cachePath returns cache path for package
func (d *Downloader) cachePath(pkg *types.Package) string {
	return filepath.Join(d.cacheDir, pkg.Name, pkg.Version)
}

// copyDir recursively copies directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, content, info.Mode())
	})
}
