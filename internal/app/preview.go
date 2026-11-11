package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/hugo"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/pagefind"
	"github.com/gabriellsalesx/blup/internal/proc"
	"github.com/gabriellsalesx/blup/internal/scaffold"
	"github.com/gabriellsalesx/blup/internal/ui"
)

var serverURL = regexp.MustCompile(`https?://(localhost|127\.0\.0\.1):\d+/\S*`)

// preview runs "hugo server" so every saved note shows up in the browser
// immediately (drafts included). Ctrl+C stops it and returns to the menu.
// It renders to memory only: protected notes are shown in clear text in the
// preview, and that must never land in public/.
func preview(ctx context.Context, _ *config.Global, site *config.Site) error {
	bin, err := ensureHugo(ctx)
	if err != nil {
		return err
	}
	if err := scaffold.Write(site); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	previewSearch(ctx, bin, site)

	// The browser reaches hugo through the guard (see PreviewGuard): only
	// the preview's own pages may call it.
	guard, err := NewPreviewGuard(nil)
	if err != nil {
		return err
	}
	defer guard.Close()
	args := append([]string{"server", "--config", scaffold.PreviewConfig, "--renderToMemory", "--buildDrafts", "--navigateToChanged"},
		guard.HugoArgs()...)
	cmd := bin.Command(ctx, site.Dir(), append(args, "--logLevel", "warn")...)
	cmd.Env = append(proc.Environ(), "HUGO_BLUP_MODE=preview")
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); pw.Close() }()

	ui.Step(i18n.T("preview.starting"))
	opened := false
	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case !opened && serverURL.MatchString(line):
			url := guard.URL
			opened = true
			ui.OK(i18n.T("preview.ready", ui.Highlight(url)))
			ui.Hint(i18n.T("preview.hint"))
			ui.OpenBrowser(url)
		default:
			for _, p := range hugoProblems(line) {
				ui.Warn(p)
			}
		}
	}
	err = <-done
	if ctx.Err() != nil {
		ui.Hint(i18n.T("preview.stopped"))
		return nil
	}
	return err
}

// previewSearch builds the site once into a temporary folder and indexes it,
// so search works in the preview (with the notes as they were at start).
func previewSearch(ctx context.Context, bin hugo.Binary, site *config.Site) {
	tmp, err := os.MkdirTemp("", "blup-preview-")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmp)
	err = ui.Spin(ctx, i18n.T("preview.indexing"), func(ctx context.Context) error {
		cmd := bin.Command(ctx, site.Dir(), "--buildDrafts", "--baseURL", "http://localhost/", "--destination", tmp, "--quiet")
		cmd.Env = append(proc.Environ(), "HUGO_BLUP_MODE=preview")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		pf, err := pagefind.Ensure(ctx)
		if err != nil {
			return err
		}
		if out, err := pagefind.Index(ctx, pf, tmp); err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
		return placeSearch(site, tmp)
	})
	if err != nil {
		ui.Warn(i18n.T("build.search_failed", err))
	}
}

// placeSearch puts the index built in tmp where the preview serves it. The
// temporary folder is often on another disk (/tmp in memory), where a plain
// rename fails: moveDir copies then.
func placeSearch(site *config.Site, tmp string) error {
	dest := filepath.Join(site.Dir(), filepath.FromSlash(scaffold.PreviewStatic), "pagefind")
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return moveDir(filepath.Join(tmp, "pagefind"), dest)
}

// PreviewGuard stands between the browser and hugo server, against DNS
// rebinding: a page of another site whose name was made to point at
// 127.0.0.1 would otherwise read the preview, drafts and locked notes in
// clear text included. hugo listens on a random port of 127.0.0.1 and the
// guard on another one, which answers only requests addressed to itself
// (Host 127.0.0.1:<port> or localhost:<port>) and, when the browser says
// where a request came from (Origin), only from the preview itself or from
// the origins appOrigin accepts (the Studio's window).
type PreviewGuard struct {
	URL       string // the address to open: http://127.0.0.1:<port>/
	port      int    // the guard's
	hugo      int    // hugo server's
	appOrigin func(origin string) bool
	ln        net.Listener
	srv       *http.Server
}

// NewPreviewGuard starts the guard on a free port of 127.0.0.1, before
// hugo (whose pages must link to the guard: see HugoArgs). appOrigin may
// be nil: then only the preview's own pages are accepted.
func NewPreviewGuard(appOrigin func(origin string) bool) (*PreviewGuard, error) {
	hugoPort, err := freePort()
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	g := &PreviewGuard{port: ln.Addr().(*net.TCPAddr).Port, hugo: hugoPort, appOrigin: appOrigin, ln: ln}
	g.URL = "http://" + g.host("127.0.0.1") + "/"
	g.srv = &http.Server{Handler: g.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go g.srv.Serve(ln)
	return g, nil
}

// freePort is a port of 127.0.0.1 free right now, for hugo server.
func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func (g *PreviewGuard) host(name string) string {
	return net.JoinHostPort(name, strconv.Itoa(g.port))
}

// HugoArgs are hugo server's address flags: it listens on its own port of
// 127.0.0.1 only, while its pages (links, live reload) point to the guard.
func (g *PreviewGuard) HugoArgs() []string {
	return []string{"--bind", "127.0.0.1", "--port", strconv.Itoa(g.hugo),
		"--baseURL", g.URL, "--appendPort=false", "--liveReloadPort", strconv.Itoa(g.port)}
}

// Close stops the guard (hugo is stopped by its caller).
func (g *PreviewGuard) Close() error {
	if g == nil || g.srv == nil {
		return nil
	}
	return g.srv.Close()
}

// Handler checks each request and passes the good ones to hugo (live
// reload's WebSocket included: the reverse proxy carries upgrades).
func (g *PreviewGuard) Handler() http.Handler {
	target := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(g.hugo))}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = r.In.Host // hugo's live reload compares it with Origin
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "the preview is starting", http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := g.check(r); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		proxy.ServeHTTP(w, r)
	})
}

// check tells why a request may not reach hugo (nil when it may).
func (g *PreviewGuard) check(r *http.Request) error {
	host := strings.ToLower(r.Host)
	if host != g.host("127.0.0.1") && host != g.host("localhost") {
		return errors.New("forbidden host")
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return nil // navigations and same-origin GETs: no Origin
	}
	if u, err := url.Parse(origin); err == nil && u.Scheme == "http" && u.Path == "" &&
		(strings.EqualFold(u.Host, g.host("127.0.0.1")) || strings.EqualFold(u.Host, g.host("localhost"))) {
		return nil
	}
	if g.appOrigin != nil && g.appOrigin(origin) {
		return nil
	}
	return errors.New("forbidden origin")
}
