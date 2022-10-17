package registry

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Registry is a registry that serves the same image for every tag
type Registry interface {
	http.Handler
}

type registry struct {
	ctx context.Context
	ref name.Reference
}

// NewRegistry returns a registry that serves the same image for every tag.
func NewRegistry(tag string, opts ...Option) (Registry, error) {
	ref, err := name.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("parsing reference: %w", err)
	}
	reg := &registry{
		ctx: context.Background(),
		ref: ref,
	}
	for _, opt := range opts {
		opt(reg)
	}
	return reg, nil
}

// ServeHTTP serves the registry
func (reg *registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serveError(w, reg.serveHTTP(w, r))
}

func (reg *registry) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return errReadOnly
	}

	if r.URL.Path == "/v2/" || r.URL.Path == "/v2" {
		w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		return nil
	}

	if !strings.HasPrefix(r.URL.Path, "/v2/") {
		return nil
	}

	url, err := reg.makeURL(r)
	if err != nil {
		return newError(fmt.Errorf("making url: %w", err))
	}

	log.Printf("%s %s -> %s\n", r.Method, r.URL.Path, url)

	req, _ := http.NewRequest(r.Method, url, nil)
	for k, v := range r.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	tr, err := reg.newTransport()
	if err != nil {
		return newError(fmt.Errorf("creating transport: %w", err))
	}

	resp, err := tr.RoundTrip(req)
	if err != nil {
		return newError(fmt.Errorf("fetching %q: %v", url, err))
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if !isBlobs(r) {
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("ERROR copying response body: %s", err)
		}
	}

	return nil
}

func (reg *registry) makeURL(r *http.Request) (string, error) {
	parts := strings.Split(r.URL.Path, "/")

	switch parts[len(parts)-2] {
	case "manifests":
		// Figure out if the digest belongs to our target. It may be
		// the digest of the manifest or, if it's a manifest list,
		// it may be the digest of one of its children.
		if strings.HasPrefix(parts[len(parts)-1], "sha256:") {
			h, err := v1.NewHash(parts[len(parts)-1])
			if err != nil {
				return "", fmt.Errorf("parsing digest: %w", err)
			}
			found, err := reg.inManifest(h)
			if err != nil {
				return "", fmt.Errorf("comparing digests: %w", err)
			}
			if found {
				return fmt.Sprintf(
					"%s://%s/v2/%s/%s/%s",
					reg.ref.Context().Scheme(),
					reg.ref.Context().RegistryStr(),
					reg.ref.Context().RepositoryStr(),
					"manifests",
					h.String(),
				), nil
			}
		}

		// Otherwise, proxy all manifest requests to the target
		// reference
		return fmt.Sprintf(
			"%s://%s/v2/%s/%s/%s",
			reg.ref.Context().Scheme(),
			reg.ref.Context().RegistryStr(),
			reg.ref.Context().RepositoryStr(),
			"manifests",
			reg.ref.Identifier(),
		), nil

	default:
		return fmt.Sprintf(
			"%s://%s/v2/%s/%s",
			reg.ref.Context().Scheme(),
			reg.ref.Context().RegistryStr(),
			reg.ref.Context().RepositoryStr(),
			strings.Join(parts[len(parts)-2:], "/"),
		), nil
	}
}

func (reg *registry) inManifest(h v1.Hash) (bool, error) {
	// If the reference is a digest, then compare that
	if strings.HasPrefix(reg.ref.Identifier(), "sha256:") {
		digest, err := v1.NewHash(reg.ref.Identifier())
		if err != nil {
			return false, fmt.Errorf("parsing digest: %w", err)
		}
		if digest == h {
			return true, nil
		}
	}

	// Otherwise we need to get the manifest to figure this out
	tr, err := reg.newTransport()
	if err != nil {
		return false, fmt.Errorf("creating transport: %w", err)
	}
	c := &http.Client{Transport: tr}

	url := fmt.Sprintf(
		"%s://%s/v2/%s/%s/%s",
		reg.ref.Context().Scheme(),
		reg.ref.Context().RegistryStr(),
		reg.ref.Context().RepositoryStr(),
		"manifests",
		reg.ref.Identifier(),
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return false, fmt.Errorf("fetching manifest: %w", err)
	}
	defer resp.Body.Close()

	if err := transport.CheckError(resp, http.StatusOK); err != nil {
		return false, fmt.Errorf("fetching manifest: %w", err)
	}

	digest, err := v1.NewHash(resp.Header.Get("Docker-Content-Digest"))
	if err != nil {
		return false, fmt.Errorf("parsing digest: %w", err)
	}
	if digest == h {
		return true, nil
	}

	mediaType := types.MediaType(resp.Header.Get("Content-Type"))
	if mediaType == types.DockerManifestList || mediaType == types.OCIImageIndex {
		indexManifest, err := v1.ParseIndexManifest(resp.Body)
		if err != nil {
			return false, fmt.Errorf("parsing index manifest: %w", err)
		}
		// TODO: these could be manifest lists as well, so we should
		// really walk the tree
		for _, manifest := range indexManifest.Manifests {
			if h == manifest.Digest {
				return true, nil
			}
		}
	}

	return false, nil
}

func (reg *registry) newTransport() (http.RoundTripper, error) {
	auth, err := authn.DefaultKeychain.Resolve(reg.ref.Context())
	if err != nil {
		return nil, fmt.Errorf("resolving auth from keychain: %w", err)
	}

	tr, err := transport.NewWithContext(reg.ctx, reg.ref.Context().Registry, auth, remote.DefaultTransport, []string{reg.ref.Scope(transport.PullScope)})
	if err != nil {
		return nil, fmt.Errorf("creating transport: %w", err)
	}

	return tr, nil
}

func isBlobs(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 2 {
		return false
	}

	return parts[len(parts)-2] == "blobs"
}
