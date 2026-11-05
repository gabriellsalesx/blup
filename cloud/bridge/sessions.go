package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Os estados de uma sessão, como o Blup Cloud os vê.
const (
	statePairing      = "pairing"      // esperando ler o QR
	stateConnected    = "connected"    // pronta para publicar
	stateDisconnected = "disconnected" // caiu e está voltando sozinha
	stateExpired      = "expired"      // o QR venceu sem ser lido
	stateLoggedOut    = "logged_out"   // a pessoa desconectou o aparelho
	stateBanned       = "banned"       // o WhatsApp suspendeu o número
	stateReplaced     = "replaced"     // a mesma sessão abriu em outro lugar
	stateOutdated     = "outdated"     // o whatsmeow precisa de atualização
)

var (
	errUnknown      = errors.New("sessão desconhecida")
	errNotConnected = errors.New("sessão não conectada")
	errNotAdmin     = errors.New("não é dono nem administrador do Canal")
	errTooSoon      = errors.New("cedo demais para este Canal")
	errBusy         = errors.New("muitas conexões esperando o QR")
)

// View é o que o Blup Cloud recebe sobre uma sessão. O QR só existe
// enquanto espera ser lido; do número, só os quatro últimos dígitos.
type View struct {
	State       string     `json:"state"`
	QR          string     `json:"qr,omitempty"`
	QRExpiresAt *time.Time `json:"qrExpiresAt,omitempty"`
	PhoneEnd    string     `json:"phoneEnd,omitempty"`
}

type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Post struct {
	Channel string
	Key     string
	Text    string
	Image   []byte
}

type Sent struct {
	ID string    `json:"id"`
	At time.Time `json:"at"`
}

// TooSoon diz quanto falta para o Canal aceitar outro post.
type TooSoon struct{ Wait time.Duration }

func (e TooSoon) Error() string { return errTooSoon.Error() }
func (e TooSoon) Unwrap() error { return errTooSoon }

type session struct {
	id      string
	mu      sync.Mutex
	cli     *whatsmeow.Client
	state   string
	qr      string
	qrUntil time.Time
	stop    context.CancelFunc
	// Os Canais em que pode publicar, lidos há pouco.
	channels   map[string]Channel
	channelsAt time.Time
}

type manager struct {
	box    *sqlstore.Container
	db     *ledger
	log    *slog.Logger
	notify func(id, state string)
	gap    time.Duration
	now    func() time.Time
	mu     sync.Mutex
	all    map[string]*session
}

const maxPairing = 20

func newManager(box *sqlstore.Container, db *ledger, log *slog.Logger, notify func(id, state string), gap time.Duration, deviceName string) *manager {
	store.SetOSInfo(deviceName, [3]uint32{1, 0, 0})
	// A Ponte só publica: não precisa do histórico das conversas.
	store.DeviceProps.RequireFullSync = proto.Bool(false)
	return &manager{box: box, db: db, log: log, notify: notify, gap: gap, now: time.Now, all: map[string]*session{}}
}

func (m *manager) client(device *store.Device, id string) *whatsmeow.Client {
	cli := whatsmeow.NewClient(device, waLogger(m.log.With("session", id)))
	cli.ManualHistorySyncDownload = true
	return cli
}

// resume liga de novo as sessões já conectadas, ao subir.
func (m *manager) resume(ctx context.Context) error {
	saved, err := m.db.sessions(ctx)
	if err != nil {
		return err
	}
	for _, s := range saved {
		jid, err := types.ParseJID(s.JID)
		var device *store.Device
		if err == nil {
			device, err = m.box.GetDevice(ctx, jid)
		}
		if err != nil || device == nil {
			m.log.Warn("sessão sem aparelho, esquecida", "session", s.ID)
			_ = m.db.forget(ctx, s.ID)
			m.notify(s.ID, stateLoggedOut)
			continue
		}
		ss := &session{id: s.ID, state: stateDisconnected}
		ss.cli = m.client(device, s.ID)
		ss.cli.AddEventHandler(m.events(ss))
		m.mu.Lock()
		m.all[s.ID] = ss
		m.mu.Unlock()
		if err := ss.cli.Connect(); err != nil {
			m.log.Warn("não conectou ao subir", "session", s.ID, "err", err)
		}
	}
	return nil
}

