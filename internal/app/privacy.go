package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The privacy policy is a note at the top of the site folder with
// "blup: privacy". The theme links it from the cookie banner and the footer
// and shows it as a plain page. Blup writes a starting point from the
// site's tools; the note is the user's from then on.

// privacyNote is the site's privacy policy note, "" when there is none.
func privacyNote(site *config.Site) string {
	files, _ := filepath.Glob(filepath.Join(site.ContentDir(), "*.md"))
	for _, f := range files {
		if n, err := obsidian.ReadNote(f); err == nil && strings.EqualFold(n.String("blup"), "privacy") {
			return f
		}
	}
	return ""
}

// writePrivacyNote creates the privacy policy note from the site's tools,
// unless the site has one. It returns the note's path.
func writePrivacyNote(site *config.Site, contact string) (string, error) {
	if p := privacyNote(site); p != "" {
		return p, nil
	}
	name := map[string]string{"pt": "Política de privacidade.md"}[site.Lang]
	if name == "" {
		name = "Privacy policy.md"
	}
	path := filepath.Join(site.ContentDir(), name)
	if _, err := os.Stat(path); err == nil {
		// A note with that name that isn't the policy: don't touch it.
		path = filepath.Join(site.ContentDir(), strings.TrimSuffix(name, ".md")+" (blup).md")
	}
	_, err := obsidian.WriteNoteIfMissing(path, privacyBody(site, contact, time.Now()))
	return path, err
}

// privacyText holds the policy's sentences in one language.
type privacyText struct {
	title, description, todo, updated, dateFormat           string
	who, whoText, contactTodo                               string
	collect, collectNone, collectIntro                      string
	ga, gtm, umami, plausible, pixel, youtube, comments     string
	browser, browserText                                    string
	hosting, hostingCloudflare, hostingGitHub, hostingOther string
	choices, choicesText, choicesBanner                     string
	changes, changesText                                    string
}

