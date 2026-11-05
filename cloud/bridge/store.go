package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// O que a Ponte guarda além das chaves do whatsmeow: qual sessão do Blup é
// qual aparelho do WhatsApp, a hora do último post de cada Canal (o modo
// segurança) e os posts já feitos (um pedido repetido não publica duas
// vezes). Nada do negócio: isso mora no D1, com o Blup Cloud.
type ledger struct {
	db       *sql.DB
	postgres bool
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS bridge_session(id TEXT PRIMARY KEY, jid TEXT NOT NULL, created_at BIGINT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS bridge_gap(session TEXT NOT NULL, channel TEXT NOT NULL, last_at BIGINT NOT NULL, PRIMARY KEY(session, channel))`,
	`CREATE TABLE IF NOT EXISTS bridge_sent(key TEXT PRIMARY KEY, session TEXT NOT NULL, message_id TEXT NOT NULL, sent_at BIGINT NOT NULL)`,
	`CREATE INDEX IF NOT EXISTS bridge_sent_at ON bridge_sent(sent_at)`,
}

func (s *ledger) migrate(ctx context.Context) error {
	for _, q := range schema {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migrar: %w", err)
		}
	}
	return nil
}

// q troca os "?" pelos "$1, $2..." do Postgres.
func (s *ledger) q(query string) string {
	if !s.postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

type savedSession struct{ ID, JID string }

func (s *ledger) sessions(ctx context.Context) ([]savedSession, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, jid FROM bridge_session`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []savedSession
	for rows.Next() {
		var v savedSession
		if err := rows.Scan(&v.ID, &v.JID); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *ledger) saveSession(ctx context.Context, id, jid string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO bridge_session(id, jid, created_at) VALUES(?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET jid = excluded.jid`), id, jid, now.UnixMilli())
	return err
}

func (s *ledger) forget(ctx context.Context, id string) error {
	for _, q := range []string{`DELETE FROM bridge_session WHERE id = ?`, `DELETE FROM bridge_gap WHERE session = ?`, `DELETE FROM bridge_sent WHERE session = ?`} {
		if _, err := s.db.ExecContext(ctx, s.q(q), id); err != nil {
			return err
		}
	}
	return nil
}

// claim reserva a vez do Canal: só passa se o último post foi há pelo menos
// gap. Devolve quanto falta quando não passa, e o último horário de antes,
// para devolver a vez se o envio falhar.
func (s *ledger) claim(ctx context.Context, session, channel string, now time.Time, gap time.Duration) (ok bool, wait time.Duration, prev int64, err error) {
	at := now.UnixMilli()
	err = s.db.QueryRowContext(ctx, s.q(`SELECT last_at FROM bridge_gap WHERE session = ? AND channel = ?`), session, channel).Scan(&prev)
	if errors.Is(err, sql.ErrNoRows) {
		prev, err = 0, nil
	}
	if err != nil {
		return false, 0, 0, err
	}
	res, err := s.db.ExecContext(ctx, s.q(`INSERT INTO bridge_gap(session, channel, last_at) VALUES(?, ?, ?)
		ON CONFLICT(session, channel) DO UPDATE SET last_at = excluded.last_at WHERE bridge_gap.last_at <= ?`),
		session, channel, at, at-gap.Milliseconds())
	if err != nil {
		return false, 0, 0, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return true, 0, prev, nil
	}
	return false, time.UnixMilli(prev).Add(gap).Sub(now), prev, nil
}

// release devolve a vez quando o post não saiu.
func (s *ledger) release(ctx context.Context, session, channel string, claimed time.Time, prev int64) error {
	if prev == 0 {
		_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM bridge_gap WHERE session = ? AND channel = ? AND last_at = ?`), session, channel, claimed.UnixMilli())
		return err
	}
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE bridge_gap SET last_at = ? WHERE session = ? AND channel = ? AND last_at = ?`), prev, session, channel, claimed.UnixMilli())
	return err
}

func (s *ledger) sent(ctx context.Context, key string) (Sent, bool, error) {
	var id string
	var at int64
	err := s.db.QueryRowContext(ctx, s.q(`SELECT message_id, sent_at FROM bridge_sent WHERE key = ?`), key).Scan(&id, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return Sent{}, false, nil
	}
	return Sent{ID: id, At: time.UnixMilli(at)}, err == nil, err
}

func (s *ledger) markSent(ctx context.Context, key, session string, p Sent) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO bridge_sent(key, session, message_id, sent_at) VALUES(?, ?, ?, ?) ON CONFLICT(key) DO NOTHING`), key, session, p.ID, p.At.UnixMilli())
	return err
}

// prune esquece os posts feitos há mais de uma semana: depois disso o Blup
// Cloud não repete mais o pedido.
func (s *ledger) prune(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM bridge_sent WHERE sent_at < ?`), now.Add(-7*24*time.Hour).UnixMilli())
	return err
}
