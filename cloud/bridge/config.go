package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// Config vem toda de variáveis de ambiente, com nomes nossos: a Ponte roda
// igual em qualquer servidor, sem nada de uma plataforma.
type Config struct {
	Host           string        // HOST: onde escuta (127.0.0.1; 0.0.0.0 quando o servidor exige)
	Port           string        // PORT
	DataDir        string        // DATA_DIR: pasta do SQLite com as sessões
	DatabaseURL    string        // DATABASE_URL: Postgres no lugar do SQLite (opcional)
	Secret         []byte        // BRIDGE_SECRET: assina os pedidos do Blup Cloud
	CallbackURL    string        // CALLBACK_URL: para onde vão os avisos de sessão
	CallbackSecret []byte        // CALLBACK_SECRET: assina os avisos
	MinGap         time.Duration // MIN_GAP: o intervalo mínimo entre posts no mesmo Canal
	SessionGap     time.Duration // SESSION_GAP: o intervalo mínimo entre posts do mesmo número
	DeviceName     string        // DEVICE_NAME: o nome em "Aparelhos conectados"
	DevicePlatform string        // DEVICE_PLATFORM: o ícone ao lado do nome
}

const minSecret = 32

func loadConfig(get func(string) string) (Config, error) {
	c := Config{
		Host:           or(get("HOST"), "127.0.0.1"),
		Port:           or(get("PORT"), "8080"),
		DataDir:        or(get("DATA_DIR"), "data"),
		DatabaseURL:    get("DATABASE_URL"),
		Secret:         []byte(get("BRIDGE_SECRET")),
		CallbackURL:    get("CALLBACK_URL"),
		DeviceName:     or(get("DEVICE_NAME"), "Blup Deals"),
		MinGap:         30 * time.Minute,
		SessionGap:     5 * time.Minute,
		DevicePlatform: or(get("DEVICE_PLATFORM"), "desktop"),
	}
	c.CallbackSecret = []byte(get("CALLBACK_SECRET"))
	if len(c.Secret) < minSecret {
		return c, fmt.Errorf("BRIDGE_SECRET precisa de ao menos %d caracteres", minSecret)
	}
	if c.CallbackURL != "" {
		u, err := url.Parse(c.CallbackURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return c, errors.New("CALLBACK_URL precisa ser um endereço https")
		}
		if len(c.CallbackSecret) < minSecret {
			return c, fmt.Errorf("CALLBACK_SECRET precisa de ao menos %d caracteres", minSecret)
		}
	}
	if v := get("MIN_GAP"); v != "" {
		d, err := time.ParseDuration(v)
		// Menos de 30 minutos não: é o modo segurança do Canal.
		if err != nil || d < 30*time.Minute {
			return c, errors.New("MIN_GAP precisa ser uma duração de 30m ou mais")
		}
		c.MinGap = d
	}
	if v := get("SESSION_GAP"); v != "" {
		d, err := time.ParseDuration(v)
		// Canais do mesmo número nunca recebem posts no mesmo instante.
		if err != nil || d < 5*time.Minute {
			return c, errors.New("SESSION_GAP precisa ser uma duração de 5m ou mais")
		}
		c.SessionGap = d
	}
	if _, ok := platforms[c.DevicePlatform]; !ok {
		return c, errors.New("DEVICE_PLATFORM aceita desktop, chrome, firefox, edge ou safari")
	}
	if len(c.DeviceName) > 40 {
		return c, errors.New("DEVICE_NAME aceita até 40 caracteres")
	}
	return c, nil
}

func (c Config) sqlitePath() string { return filepath.Join(c.DataDir, "bridge.db") }

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func env(key string) string { return os.Getenv(key) }
