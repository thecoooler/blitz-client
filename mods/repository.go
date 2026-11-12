package mods

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const maxRepositoryDownload int64 = 256 * 1024 * 1024
const maxRepositoryExpanded uint64 = 1024 * 1024 * 1024
const maxRepositoryEntries = 50000

type GitRepository struct {
	URL       string  `json:"url"`
	Reference *string `json:"reference"`
}

func NewGitRepository(raw, revision string) (GitRepository, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return GitRepository{}, err
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Port() != "" {
		return GitRepository{}, fmt.Errorf("нужен HTTPS URL без авторизации, порта и фрагмента")
	}
	revision = strings.TrimSpace(revision)
	if strings.HasSuffix(u.Path, ".zip") {
		if revision != "" {
			return GitRepository{}, fmt.Errorf("для ZIP нельзя задавать ревизию")
		}
	} else {
		if (u.Host != "github.com" && u.Host != "gitlab.com") || u.RawQuery != "" {
			return GitRepository{}, fmt.Errorf("нужен GitHub/GitLab репозиторий или HTTPS .zip")
		}
		u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), ".git")
		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(parts) < 2 || (u.Host == "github.com" && len(parts) != 2) {
			return GitRepository{}, fmt.Errorf("нужен URL корня репозитория")
		}
		for _, p := range parts {
			if p == "" || p == "-" || p == ".." || p == "." {
				return GitRepository{}, fmt.Errorf("некорректный путь репозитория")
			}
		}
	}
	repo := GitRepository{URL: u.String()}
	if revision != "" {
		repo.Reference = &revision
	}
	return repo, nil
}
func (r GitRepository) validated() (GitRepository, error) {
	revision := ""
	if r.Reference != nil {
		revision = *r.Reference
	}
	return NewGitRepository(r.URL, revision)
}
func (r GitRepository) ArchiveURL() (string, error) {
	r, err := r.validated()
	if err != nil {
		return "", err
	}
	u, _ := url.Parse(r.URL)
	if strings.HasSuffix(u.Path, ".zip") {
		return u.String(), nil
	}
	if u.Host == "github.com" {
		result := "https://api.github.com/repos" + u.EscapedPath() + "/zipball"
		if r.Reference != nil {
			result += "/" + url.PathEscape(*r.Reference)
		}
		return result, nil
	}
	result := "https://gitlab.com/api/v4/projects/" + url.PathEscape(strings.TrimPrefix(u.Path, "/")) + "/repository/archive.zip"
	if r.Reference != nil {
		result += "?sha=" + url.QueryEscape(*r.Reference)
	}
	return result, nil
}

// RepositoryCache publishes complete immutable snapshots. Failed refresh leaves current intact.
// Callers serialize reads and refreshes for each cache.
type RepositoryCache struct{ Root string }

