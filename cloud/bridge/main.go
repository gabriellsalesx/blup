// Command bridge é a Ponte do Blup Deals: mantém conectados os WhatsApp dos
// clientes e publica nos Canais deles quando o Blup Cloud pede. Não guarda
// nada do negócio (fila, posts, contas ficam no D1, com o Blup Cloud): só as
// chaves das sessões. Configuração em config.go; contrato em README.md.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow/store/sqlstore"
	_ "modernc.org/sqlite"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err := run(log)
	if err != nil {
		log.Error("a Ponte parou", "err", err)
		sentry.CaptureException(err)
	}
	// O que ainda não saiu para o Blup Pulse sai antes de terminar.
	sentry.Flush(5 * time.Second)
	if err != nil {
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := loadConfig(env)
	if err != nil {
		return err
	}
	if on, err := initSentry(cfg.SentryDSN, cfg.SentryEnv); err != nil {
		log.Warn("SENTRY_DSN inválida: os erros não vão para o Blup Pulse", "err", err)
	} else if on {
		log.Info("erros vão para o Blup Pulse")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, dialect, err := openDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	box := sqlstore.NewWithDB(db, dialect, waLogger(log.With("module", "store")))
	if err := box.Upgrade(ctx); err != nil {
		return fmt.Errorf("preparar o banco do whatsmeow: %w", err)
	}
	st := &ledger{db: db, postgres: dialect == "postgres"}
	if err := st.migrate(ctx); err != nil {
		return err
	}

	tell := newNotifier(cfg, log)
	m := newManager(box, st, log, tell.send, cfg)
	if err := m.resume(ctx); err != nil {
		return fmt.Errorf("religar as sessões: %w", err)
	}
	defer m.Close()

	go func() {
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for {
			_ = st.prune(ctx, time.Now())
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	srv := &http.Server{
		Addr: net.JoinHostPort(cfg.Host, cfg.Port),
		// Um pânico num pedido vira erro no Blup Pulse e segue para o
		// tratamento padrão do net/http (Repanic).
		Handler: sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(
			routes(m, newVerifier(cfg.Secret, time.Now), log, &readiness{db: db})),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	log.Info("a Ponte está no ar", "addr", srv.Addr)
	select {
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shut, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return srv.Shutdown(shut)
}

// openDB usa o Postgres de DATABASE_URL, ou o SQLite de DATA_DIR. Uma
// sessão do WhatsApp só pode estar ligada em um lugar: a Ponte roda em uma
// cópia só, e o SQLite basta.
func openDB(cfg Config) (*sql.DB, string, error) {
	if cfg.DatabaseURL != "" {
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		return db, "postgres", err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, "", fmt.Errorf("criar DATA_DIR: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+cfg.sqlitePath()+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, "", err
	}
	// Uma conexão só: o SQLite não aceita duas escritas ao mesmo tempo.
	db.SetMaxOpenConns(1)
	return db, "sqlite3", nil
}

// notifier avisa o Blup Cloud quando uma sessão muda de estado, assinando
// como os pedidos dele. Tenta três vezes; o Blup Cloud também consulta o
// estado sozinho, então um aviso perdido não trava nada.
type notifier struct {
	url    string
	secret []byte
	log    *slog.Logger
	http   *http.Client
}

func newNotifier(cfg Config, log *slog.Logger) *notifier {
	return &notifier{url: cfg.CallbackURL, secret: cfg.CallbackSecret, log: log, http: &http.Client{Timeout: 15 * time.Second}}
}

func (n *notifier) send(id, state string) {
	if n.url == "" {
		return
	}
	body, _ := json.Marshal(map[string]any{"session": id, "state": state, "at": time.Now().UTC()})
	go func() {
		for try := 0; try < 3; try++ {
			if try > 0 {
				time.Sleep(time.Duration(try*try) * 5 * time.Second)
			}
			req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			sign(req, n.secret, body, time.Now())
			res, err := n.http.Do(req)
			if err == nil {
				res.Body.Close()
				if res.StatusCode < 300 {
					return
				}
			}
		}
		n.log.Warn("aviso não chegou ao Blup Cloud", "session", id, "state", state)
	}()
}
