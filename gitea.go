package gitea

import (
	"fmt"
	"io"
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
	Client             *gitea.Client `json:"-"`
	Server             string        `json:"server,omitempty"`
	Token              string        `json:"token,omitempty"`
	GiteaPages         string        `json:"gitea_pages,omitempty"`
	GiteaPagesAllowAll string        `json:"gitea_pages_allowall,omitempty"`
	Domain             string        `json:"domain,omitempty"`
	Simple             string        `json:"simple,omitempty"`
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

	module.Client, err = gitea.NewClient(logger, module.Server, module.Token, module.GiteaPages, module.GiteaPagesAllowAll)

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
			case "gitea_pages_allowall":
				d.Args(&module.GiteaPagesAllowAll)
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
func (module GiteaPagesModule) ServeHTTP(w http.ResponseWriter, r *http.Request, _ caddyhttp.Handler) error {

	fmt.Println("URL " + r.URL.Path)

	var fp, ref string
	if module.Simple != "" {
		fp = strings.TrimPrefix(r.URL.Path, "/") // we need to trim the leading prefix because the rest of the module is too stupid
		ref = r.URL.Query().Get("ref")
	} else {
		fmt.Println("NON SIMPLE SETUP")
		// remove the domain if it's set (works fine if it's empty)
		host := strings.TrimRight(strings.TrimSuffix(r.Host, module.Domain), ".")
		h := strings.Split(host, ".")

		fp = h[0] + r.URL.Path
		ref = r.URL.Query().Get("ref")

		// if we haven't specified a domain, do not support repo.username and branch.repo.username
		if module.Domain != "" {
			switch {
			case len(h) == 2:
				fp = h[1] + "/" + h[0] + r.URL.Path
			case len(h) == 3:
				fp = h[2] + "/" + h[1] + r.URL.Path
				ref = h[0]
			}
		}
	}

	f, err := module.Client.Open(fp, ref)
	if err != nil {
		return caddyhttp.Error(http.StatusNotFound, err)
	}

	// try to determine mime type based on extenstion of file
	parts := strings.Split(r.URL.Path, ".")
	var ext string
	if len(parts) > 1 {
		ext = parts[len(parts)-1]
		// fmt.Println(ext)
		w.Header().Add("Content-Type", mime.TypeByExtension("."+ext))
	}

	// w.Header().Add("Content-Type", mime.TypeByExtension(".css"))

	_, err = io.Copy(w, f)

	return err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*GiteaPagesModule)(nil)
	_ caddy.Validator             = (*GiteaPagesModule)(nil)
	_ caddyhttp.MiddlewareHandler = (*GiteaPagesModule)(nil)
	_ caddyfile.Unmarshaler       = (*GiteaPagesModule)(nil)
)