func (c RepositoryCache) sourceDir(r GitRepository) (string, error) {
	r, err := r.validated()
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(r)
	hash := sha256.Sum256(data)
	return filepath.Join(c.Root, hex.EncodeToString(hash[:])), nil
}
func (c RepositoryCache) CachedMods(r GitRepository) (string, error) {
	root, err := c.sourceDir(r)
	if err != nil {
		return "", err
	}
	pointer, err := os.ReadFile(filepath.Join(root, "current"))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	name := string(pointer)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:") {
		return "", fmt.Errorf("повреждён указатель кэша")
	}
	mods := filepath.Join(root, name, "mods")
	info, err := os.Stat(mods)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("нет каталога mods в кэше")
	}
	return mods, nil
}
func (c RepositoryCache) Refresh(ctx context.Context, r GitRepository) (string, error) {
	archive, err := r.ArchiveURL()
	if err != nil {
		return "", err
	}
	root, err := c.sourceDir(r)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	download, err := os.CreateTemp(root, "download-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(download.Name())
	defer download.Close()
	request, err := http.NewRequestWithContext(ctx, "GET", archive, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "ProBlitz/Go")
	client := &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != "https" {
			return fmt.Errorf("недопустимое перенаправление")
		}
		return nil
	}}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", fmt.Errorf("загрузка: HTTP %d", response.StatusCode)
	}
	n, err := io.Copy(download, io.LimitReader(response.Body, maxRepositoryDownload+1))
	if err != nil {
		return "", err
	}
	if n > maxRepositoryDownload {
		return "", fmt.Errorf("ZIP больше 256 MiB")
	}
	stage, err := os.MkdirTemp(root, "generation-")
	if err != nil {
		return "", err
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(stage)
		}
	}()
	if err = extractRepository(download.Name(), stage); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	pointer, err := os.CreateTemp(root, "pointer-")
	if err != nil {
		return "", err
	}
	defer os.Remove(pointer.Name())
	if _, err = pointer.WriteString(filepath.Base(stage)); err == nil {
		err = pointer.Sync()
	}
	closeErr := pointer.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err = os.Rename(pointer.Name(), filepath.Join(root, "current")); err != nil {
		return "", err
	}
	committed = true
	return filepath.Join(stage, "mods"), nil
}
func extractRepository(archive, destination string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	if len(z.File) > maxRepositoryEntries {
		return fmt.Errorf("слишком много файлов в ZIP")
	}
	// Accept mods/ at root or inside exactly one provider archive root.
	prefix := ""
	found := false
	for _, f := range z.File {
		parts := strings.Split(strings.TrimSuffix(f.Name, "/"), "/")
		var candidate string
		if len(parts) > 0 && parts[0] == "mods" {
			candidate = ""
		} else if len(parts) > 1 && parts[1] == "mods" {
			candidate = parts[0] + "/"
		} else {
			continue
		}
		if found && prefix != candidate {
			return fmt.Errorf("неоднозначный корень ZIP")
		}
		prefix = candidate
		found = true
	}
	if !found {
		return fmt.Errorf("репозиторий должен содержать каталог mods/")
	}
	seen := map[string]bool{}
	var expanded uint64
	for _, f := range z.File {
		name := f.Name
		if strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
			return fmt.Errorf("недопустимый путь ZIP: %q", name)
		}
		for _, part := range strings.Split(name, "/") {
			if part == ".." || part == "." {
				return fmt.Errorf("выход за каталог ZIP")
			}
		}
		key := strings.ToLower(strings.TrimSuffix(name, "/"))
		if seen[key] {
			return fmt.Errorf("дублирующийся путь ZIP")
		}
		seen[key] = true
		if f.Mode()&os.ModeSymlink != 0 || (!f.FileInfo().IsDir() && !f.Mode().IsRegular()) {
			return fmt.Errorf("ссылки и специальные файлы в ZIP запрещены")
		}
		if f.UncompressedSize64 > maxRepositoryExpanded-expanded {
			return fmt.Errorf("распакованный ZIP больше 1 GiB")
		}
		expanded += f.UncompressedSize64
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		relative := strings.TrimPrefix(name, prefix)
		if relative == "" {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(path.Clean(relative)))
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		input, e := f.Open()
		if e != nil {
			return e
		}
		output, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			input.Close()
			return e
		}
		n, e := io.Copy(output, io.LimitReader(input, int64(f.UncompressedSize64)+1))
		input.Close()
		closeErr := output.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		if uint64(n) != f.UncompressedSize64 {
			return fmt.Errorf("некорректный размер ZIP")
		}
	}
	entries, err := os.ReadDir(filepath.Join(destination, "mods"))
	if err != nil {
		return err
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		count++
		manifest := filepath.Join(destination, "mods", e.Name(), "mod.yaml")
		info, err := os.Stat(manifest)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s: нужен mod.yaml", e.Name())
		}
	}
	if count == 0 {
		return fmt.Errorf("пустой каталог mods")
	}
	return nil
}

// DiscoverRepositoryMods scans only immediate repository mod folders.
func DiscoverRepositoryMods(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, e := range entries {
		if e.IsDir() {
			result = append(result, filepath.Join(directory, e.Name(), "mod.yaml"))
		}
	}
	return result, nil
}
