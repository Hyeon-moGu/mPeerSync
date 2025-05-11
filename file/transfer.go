package file

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"mPeerSync/util"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ChunkSize = 8 * 1024 * 1024 // 8MB

var (
	receivedChunks      = make(map[string]map[int][]byte)
	receivedChunksMutex sync.Mutex
)

type FileChunkMeta struct {
	FileName    string `json:"fileName"`
	ChunkIndex  int    `json:"chunkIndex"`
	TotalChunks int    `json:"totalChunks"`
	ChunkSize   int64  `json:"chunkSize"`
	FileHash    string `json:"fileHash"`
	Signature   string `json:"signature"`
}

func SendFileWithChunks(h host.Host, pid peer.ID, filePath string, privateKey ed25519.PrivateKey, fileHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	totalChunks := int((fi.Size() + ChunkSize - 1) / ChunkSize)

	var wg sync.WaitGroup
	for i := 0; i < totalChunks; i++ {
		wg.Add(1)
		chunkIndex := i
		offset := int64(chunkIndex) * int64(ChunkSize)

		go func() {
			defer wg.Done()

			stream, err := h.NewStream(context.Background(), pid, "/p2p-sync/1.0.0")
			if err != nil {
				util.Fail("Failed to open stream to %s: %v", pid, err)
				return
			}
			defer stream.Close()

			signature := ed25519.Sign(privateKey, []byte(fileHash))
			meta := FileChunkMeta{
				FileName:    filepath.Base(filePath),
				ChunkIndex:  chunkIndex,
				TotalChunks: totalChunks,
				ChunkSize:   ChunkSize,
				FileHash:    fileHash,
				Signature:   hex.EncodeToString(signature),
			}

			metaBytes, _ := json.Marshal(meta)
			if _, err := stream.Write(append(metaBytes, '\n')); err != nil {
				util.Fail("Failed to send metadata: %v", err)
				return
			}

			chunk := make([]byte, ChunkSize)
			n, err := file.ReadAt(chunk, offset)
			if err != nil && err != io.EOF {
				util.Fail("Failed to read chunk: %v", err)
				return
			}

			if _, err := stream.Write(chunk[:n]); err != nil {
				util.Fail("Failed to send chunk %d: %v", chunkIndex, err)
				return
			}

			util.Info("Sent chunk %d/%d to %s", chunkIndex+1, totalChunks, pid)
		}()
	}

	wg.Wait()
	return nil
}

func HandleIncomingStream(stream network.Stream, savePath string, keyring []ed25519.PublicKey, filenames []string) {
	defer stream.Close()

	decoder := json.NewDecoder(stream)
	var meta FileChunkMeta
	if err := decoder.Decode(&meta); err != nil {
		util.Fail("Failed to decode chunk meta: %v", err)
		return
	}
	util.Info("Received chunk meta: %+v", meta)

	chunkData, err := io.ReadAll(stream)
	if err != nil {
		util.Fail("Failed to read chunk data: %v", err)
		return
	}

	receivedChunksMutex.Lock()
	if receivedChunks[meta.FileName] == nil {
		receivedChunks[meta.FileName] = make(map[int][]byte)
	}
	receivedChunks[meta.FileName][meta.ChunkIndex] = chunkData
	receivedChunksMutex.Unlock()

	util.Info("Stored chunk %d/%d for %s", meta.ChunkIndex+1, meta.TotalChunks, meta.FileName)

	if len(receivedChunks[meta.FileName]) == meta.TotalChunks {
		util.Info("All chunks received for %s. Assembling file...", meta.FileName)
		assembleAndVerify(meta, savePath, keyring, filenames)
		delete(receivedChunks, meta.FileName)
	}
}

func assembleAndVerify(meta FileChunkMeta, path string, keyring []ed25519.PublicKey, filenames []string) {
	outPath := filepath.Join(path, meta.FileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		util.Fail("Failed to create file: %v", err)
		return
	}
	defer outFile.Close()

	for i := 0; i < meta.TotalChunks; i++ {
		chunk := receivedChunks[meta.FileName][i]
		if _, err := outFile.Write(chunk); err != nil {
			util.Fail("Failed writing chunk: %v", err)
			return
		}
	}

	hash, err := CalculateSHA256(outPath)
	if err != nil {
		util.Fail("Hash calculation failed: %v", err)
		return
	}

	if hash != meta.FileHash {
		util.Fail("Hash mismatch! File %s is corrupted", meta.FileName)
		return
	}

	for i, pub := range keyring {
		sig, _ := hex.DecodeString(meta.Signature)
		if ed25519.Verify(pub, []byte(meta.FileHash), sig) {
			util.Success("Signature verified by key: %s", filenames[i])
			return
		}
	}
	util.Fail("Signature verification failed for %s", meta.FileName)
}

func CalculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