// Start abre o QR de uma sessão nova, ou devolve como ela está.
func (m *manager) Start(ctx context.Context, id string) (View, error) {
	m.mu.Lock()
	ss := m.all[id]
	if ss != nil {
		ss.mu.Lock()
		alive := ss.state != stateExpired && ss.state != stateLoggedOut
		ss.mu.Unlock()
		if alive {
			m.mu.Unlock()
			return m.view(ss), nil
		}
	}
	pairing := 0
	for _, s := range m.all {
		s.mu.Lock()
		if s.state == statePairing {
			pairing++
		}
		s.mu.Unlock()
	}
	if pairing >= maxPairing {
		m.mu.Unlock()
		return View{}, errBusy
	}
	ss = &session{id: id, state: statePairing}
	ss.cli = m.client(m.box.NewDevice(), id)
	ss.cli.AddEventHandler(m.events(ss))
	m.all[id] = ss
	m.mu.Unlock()

	pairCtx, stop := context.WithTimeout(context.Background(), 5*time.Minute)
	ss.stop = stop
	qr, err := ss.cli.GetQRChannel(pairCtx)
	if err == nil {
		err = ss.cli.Connect()
	}
	if err != nil {
		stop()
		m.drop(id)
		return View{}, fmt.Errorf("abrir o QR: %w", err)
	}
	ready := make(chan struct{}, 1)
	go m.pair(ss, qr, ready, stop)
	// Espera o primeiro QR, para já devolver com ele.
	select {
	case <-ready:
	case <-time.After(15 * time.Second):
	case <-ctx.Done():
	}
	return m.view(ss), nil
}

func (m *manager) pair(ss *session, qr <-chan whatsmeow.QRChannelItem, ready chan<- struct{}, stop context.CancelFunc) {
	defer stop()
	for item := range qr {
		switch item.Event {
		case "code":
			png, err := qrcode.Encode(item.Code, qrcode.Medium, 512)
			ss.mu.Lock()
			if err == nil {
				ss.qr = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
				ss.qrUntil = m.now().Add(item.Timeout)
			}
			ss.mu.Unlock()
			select {
			case ready <- struct{}{}:
			default:
			}
		case "success":
			ss.mu.Lock()
			ss.qr, ss.qrUntil = "", time.Time{}
			ss.mu.Unlock()
			// O aparelho só é salvo no PairSuccess; aqui só confirma.
		default:
			ss.mu.Lock()
			paired := ss.state != statePairing
			if !paired {
				ss.state, ss.qr, ss.qrUntil = stateExpired, "", time.Time{}
			}
			ss.mu.Unlock()
			if !paired {
				ss.cli.Disconnect()
				m.log.Info("QR não foi lido", "session", ss.id, "event", item.Event)
				m.notify(ss.id, stateExpired)
			}
			select {
			case ready <- struct{}{}:
			default:
			}
		}
	}
}

func (m *manager) events(ss *session) func(any) {
	return func(evt any) {
		switch e := evt.(type) {
		case *events.PairSuccess:
			if err := m.db.saveSession(context.Background(), ss.id, e.ID.String(), m.now()); err != nil {
				m.log.Error("não salvou a sessão", "session", ss.id, "err", err)
			}
			m.set(ss, stateConnected, true)
		case *events.Connected:
			if ss.cli.Store.ID != nil {
				m.set(ss, stateConnected, false)
			}
		case *events.Disconnected:
			m.set(ss, stateDisconnected, false)
		case *events.LoggedOut:
			m.end(ss, stateLoggedOut)
		case *events.TemporaryBan:
			m.log.Warn("número suspenso pelo WhatsApp", "session", ss.id, "code", e.Code, "expire", e.Expire)
			m.set(ss, stateBanned, true)
		case *events.StreamReplaced:
			m.set(ss, stateReplaced, true)
		case *events.ClientOutdated:
			m.log.Error("o whatsmeow precisa de atualização", "session", ss.id)
			m.set(ss, stateOutdated, true)
		}
	}
}

