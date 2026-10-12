package export

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	blitz "github.com/thecoooler/blitz-client"
	"github.com/thecoooler/blitz-client/internal/jsonx"
	"github.com/thecoooler/blitz-client/internal/lossy"
)

var tempFileSequence atomic.Uint64

type assetStats struct {
	storedBytes       int
	uncompressedBytes int
}

func writeCompressedAsset(path string, data []byte, compress bool) (assetStats, error) {
	storedPath, stale := path, path+".br"
	if compress {
		storedPath, stale = path+".br", path
	}
	parent := filepath.Dir(storedPath)
	if err := os.MkdirAll(parent, 0o777); err != nil {
		return assetStats{}, blitz.IOError("could not create asset directory", parent, err)
	}
	stored := data
	if compress {
		var err error
		if stored, err = brotliCompress(data); err != nil {
			return assetStats{}, err
		}
	}
	if err := atomicWrite(storedPath, stored); err != nil {
		return assetStats{}, err
	}
	os.Remove(stale)
	return assetStats{storedBytes: len(stored), uncompressedBytes: len(data)}, nil
}

func writeJSON(path string, value any, compress bool) error {
	data, err := jsonx.MarshalIndent(value)
	if err != nil {
		return blitz.NewError("JSON_SERIALIZE_FAILED", err.Error())
	}
	_, err = writeCompressedAsset(path, data, compress)
	return err
}

func brotliCompress(data []byte) ([]byte, error) {
	command := exec.Command("brotli", "-q", "9", "-c")
	command.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return nil, blitz.NewError("BROTLI_NOT_FOUND",
			fmt.Sprintf("could not start brotli: %s; use --no-compress or install Brotli CLI", blitz.IOMessage(err)))
	}
	if err := command.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, blitz.NewError("BROTLI_FAILED", err.Error())
		}
		return nil, blitz.NewError("BROTLI_FAILED", strings.TrimSpace(lossy.String(stderr.Bytes())))
	}
	return stdout.Bytes(), nil
}

// atomicWrite writes a temporary sibling and renames it over path.
func atomicWrite(path string, data []byte) error {
	sequence := tempFileSequence.Add(1) - 1
	temporary := filepath.Join(filepath.Dir(path),
		fmt.Sprintf(".%s.%d.%d.tmp", filepath.Base(path), os.Getpid(), sequence))
	if err := os.WriteFile(temporary, data, 0o666); err != nil {
		return blitz.IOError("could not write asset", temporary, err)
	}
	if _, err := os.Lstat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return blitz.IOError("could not replace asset", path, err)
		}
	}
	if err := os.Rename(temporary, path); err != nil {
		return blitz.IOError("could not replace asset", path, err)
	}
	return nil
}