var privacyTexts = map[string]privacyText{
	"pt": {
		title:             "Política de privacidade",
		description:       "Como este site trata os dados de quem visita.",
		todo:              "%%\nModelo criado pelo Blup a partir das ferramentas do site. Revise cada parte, complete o que faltar e apague o que não se aplica. Este comentário não aparece no site.\nSe trocar as ferramentas em blup personalizar, atualize esta página.\n%%",
		updated:           "Última atualização: %s.",
		dateFormat:        "02/01/2006",
		who:               "Quem cuida deste site",
		whoText:           "Este site é mantido por %s. Para qualquer assunto sobre privacidade, escreva para %s.",
		contactTodo:       "[seu e-mail de contato]",
		collect:           "O que é coletado",
		collectNone:       "Este site não pede cadastro e não usa ferramentas de estatística ou de anúncios.",
		collectIntro:      "Este site não pede cadastro. Os dados abaixo vêm das ferramentas que ele usa:",
		ga:                "**Google Analytics**: conta visitas, páginas vistas, tempo no site, tipo de aparelho e região aproximada. Usa cookies do Google e só carrega se você aceitar **Estatísticas** no aviso de cookies. Veja a [política de privacidade do Google](https://policies.google.com/privacy).",
		gtm:               "**Google Tag Manager**: carrega as tags de medição e de anúncios configuradas no Google. Só carrega se você aceitar **Estatísticas** ou **Marketing**, e as tags do Google seguem cada escolha. Veja a [política de privacidade do Google](https://policies.google.com/privacy).",
		umami:             "**Umami**: conta visitas e páginas vistas de forma agregada, sem cookies e sem identificar você. Só carrega se você aceitar **Estatísticas**.",
		plausible:         "**Plausible**: conta visitas e páginas vistas de forma agregada, sem cookies e sem identificar você. Só carrega se você aceitar **Estatísticas**.",
		pixel:             "**Meta Pixel**: mede o resultado de anúncios no Facebook e no Instagram. Usa cookies da Meta e só carrega se você aceitar **Marketing**. Veja a [política de privacidade da Meta](https://www.facebook.com/privacy/policy).",
		youtube:           "**Vídeos do YouTube**: vêm do endereço youtube-nocookie.com e só carregam se você aceitar **Vídeos e conteúdo externo** ou clicar em **Carregar vídeo**. Depois disso, o YouTube pode usar cookies. Veja a [política de privacidade do Google](https://policies.google.com/privacy).",
		comments:          "**Comentários e reações**: para comentar ou reagir, você entra com a sua conta do GitHub. O comentário, o seu nome de usuário e a sua foto ficam públicos no GitHub. Veja a [declaração de privacidade do GitHub](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement).",
		browser:           "O que fica no seu navegador",
		browserText:       "Para o site funcionar, o seu navegador guarda algumas informações que nunca são enviadas para nós:\n\n- a sua escolha de tema claro ou escuro;\n- a sua escolha no aviso de cookies, por um ano;\n- a conexão com o GitHub, se você entrou para comentar, até você sair;\n- uma cópia do site, para ele abrir sem internet quando instalado como app.",
		hosting:           "Hospedagem",
		hostingCloudflare: "O site é hospedado no Cloudflare Pages, que registra dados técnicos de acesso, como endereço IP e navegador, para entregar as páginas e proteger o site contra ataques. Veja a [política de privacidade da Cloudflare](https://www.cloudflare.com/privacypolicy/).",
		hostingGitHub:     "O site é hospedado no GitHub Pages, que registra dados técnicos de acesso, como endereço IP, para entregar as páginas e proteger o site contra ataques. Veja a [declaração de privacidade do GitHub](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement).",
		hostingOther:      "[Diga onde o site está hospedado e o que esse serviço registra.]",
		choices:           "Suas escolhas",
		choicesText:       "Você pode apagar os dados deste site nas configurações do seu navegador e pedir informações ou a exclusão de dados pelo contato acima, como prevê a Lei Geral de Proteção de Dados (LGPD).",
		choicesBanner:     "Você pode mudar as suas escolhas de cookies a qualquer momento pelo link **Cookies** no rodapé.",
		changes:           "Mudanças nesta política",
		changesText:       "Quando esta política mudar, a data abaixo muda junto.",
	},
	"en": {
		title:             "Privacy policy",
		description:       "How this site handles visitors' data.",
		todo:              "%%\nTemplate made by Blup from the site's tools. Review each part, fill in what's missing and delete what doesn't apply. This comment doesn't show on the site.\nIf you change the tools in blup customize, update this page.\n%%",
		updated:           "Last updated: %s.",
		dateFormat:        "January 2, 2006",
		who:               "Who runs this site",
		whoText:           "This site is run by %s. For anything about privacy, write to %s.",
		contactTodo:       "[your contact email]",
		collect:           "What is collected",
		collectNone:       "This site asks for no sign up and uses no statistics or ads tools.",
		collectIntro:      "This site asks for no sign up. The data below comes from the tools it uses:",
		ga:                "**Google Analytics**: counts visits, page views, time on the site, device type and approximate region. It uses Google cookies and only loads if you accept **Statistics** in the cookie notice. See [Google's privacy policy](https://policies.google.com/privacy).",
		gtm:               "**Google Tag Manager**: loads the measurement and ads tags set up in Google. It only loads if you accept **Statistics** or **Marketing**, and Google's tags follow each choice. See [Google's privacy policy](https://policies.google.com/privacy).",
		umami:             "**Umami**: counts visits and page views in aggregate, with no cookies and without identifying you. It only loads if you accept **Statistics**.",
		plausible:         "**Plausible**: counts visits and page views in aggregate, with no cookies and without identifying you. It only loads if you accept **Statistics**.",
		pixel:             "**Meta Pixel**: measures the results of ads on Facebook and Instagram. It uses Meta cookies and only loads if you accept **Marketing**. See [Meta's privacy policy](https://www.facebook.com/privacy/policy).",
		youtube:           "**YouTube videos**: come from youtube-nocookie.com and only load if you accept **Videos and outside content** or click **Load video**. After that, YouTube may use cookies. See [Google's privacy policy](https://policies.google.com/privacy).",
		comments:          "**Comments and reactions**: to comment or react, you sign in with your GitHub account. The comment, your username and your picture are public on GitHub. See [GitHub's privacy statement](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement).",
		browser:           "What stays in your browser",
		browserText:       "For the site to work, your browser keeps a few things that are never sent to us:\n\n- your choice of light or dark theme;\n- your choice in the cookie notice, for a year;\n- the connection to GitHub, if you signed in to comment, until you sign out;\n- a copy of the site, so it opens offline when installed as an app.",
		hosting:           "Hosting",
		hostingCloudflare: "The site is hosted on Cloudflare Pages, which logs technical access data, such as IP address and browser, to deliver the pages and protect the site from attacks. See [Cloudflare's privacy policy](https://www.cloudflare.com/privacypolicy/).",
		hostingGitHub:     "The site is hosted on GitHub Pages, which logs technical access data, such as IP address, to deliver the pages and protect the site from attacks. See [GitHub's privacy statement](https://docs.github.com/site-policy/privacy-policies/github-general-privacy-statement).",
		hostingOther:      "[Say where the site is hosted and what that service logs.]",
		choices:           "Your choices",
		choicesText:       "You can delete this site's data in your browser settings, and ask for information about your data or for its deletion through the contact above.",
		choicesBanner:     "You can change your cookie choices anytime with the **Cookies** link in the footer.",
		changes:           "Changes to this policy",
		changesText:       "When this policy changes, the date below changes too.",
	},
}

