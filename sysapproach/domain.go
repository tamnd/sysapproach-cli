package sysapproach

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes sysapproach as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/sysapproach-cli/sysapproach"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// sysapproach:// URIs by routing to the operations Register installs. The same
// Domain also builds the standalone sysapproach binary (see cli.NewApp), so the
// binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the sysapproach driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "sysapproach",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "sysapproach",
			Short:  "Browse Computer Networks: A Systems Approach from the command line.",
			Long: `Browse Computer Networks: A Systems Approach from the command line.

sysapproach reads public data from book.systemsapproach.org over HTTPS, shapes
it into clean records, and prints output that pipes into the rest of your tools.
No API key, nothing to run alongside it.`,
			Site: Host,
			Repo: "https://github.com/tamnd/sysapproach-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// chapters: list all book chapters.
	kit.Handle(app, kit.OpMeta{Name: "chapters", Group: "read", List: true,
		Summary: "List all chapters of Computer Networks: A Systems Approach"},
		listChapters)

	// page: resolver op — fetch one page by path or URL.
	kit.Handle(app, kit.OpMeta{Name: "page", Group: "read", Single: true,
		Summary: "Fetch a page by path or URL", URIType: "page", Resolver: true,
		Args: []kit.Arg{{Name: "ref", Help: "page path or URL"}}}, getPage)

	// links: list the pages a page links to.
	kit.Handle(app, kit.OpMeta{Name: "links", Group: "read", List: true,
		Summary: "List the pages a page links to", URIType: "page",
		Args: []kit.Arg{{Name: "ref", Help: "page path or URL"}}}, listLinks)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	dcfg := DefaultConfig()
	if cfg.UserAgent != "" {
		dcfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		dcfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		dcfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		dcfg.Timeout = cfg.Timeout
	}
	return NewClient(dcfg), nil
}

// --- input types ---

type chaptersIn struct {
	Client *Client `kit:"inject"`
}

type pageRef struct {
	Ref    string  `kit:"arg" help:"page path or URL"`
	Client *Client `kit:"inject"`
}

type listRef struct {
	Ref    string  `kit:"arg" help:"page path or URL"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listChapters(ctx context.Context, in chaptersIn, emit func(*Chapter) error) error {
	chapters, err := in.Client.Chapters(ctx)
	if err != nil {
		return mapErr(err)
	}
	for _, ch := range chapters {
		if err := emit(ch); err != nil {
			return err
		}
	}
	return nil
}

// Page is a generic page record used by the page/links ops.
type Page struct {
	ID    string `json:"id" kit:"id"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty" kit:"body"`
}

func getPage(ctx context.Context, in pageRef, emit func(*Page) error) error {
	path := pagePath(in.Ref)
	url := DefaultConfig().BaseURL + "/" + path
	body, err := in.Client.get(ctx, url)
	if err != nil {
		return mapErr(err)
	}
	return emit(&Page{ID: path, URL: url, Title: path, Body: pageText(body)})
}

func listLinks(ctx context.Context, in listRef, emit func(*Page) error) error {
	baseURL := DefaultConfig().BaseURL
	path := pagePath(in.Ref)
	body, err := in.Client.get(ctx, baseURL+"/"+path)
	if err != nil {
		return mapErr(err)
	}
	seen := map[string]bool{}
	count := 0
	for _, p := range linkPaths(body) {
		if seen[p] {
			continue
		}
		seen[p] = true
		if err := emit(&Page{ID: p, URL: baseURL + "/" + p}); err != nil {
			return err
		}
		count++
		if in.Limit > 0 && count >= in.Limit {
			break
		}
	}
	return nil
}

// --- Resolver: URI string functions, pure and network-free ---

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = pagePath(input)
	if id == "" {
		return "", "", errs.Usage("unrecognized sysapproach reference: %q", input)
	}
	return "page", id, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "page" {
		return "", errs.Usage("sysapproach has no resource type %q", uriType)
	}
	return DefaultConfig().BaseURL + "/" + strings.Trim(id, "/"), nil
}

// --- helpers ---

func pagePath(input string) string {
	input = strings.TrimSpace(input)
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return strings.Trim(u.Path, "/")
	}
	return strings.Trim(input, "/")
}

var hrefRE = regexp.MustCompile(`href="(/[^":#?]+)"`)

func linkPaths(body []byte) []string {
	var out []string
	for _, m := range hrefRE.FindAllSubmatch(body, -1) {
		if p := strings.Trim(string(m[1]), "/"); p != "" {
			out = append(out, p)
		}
	}
	return out
}

var tagRE = regexp.MustCompile(`<[^>]+>`)

func pageText(body []byte) string {
	s := strings.Join(strings.Fields(tagRE.ReplaceAllString(string(body), " ")), " ")
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}

func mapErr(err error) error {
	return err
}

// satisfy compile: time is used by newClient's cfg.Timeout check via kit.Config
var _ = time.Second
