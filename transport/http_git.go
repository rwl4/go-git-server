package transport

import (
	"fmt"
	"net/http"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rwl4/go-git-server/packproto"
	"github.com/rwl4/go-git-server/storage"
)

// GitHTTPService is a git http server
type GitHTTPService struct {
	// Store containing all repo storage
	stores storage.GitRepoStorage
}

// NewGitHTTPService instantiates the git http service with the provided repo store
// and object store.
func NewGitHTTPService(objstore storage.GitRepoStorage) *GitHTTPService {
	svr := &GitHTTPService{
		stores: objstore,
	}

	return svr
}

// ListReferences per the git protocol
func (svr *GitHTTPService) ListReferences(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := r.Context()
	repoID := ctx.Value(ctxKeyRepo).(string)
	service := ctx.Value(ctxKeyService).(string)

	st := svr.stores.GetStore(repoID)
	if st == nil {
		w.WriteHeader(404)
		return
	}
	riter, err := st.IterReferences()
	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	refs := make([]*plumbing.Reference, 0)
	err = riter.ForEach(func(ref *plumbing.Reference) error {
		switch ref.Type() {
		case plumbing.SymbolicReference:
			_ref, err := svr.stores.GetStore(repoID).Reference(ref.Target())
			if err == nil && _ref.Type() == plumbing.HashReference {
				refs = append(refs, plumbing.NewHashReference(ref.Name(), _ref.Hash()))
			}
		case plumbing.HashReference:
			refs = append(refs, ref)
		case plumbing.InvalidReference:
			// TODO
		default:
			// TODO
		}

		return nil
	})
	if err != nil {
		// TODO
	}

	w.Header().Add("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.WriteHeader(200)

	proto := packproto.NewProtocol(w, nil)
	proto.ListReferences(service, refs)
}

// ReceivePack implements the receive-pack protocol over http
func (svr *GitHTTPService) ReceivePack(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	repoID := r.Context().Value(ctxKeyRepo).(string)
	st := svr.stores.GetStore(repoID)
	if st == nil {
		w.WriteHeader(404)
		return
	}

	w.Header().Add("Content-Type", "application/x-git-receive-pack-result")
	w.WriteHeader(200)

	proto := packproto.NewProtocol(w, r.Body)
	proto.ReceivePack(st)
}

// UploadPack implements upload-pack protocol over http
func (svr *GitHTTPService) UploadPack(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	repoID := r.Context().Value(ctxKeyRepo).(string)
	st := svr.stores.GetStore(repoID)
	if st == nil {
		w.WriteHeader(404)
		return
	}

	proto := packproto.NewProtocol(w, r.Body)
	proto.UploadPack(st)
}
