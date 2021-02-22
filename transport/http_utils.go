package transport

import (
	"net/http"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

func isListRefRequest(r *http.Request) (repo string, service string, ok bool) {
	ss := r.URL.Query()["service"]
	if len(ss) < 1 || (ss[0] != transport.ReceivePackServiceName && ss[0] != transport.UploadPackServiceName) {
		return
	}
	service = ss[0]

	if strings.HasSuffix(r.URL.Path, "/info/refs") {
		repo = strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/info/refs"), "/")
		ok = true
	}

	return
}

func isPackfileRequest(r *http.Request) (repo string, service string, ok bool) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/"+transport.ReceivePackServiceName):
		repo = strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/"+transport.ReceivePackServiceName), "/")
		service = transport.ReceivePackServiceName
		ok = true

	case strings.HasSuffix(r.URL.Path, "/"+transport.UploadPackServiceName):
		repo = strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/"+transport.UploadPackServiceName), "/")
		service = transport.UploadPackServiceName
		ok = true
	}

	return
}

func isUIRequest(r *http.Request) bool {
	agent := r.Header.Get("User-Agent")
	switch {
	case strings.Contains(agent, "Chrome"),
		strings.Contains(agent, "Safari"),
		strings.Contains(agent, "FireFox"),
		strings.Contains(agent, "Mozilla"):
		return true
	}

	return false
}