// set muda o estado; tell avisa o Blup Cloud.
func (m *manager) set(ss *session, state string, tell bool) {
	ss.mu.Lock()
	before := ss.state
	// Suspensão, troca e versão velha só saem com uma conexão nova.
	stuck := before == stateBanned || before == stateReplaced || before == stateOutdated
	if !stuck || tell {
		ss.state = state
	}
	ss.mu.Unlock()
	if tell && before != state {
		m.notify(ss.id, state)
	}
}

// end encerra de vez uma sessão desconectada pelo celular.
func (m *manager) end(ss *session, state string) {
	ctx := context.Background()
	ss.cli.Disconnect()
	if ss.cli.Store.ID != nil {
		_ = ss.cli.Store.Delete(ctx)
	}
	_ = m.db.forget(ctx, ss.id)
	m.set(ss, state, true)
}

func (m *manager) drop(id string) {
	m.mu.Lock()
	delete(m.all, id)
	m.mu.Unlock()
}

func (m *manager) find(id string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ss := m.all[id]
	if ss == nil {
		return nil, errUnknown
	}
	return ss, nil
}

func (m *manager) View(id string) (View, error) {
	ss, err := m.find(id)
	if err != nil {
		return View{}, err
	}
	return m.view(ss), nil
}

func (m *manager) view(ss *session) View {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	v := View{State: ss.state}
	if ss.state == statePairing && ss.qr != "" && m.now().Before(ss.qrUntil) {
		until := ss.qrUntil
		v.QR, v.QRExpiresAt = ss.qr, &until
	}
	if id := ss.cli.Store.ID; id != nil && len(id.User) >= 4 {
		v.PhoneEnd = id.User[len(id.User)-4:]
	}
	return v
}

func (m *manager) connected(id string) (*session, error) {
	ss, err := m.find(id)
	if err != nil {
		return nil, err
	}
	ss.mu.Lock()
	ok := ss.state == stateConnected
	ss.mu.Unlock()
	if !ok || !ss.cli.IsLoggedIn() {
		return nil, errNotConnected
	}
	return ss, nil
}

// Channels lista só os Canais em que a sessão é dona ou administradora:
// os que ela só segue, e grupos, nunca aparecem.
func (m *manager) Channels(ctx context.Context, id string) ([]Channel, error) {
	ss, err := m.connected(id)
	if err != nil {
		return nil, err
	}
	all, err := ss.cli.GetSubscribedNewsletters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listar Canais: %w", err)
	}
	out := []Channel{}
	mine := map[string]Channel{}
	for _, n := range all {
		if n.ViewerMeta == nil || (n.ViewerMeta.Role != types.NewsletterRoleOwner && n.ViewerMeta.Role != types.NewsletterRoleAdmin) {
			continue
		}
		c := Channel{ID: n.ID.User, Name: n.ThreadMeta.Name.Text, Role: string(n.ViewerMeta.Role)}
		out = append(out, c)
		mine[c.ID] = c
	}
	ss.mu.Lock()
	ss.channels, ss.channelsAt = mine, m.now()
	ss.mu.Unlock()
	return out, nil
}

func (m *manager) canPost(ctx context.Context, ss *session, channel string) error {
	ss.mu.Lock()
	_, ok := ss.channels[channel]
	fresh := m.now().Sub(ss.channelsAt) < 10*time.Minute
	ss.mu.Unlock()
	if ok && fresh {
		return nil
	}
	if _, err := m.Channels(ctx, ss.id); err != nil {
		return err
	}
	ss.mu.Lock()
	_, ok = ss.channels[channel]
	ss.mu.Unlock()
	if !ok {
		return errNotAdmin
	}
	return nil
}

