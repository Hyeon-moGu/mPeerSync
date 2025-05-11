package file

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"mPeerSync/util"

	"github.com/fsnotify/fsnotify"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ProcessedFileInfo struct {
	Size    int64
	ModTime time.Time
}

var processedFiles = make(map[string]ProcessedFileInfo)
var mu sync.Mutex

func WatchFolder(h host.Host, outboxPath string, listPeers func() []peer.ID, onFile func(host.Host, peer.ID, string) error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(outboxPath); err != nil {
		return err
	}

	util.Info("Watching %s for changes...", outboxPath)

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Create) != 0 {
				handleNewFile(ev.Name, outboxPath, h, listPeers, onFile)
			}
		case err := <-watcher.Errors:
			util.Fail("Watcher error: %v", err)
		}
	}
}

func handleNewFile(path, base string, h host.Host, listPeers func() []peer.ID, onFile func(host.Host, peer.ID, string) error) {
	rel, _ := filepath.Rel(base, path)

	info, err := os.Stat(path)
	if err != nil {
		util.Fail("Failed to stat file %s: %v", path, err)
		return
	}

	mu.Lock()
	prev, exists := processedFiles[rel]
	if exists && prev.Size == info.Size() && prev.ModTime.Equal(info.ModTime()) {
		mu.Unlock()
		util.Warn("Skipped duplicate unchanged file %s", rel)
		return
	}
	processedFiles[rel] = ProcessedFileInfo{Size: info.Size(), ModTime: info.ModTime()}
	mu.Unlock()

	for _, pid := range listPeers() {
		for retries := 0; retries < 3; retries++ {
			file, err := os.Open(path)
			if err == nil {
				file.Close()
				break
			}
			util.Warn("Waiting to open %s (retry %d): %v", path, retries+1, err)
			time.Sleep(300 * time.Millisecond)
		}
		if err := onFile(h, pid, path); err != nil {
			util.Fail("Error sending to %s: %v", pid, err)
		}
	}
	util.Info("Event: create on %s", rel)
}
