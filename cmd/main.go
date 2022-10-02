package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rwl4/go-git-server/repository"
	"github.com/rwl4/go-git-server/storage"
	"github.com/rwl4/go-git-server/transport"
)

var (
	dataDir = flag.String("data-dir", "", "dir")
	host    = flag.String("host", "127.0.0.1", "host")
	port    = flag.String("port", "12345", "port")
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func makeManager() *repository.Manager {
	os.MkdirAll(*dataDir, 0755)
	repoStore := repository.NewFilesystemRepoStore(*dataDir)
	gitRepoMgr := repository.NewGitRepoManager(*dataDir)
	return repository.NewManager(repoStore, gitRepoMgr)
}

func main() {
	flag.Parse()
	if *dataDir == "" {
		fmt.Println("-data-dir required!")
		os.Exit(1)
	}

	objStore := storage.NewFilesystemGitRepoStorage(*dataDir)
	gh := transport.NewGitHTTPService(objStore)

	mgr := makeManager()
	rh := transport.NewRepoHTTPService(mgr)

	listenAddr := strings.Join([]string{*host, *port}, ":")

	server := transport.NewHTTPTransport(gh, rh)
	if err := server.ListenAndServe(listenAddr); err != nil {
		log.Println(err)
	}
}