// Publish publica no Canal, uma vez por chave e no máximo um post a cada
// gap: a segunda trava do modo segurança, mesmo que o Blup Cloud erre.
func (m *manager) Publish(ctx context.Context, id string, p Post) (Sent, error) {
	if done, ok, err := m.db.sent(ctx, p.Key); err != nil || ok {
		return done, err
	}
	ss, err := m.connected(id)
	if err != nil {
		return Sent{}, err
	}
	if err := m.canPost(ctx, ss, p.Channel); err != nil {
		return Sent{}, err
	}
	now := m.now()
	ok, wait, prev, err := m.db.claim(ctx, id, p.Channel, now, m.gap)
	if err != nil {
		return Sent{}, err
	}
	if !ok {
		return Sent{}, TooSoon{Wait: wait}
	}
	sent, err := m.send(ctx, ss, p)
	if err != nil {
		_ = m.db.release(context.Background(), id, p.Channel, now, prev)
		return Sent{}, err
	}
	if err := m.db.markSent(ctx, p.Key, id, sent); err != nil {
		m.log.Error("post feito mas não anotado", "session", id, "err", err)
	}
	return sent, nil
}

func (m *manager) send(ctx context.Context, ss *session, p Post) (Sent, error) {
	to := types.NewJID(p.Channel, types.NewsletterServer)
	var msg *waE2E.Message
	var extra []whatsmeow.SendRequestExtra
	if len(p.Image) > 0 {
		up, err := ss.cli.UploadNewsletter(ctx, p.Image, whatsmeow.MediaImage)
		if err != nil {
			return Sent{}, fmt.Errorf("enviar a imagem: %w", err)
		}
		img := &waE2E.ImageMessage{
			Caption: proto.String(p.Text), Mimetype: proto.String(imageType(p.Image)),
			URL: &up.URL, DirectPath: &up.DirectPath, FileSHA256: up.FileSHA256, FileLength: &up.FileLength,
		}
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(p.Image)); err == nil {
			img.Width, img.Height = proto.Uint32(uint32(cfg.Width)), proto.Uint32(uint32(cfg.Height))
		}
		msg = &waE2E.Message{ImageMessage: img}
		extra = append(extra, whatsmeow.SendRequestExtra{MediaHandle: up.Handle})
	} else {
		msg = &waE2E.Message{Conversation: proto.String(p.Text)}
	}
	r, err := ss.cli.SendMessage(ctx, to, msg, extra...)
	if err != nil {
		return Sent{}, fmt.Errorf("publicar: %w", err)
	}
	return Sent{ID: r.ID, At: r.Timestamp}, nil
}

// Remove desconecta o aparelho e apaga a sessão, quando a pessoa tira o
// canal do Blup ou apaga a conta.
func (m *manager) Remove(ctx context.Context, id string) error {
	ss, err := m.find(id)
	if errors.Is(err, errUnknown) {
		return m.db.forget(ctx, id)
	}
	if ss.stop != nil {
		ss.stop()
	}
	if ss.cli.IsLoggedIn() {
		if err := ss.cli.Logout(ctx); err != nil {
			m.log.Warn("logout falhou, apagando assim mesmo", "session", id, "err", err)
		}
	}
	ss.cli.Disconnect()
	if ss.cli.Store.ID != nil {
		_ = ss.cli.Store.Delete(ctx)
	}
	m.drop(id)
	return m.db.forget(ctx, id)
}

func (m *manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ss := range m.all {
		ss.cli.Disconnect()
	}
}

func imageType(b []byte) string {
	if bytes.HasPrefix(b, []byte("\x89PNG")) {
		return "image/png"
	}
	return "image/jpeg"
}

// waLogger leva os registros do whatsmeow para o slog, sem o nível debug.
type slogWA struct{ l *slog.Logger }

func waLogger(l *slog.Logger) waLog.Logger { return slogWA{l} }

func (w slogWA) Warnf(f string, a ...any)  { w.l.Warn(fmt.Sprintf(f, a...)) }
func (w slogWA) Errorf(f string, a ...any) { w.l.Error(fmt.Sprintf(f, a...)) }
func (w slogWA) Infof(f string, a ...any)  {}
func (w slogWA) Debugf(f string, a ...any) {}
func (w slogWA) Sub(module string) waLog.Logger {
	return slogWA{w.l.With("module", strings.ToLower(module))}
}
