package main

import (
	"cmp"
	"errors"
	"fmt"
	"sync"

	"github.com/gabriellsalesx/blup/internal/i18n"
)

// The app's language: pt, en or es, picked in Account (settings.json). Until
// the screen sets it, the Blup packages and the texts below speak Portuguese,
// as the app always did. The Blup packages' own messages follow it too.
var (
	langMu     sync.Mutex
	langNow    string
	langLoaded bool
)

func validLang(l string) bool { return l == "pt" || l == "en" || l == "es" }

// appLang is the language in use, read once from the settings.
func appLang() string {
	langMu.Lock()
	defer langMu.Unlock()
	if !langLoaded {
		langLoaded = true
		if s, err := readSettings(); err == nil && validLang(s.Lang) {
			langNow = s.Lang
		}
		// The Blup packages' own messages too, once: siteAt runs for every
		// call, and several run at the same time.
		i18n.Set(i18n.Lang(cmp.Or(langNow, "pt")))
	}
	if langNow == "" {
		return "pt"
	}
	return langNow
}

// useLang changes the language in use, for this app and the Blup packages.
func useLang(l string) {
	langMu.Lock()
	langNow, langLoaded = l, true
	langMu.Unlock()
	i18n.Set(i18n.Lang(l))
}

// tr is the text of key in the app's language, formatted with args.
func tr(key string, args ...any) string {
	s, ok := texts[appLang()][key]
	if !ok {
		s = texts["en"][key]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// fail is an error whose message shows on the screen as it is.
func fail(key string, args ...any) error { return errors.New(tr(key, args...)) }

// dailyFolders is the note of the day's folder and tag in each language, the
// same as the Studio's.
var dailyFolders = map[string][2]string{"pt": {"Diário", "diario"}, "en": {"Daily", "daily"}, "es": {"Diario", "diario"}}

var texts = map[string]map[string]string{
	"pt": {
		"firstVault":     "Minhas notas", // the first vault, until renamed
		"vaultN":         "Cofre %d",
		"untitled":       "Sem título",
		"badNote":        "Nota inválida: %q",
		"badFolder":      "Pasta inválida: %q",
		"folderName":     "Escolha um nome para a pasta.",
		"folderTaken":    "Já existe uma pasta com esse nome aqui.",
		"imageFail":      "Não foi possível adicionar a imagem. Use um arquivo de imagem de até 20 MB.",
		"tagName":        "Escolha um nome para a tag.",
		"pinDigits":      "O PIN precisa ter 4 números.",
		"wrongSecret":    "Senha ou chave incorreta.",
		"notLinked":      "Este cofre ainda não está ligado ao Blup Cloud.",
		"notesLocked":    "Digite a senha das notas desta conta para sincronizar neste aparelho.",
		"shortPassword":  "Use pelo menos 8 caracteres.",
		"badVault":       "Cofre inválido.",
		"unsafeCloud":    "Endereço do Blup Cloud inseguro: %s",
		"notInVault":     "Esse arquivo não é do cofre.",
		"notBoard":       "Não é um canvas, um desenho ou um banco de dados.",
		"needOneVault":   "O aparelho precisa ter ao menos um cofre.",
		"signInDelete":   "Entre no Blup Cloud para apagar este cofre de lá também.",
		"vaultBusy":      "O cofre está sincronizando em outro aparelho. Tente de novo em instantes.",
		"cloudStuck":     "A sincronização deste cofre parou por segurança: a versão no Blup Cloud não continua a deste aparelho. Nada foi apagado.",
		"signInAgain":    "Entre no Blup Cloud de novo para apagar este cofre de lá também.",
		"cloudNoDelete":  "O Blup Cloud ainda não apaga cofres. Nada foi apagado.",
		"cloudDeleteErr": "Não deu para apagar o cofre no Blup Cloud. Confira a internet e tente de novo.",
		"vaultMissing":   "Esse cofre não está neste aparelho.",
		"vaultNameGive":  "Dê um nome ao cofre.",
		"vaultName":      "Escolha um nome para o cofre.",
		"vaultTaken":     "Já existe um cofre com esse nome.",
		"badLang":        "Idioma desconhecido: %q",
		"today":          "Hoje",
		"remStop":        "Parar",
		"remSnooze":      "Adiar 10 minutos",
		"remChannel":     "Lembretes",
		"remAlarms":      "Alarmes",
		"tomorrow":       "Amanhã",
		"inDays":         "Em %d dias (%s)",
		"dateFormat":     "02/01",
		"atTime":         "%s às %s",
		"allDay":         "%s, o dia todo",
	},
	"en": {
		"firstVault":     "My notes", // the first vault, until renamed
		"vaultN":         "Vault %d",
		"untitled":       "Untitled",
		"badNote":        "Invalid note: %q",
		"badFolder":      "Invalid folder: %q",
		"folderName":     "Pick a name for the folder.",
		"folderTaken":    "There is already a folder with that name here.",
		"imageFail":      "Couldn't add the image. Use an image file of up to 20 MB.",
		"tagName":        "Pick a name for the tag.",
		"pinDigits":      "The PIN needs 4 digits.",
		"wrongSecret":    "Wrong password or key.",
		"notLinked":      "This vault isn't linked to Blup Cloud yet.",
		"notesLocked":    "Type this account's notes password to sync on this device.",
		"shortPassword":  "Use at least 8 characters.",
		"badVault":       "Invalid vault.",
		"unsafeCloud":    "Unsafe Blup Cloud address: %s",
		"notInVault":     "That file isn't in the vault.",
		"notBoard":       "It isn't a canvas, a drawing or a database.",
		"needOneVault":   "The device needs at least one vault.",
		"signInDelete":   "Sign in to Blup Cloud to delete this vault from there too.",
		"vaultBusy":      "The vault is syncing on another device. Try again in a moment.",
		"cloudStuck":     "Syncing this vault stopped for safety: the version on Blup Cloud does not continue the one on this device. Nothing was deleted.",
		"signInAgain":    "Sign in to Blup Cloud again to delete this vault from there too.",
		"cloudNoDelete":  "Blup Cloud can't delete vaults yet. Nothing was deleted.",
		"cloudDeleteErr": "Couldn't delete the vault on Blup Cloud. Check your internet connection and try again.",
		"vaultMissing":   "That vault isn't on this device.",
		"vaultNameGive":  "Give the vault a name.",
		"vaultName":      "Pick a name for the vault.",
		"vaultTaken":     "There is already a vault with that name.",
		"badLang":        "Unknown language: %q",
		"today":          "Today",
		"remStop":        "Stop",
		"remSnooze":      "Snooze 10 minutes",
		"remChannel":     "Reminders",
		"remAlarms":      "Alarms",
		"tomorrow":       "Tomorrow",
		"inDays":         "In %d days (%s)",
		"dateFormat":     "Jan 2",
		"atTime":         "%s at %s",
		"allDay":         "%s, all day",
	},
	"es": {
		"firstVault":     "Mis notas", // the first vault, until renamed
		"vaultN":         "Bóveda %d",
		"untitled":       "Sin título",
		"badNote":        "Nota no válida: %q",
		"badFolder":      "Carpeta no válida: %q",
		"folderName":     "Elige un nombre para la carpeta.",
		"folderTaken":    "Ya hay una carpeta con ese nombre aquí.",
		"imageFail":      "No se pudo agregar la imagen. Usa un archivo de imagen de hasta 20 MB.",
		"tagName":        "Elige un nombre para la etiqueta.",
		"pinDigits":      "El PIN debe tener 4 dígitos.",
		"wrongSecret":    "Contraseña o clave incorrecta.",
		"notLinked":      "Esta bóveda aún no está vinculada a Blup Cloud.",
		"notesLocked":    "Escribe la contraseña de las notas de esta cuenta para sincronizar en este dispositivo.",
		"shortPassword":  "Usa al menos 8 caracteres.",
		"badVault":       "Bóveda inválida.",
		"unsafeCloud":    "Dirección de Blup Cloud insegura: %s",
		"notInVault":     "Ese archivo no es de la bóveda.",
		"notBoard":       "No es un lienzo, un dibujo ni una base de datos.",
		"needOneVault":   "El dispositivo necesita al menos una bóveda.",
		"signInDelete":   "Inicia sesión en Blup Cloud para borrar esta bóveda de allí también.",
		"vaultBusy":      "La bóveda se está sincronizando en otro dispositivo. Inténtalo de nuevo en un momento.",
		"cloudStuck":     "La sincronización de esta bóveda se detuvo por seguridad: la versión en Blup Cloud no continúa la de este dispositivo. No se borró nada.",
		"signInAgain":    "Vuelve a iniciar sesión en Blup Cloud para borrar esta bóveda de allí también.",
		"cloudNoDelete":  "Blup Cloud todavía no borra bóvedas. No se borró nada.",
		"cloudDeleteErr": "No se pudo borrar la bóveda en Blup Cloud. Revisa tu conexión a internet e inténtalo de nuevo.",
		"vaultMissing":   "Esa bóveda no está en este dispositivo.",
		"vaultNameGive":  "Ponle un nombre a la bóveda.",
		"vaultName":      "Elige un nombre para la bóveda.",
		"vaultTaken":     "Ya hay una bóveda con ese nombre.",
		"badLang":        "Idioma desconocido: %q",
		"today":          "Hoy",
		"remStop":        "Detener",
		"remSnooze":      "Posponer 10 minutos",
		"remChannel":     "Recordatorios",
		"remAlarms":      "Alarmas",
		"tomorrow":       "Mañana",
		"inDays":         "En %d días (%s)",
		"dateFormat":     "02/01",
		"atTime":         "%s a las %s",
		"allDay":         "%s, todo el día",
	},
}
