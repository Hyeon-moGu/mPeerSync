package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"mPeerSync/crypto"
	"mPeerSync/file"
	"mPeerSync/util"

	userpeer "mPeerSync/peer"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	libpeer "github.com/libp2p/go-libp2p/core/peer"
)

var (
	listenPort = flag.Int("port", 0, "port to listen on")
	syncDir    = flag.String("dir", "./sync", "folder to sync")
	keyMake    = flag.Bool("keymake", false, "generate a new keypair")
)

func main() {
	flag.Parse()

	os.MkdirAll(filepath.Join(*syncDir, "outbox"), 0755)
	os.MkdirAll(filepath.Join(*syncDir, "inbox"), 0755)
	os.MkdirAll("keys", 0755)

	if *keyMake {
		if err := crypto.GenerateAndSaveKeyPair(); err != nil {
			log.Fatal("Key generation failed:", err)
		}
		return
	}

	if *listenPort == 0 {
		log.Fatal("Please provide --port flag")
	}

	_, privKey, err := crypto.LoadKeyPair()
	if err != nil {
		log.Fatal("Missing key pair. Please generate with --keymake first.")
	}
	keyring, filenames, err := crypto.LoadKeyring("keys")
	if err != nil {
		log.Fatal("Failed to load keyring:", err)
	}
	util.Info("Loaded %d public keys from keys/ directory: %v", len(filenames), filenames)

	h, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *listenPort)))
	if err != nil {
		log.Fatal(err)
	}
	util.Success("Host created. We are: %s", h.ID())

	peerManager := userpeer.NewPeerManager()

	notifee := &userpeer.DiscoveryNotifee{
		Host:               h,
		OnPeerConnected:    peerManager.AddPeer,
		IsAlreadyConnected: peerManager.IsConnected,
		MarkConnected:      peerManager.MarkConnected,
		UnmarkConnected:    peerManager.UnmarkConnected,
	}

	ctx := context.Background()
	if err := userpeer.SetupMDNS(ctx, h, notifee); err != nil {
		log.Fatal("mDNS setup failed:", err)
	}

	h.SetStreamHandler("/p2p-sync/1.0.0", func(stream network.Stream) {
		file.HandleIncomingStream(stream, filepath.Join(*syncDir, "inbox"), keyring, filenames)
	})

	go func() {
		err := file.WatchFolder(h,
			filepath.Join(*syncDir, "outbox"),
			peerManager.ListPeers,
			func(h host.Host, pid libpeer.ID, path string) error {
				fileHash, err := file.CalculateSHA256(path)
				if err != nil {
					return err
				}
				return file.SendFileWithChunks(h, pid, path, privKey, fileHash)
			})
		if err != nil {
			log.Fatal(err)
		}
	}()

	select {}
}