// privacyBody is the starting text of the policy for the site's tools.
func privacyBody(site *config.Site, contact string, now time.Time) string {
	t, ok := privacyTexts[site.Lang]
	if !ok {
		t = privacyTexts["en"]
	}
	c := site.Cookies
	owner := site.Author
	if owner == "" {
		owner = site.Title
	}
	if contact == "" {
		contact = t.contactTodo
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %s\ndescription: %s\nblup: privacy\nhidden: true\ndate: %s\n---\n", t.title, t.description, now.Format("2006-01-02"))
	b.WriteString(t.todo + "\n\n")
	fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.who, fmt.Sprintf(t.whoText, owner, contact))

	var items []string
	for _, it := range []struct {
		on   bool
		text string
	}{
		{c.GoogleAnalytics != "", t.ga},
		{c.TagManager != "", t.gtm},
		{c.Umami != "", t.umami},
		{c.Plausible != "", t.plausible},
		{c.MetaPixel != "", t.pixel},
		{c.Banner, t.youtube},
		{site.Feedback.On(), t.comments},
	} {
		if it.on {
			items = append(items, "- "+it.text)
		}
	}
	fmt.Fprintf(&b, "## %s\n\n", t.collect)
	if len(items) == 0 {
		b.WriteString(t.collectNone + "\n\n")
	} else {
		b.WriteString(t.collectIntro + "\n\n" + strings.Join(items, "\n") + "\n\n")
	}

	fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.browser, t.browserText)
	hosting := t.hostingOther
	switch site.Deploy.Provider {
	case "cloudflare":
		hosting = t.hostingCloudflare
	case "github":
		hosting = t.hostingGitHub
	}
	fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.hosting, hosting)
	choices := t.choicesText
	if c.Banner {
		choices = t.choicesBanner + " " + choices
	}
	fmt.Fprintf(&b, "## %s\n\n%s\n\n", t.choices, choices)
	fmt.Fprintf(&b, "## %s\n\n%s\n\n%s\n", t.changes, t.changesText, fmt.Sprintf(t.updated, now.Format(t.dateFormat)))
	return b.String()
}
