package gitea

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/simongregorebner/caddy-gitea/pkg/gitea"
)

func init() {
	caddy.RegisterModule(GiteaPagesModule{})
	httpcaddyfile.RegisterHandlerDirective("gitea", parseCaddyfile)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var module GiteaPagesModule
	err := module.UnmarshalCaddyfile(h.Dispenser)

	return module, err
}

// GiteaPagesModule implements gitea plugin.
type GiteaPagesModule struct {
	Client     *gitea.Client `json:"-"`
	Server     string        `json:"server,omitempty"`
	Token      string        `json:"token,omitempty"`
	GiteaPages string        `json:"gitea_pages,omitempty"`
	Domain     string        `json:"domain,omitempty"`
	Simple     string        `json:"simple,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (GiteaPagesModule) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.gitea",
		New: func() caddy.Module { return new(GiteaPagesModule) },
	}
}

// Provision provisions gitea client.
func (module *GiteaPagesModule) Provision(ctx caddy.Context) error {

	var err error
	// retrieve logger from the caddy context
	// https://caddyserver.com/docs/extending-caddy#logs
	var logger = ctx.Logger() // get logger

	requiredBranchName := "gitea-pages"
	requiredTopicName := "gitea-pages"
	if module.GiteaPages != "" {
		requiredBranchName = module.GiteaPages
		requiredTopicName = module.GiteaPages
	}

	module.Client, err = gitea.NewClient(logger, module.Server, module.Token, requiredBranchName, requiredTopicName)

	return err
}

// Validate implements caddy.Validator.
func (module *GiteaPagesModule) Validate() error {
	return nil
}

// UnmarshalCaddyfile unmarshals a Caddyfile.
func (module *GiteaPagesModule) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for n := d.Nesting(); d.NextBlock(n); {
			switch d.Val() {
			case "server":
				d.Args(&module.Server)
			case "token":
				d.Args(&module.Token)
			case "gitea_pages":
				d.Args(&module.GiteaPages)
			case "domain":
				d.Args(&module.Domain)
			case "simple":
				d.Args(&module.Simple)
			}
		}
	}
	return nil
}

// ServeHTTP performs gitea content fetcher.
func (module GiteaPagesModule) ServeHTTP(writer http.ResponseWriter, request *http.Request, _ caddyhttp.Handler) error {

	fmt.Println("URL " + request.URL.Path)

	var organization, repository, path string
	if module.Simple != "" { // Simple case
		// we remove a potential prefix and then split up the path
		// The path looks like /<organization>/<repository>/ ...
		parts := strings.Split(strings.TrimPrefix(request.URL.Path, "/"), "/")

		length := len(parts)
		if length <= 1 {
			return caddyhttp.Error(http.StatusNotFound, fs.ErrNotExist)
		} else if length == 2 {
			organization = parts[0]
			repository = parts[1]
			path = "index.html" // there is no file/path specified
		} else {
			organization = parts[0]
			repository = parts[1]
			path = strings.Join(parts[2:], "/")
		}
		if path == "" {
			path = "index.html"
		}

	} else {
		// TODO not yet supported

		// fmt.Println("NON SIMPLE SETUP")
		// // remove the domain if it's set (works fine if it's empty)
		// host := strings.TrimRight(strings.TrimSuffix(request.Host, module.Domain), ".")
		// h := strings.Split(host, ".")

		// fp = h[0] + request.URL.Path

		// // if we haven't specified a domain, do not support repo.username and branch.repo.username
		// if module.Domain != "" {
		// 	switch {
		// 	case len(h) == 2:
		// 		fp = h[1] + "/" + h[0] + request.URL.Path
		// 	case len(h) == 3:
		// 		fp = h[2] + "/" + h[1] + request.URL.Path
		// 	}
		// }
	}

	// Handle request
	content, err := module.Client.Get(organization, repository, path)
	if err != nil {
		return caddyhttp.Error(http.StatusNotFound, err)
	}

	// Try to determine mime type based on extenstion of file
	parts := strings.Split(request.URL.Path, ".")
	if len(parts) > 1 {
		extension := parts[len(parts)-1] // get file extension
		writer.Header().Add("Content-Type", mime.TypeByExtension("."+extension))
	}

	_, err = io.Copy(writer, content)

	return err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*GiteaPagesModule)(nil)
	_ caddy.Validator             = (*GiteaPagesModule)(nil)
	_ caddyhttp.MiddlewareHandler = (*GiteaPagesModule)(nil)
	_ caddyfile.Unmarshaler       = (*GiteaPagesModule)(nil)
)
