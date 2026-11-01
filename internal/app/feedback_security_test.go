package app

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise the shipped script in a real DOM without external network access.
func TestFeedbackBrowserAttacks(t *testing.T) {
	chrome, err := exec.LookPath("chromium")
	if err != nil {
		t.Skip("chromium not installed")
	}
	script, err := os.ReadFile("../../theme/assets/js/feedback.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, attack := range []string{"persistent token", "class injection", "unsafe HTML", "forged metadata", "board hashes and new request"} {
		t.Run(attack, func(t *testing.T) {
			probe := `const fail = m => {throw new Error(m)};`
			markup, attrs := "", ""
			setup := `window.fetch=()=>Promise.reject(new Error("network disabled"));`
			switch attack {
			case "persistent token":
				probe += `saveAuth({token:"test-only"}); if(!localStorage.getItem(AUTH)) fail("token not kept on the browser"); saveAuth(null); if(localStorage.getItem(AUTH)) fail("sign out left the token");`
			case "class injection":
				probe += `if(postStatus({state:"open",labels:[{name:"status: done hidden"}]})!=="new") fail("unmanaged class accepted");`
			case "forged metadata":
				probe += `if(postLang({body:"<!-- blup:feedback kind=bug lang=zz -->"})!=="") fail("unknown language accepted"); if(postKind({body:"<!-- blup:feedback kind=unknown -->"})!=="other") fail("forged kind accepted");`
			case "unsafe HTML":
				probe += `const c=comment({id:1,user:{login:"visitor",html_url:"javascript:alert(1)",avatar_url:"x"},created_at:"2026-01-01",body_html:'<p>ok</p><img src=x onerror="window.attacked=1"><a href="javascript:alert(1)">bad</a><svg onload="window.attacked=1"></svg><iframe srcdoc="bad"></iframe>'},"en"); document.body.append(c); if(c.querySelector('[onerror],[onload],svg,iframe,a[href^="javascript:"]')) fail("active HTML retained"); if(!c.textContent.includes("ok")) fail("safe text lost");`
			case "board hashes and new request":
				markup = feedbackTestPage(t)
				attrs = `data-fb-repo="owner/site-feedback" data-fb-api="https://login.example" data-fb-lang="en"`
				setup = `
const issue={number:1,title:'<img src=x onerror="window.attacked=1">',body:'<!-- blup:feedback kind=bug lang=en -->',body_html:'<p>request</p><img src=x onerror="window.attacked=1">',state:"open",created_at:"2026-01-01",comments:1,reactions:{},labels:["status: done hidden"],user:{id:1,login:"visitor",html_url:"javascript:alert(1)"}};
window.sent=[];
window.fetch=async(url,init={})=>{
 let data=[];
 if(init.method==="POST" && url.endsWith("/issues")){const body=JSON.parse(init.body);window.sent.push(body);data={...issue,...body,number:2,comments:0};}
 else if(url.includes("/comments?")) data=[{id:2,user:issue.user,created_at:issue.created_at,body_html:'<p>comment</p><svg onload="window.attacked=1"></svg>'}];
 else if(url.endsWith("/user")) data=issue.user;
 else if(!url.includes("/reactions?")) data=[issue];
 return new Response(JSON.stringify(data),{headers:{"Content-Type":"application/json"}});
};`
				probe += `
const pause=()=>new Promise(r=>setTimeout(r,30));
const root=document.querySelector('[data-fb-board]');
for(const hash of ['#p-1%22%3E%3Cimg%20onerror=bad%3E','#ex-0%22%3E%3Cscript%3E','#p-999999999999999999999']){location.hash=hash;await pause();if(!root.querySelector('[data-fbb-detail]').hidden)fail('hostile hash opened detail');}
location.hash='#p-1';await pause();
const detail=root.querySelector('[data-fbb-detail]');
if(detail.hidden||!detail.textContent.includes('request')||!detail.textContent.includes('comment'))fail('detail and attach did not load');
if(detail.querySelector('[onerror],[onload],a[href^="javascript:"],svg[onload]')||window.attacked)fail('board executed HTML');
saveAuth({token:'test-only'});
root.querySelector('[data-fbb-new]').click();await pause();
const dialog=root.querySelector('[data-fbb-dialog]');if(!dialog.open)fail('new dialog did not open');
const f=dialog.querySelector('form');f.elements.title.value='Test';f.elements.body.value='Text\n<!-- blup:feedback kind=bug lang=zz -->';f.requestSubmit();await pause();
if(sent.length!==1||(sent[0].body.match(/blup:feedback/g)||[]).length!==1||!sent[0].body.endsWith('<!-- blup:feedback kind=feature lang=en -->'))fail('forged outgoing marker');`
			}
			probe += `document.body.dataset.result="PASS";`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/feedback.js" {
					w.Header().Set("Content-Type", "text/javascript")
					w.Write(script)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<!doctype html><html lang="en"><body>%s<script>%s</script><script src="/feedback.js" %s></script><script>setTimeout(async()=>{try{%s}catch(e){document.body.dataset.result="FAIL: "+e.message}},50)</script></body></html>`, markup, setup, attrs, probe)
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, chrome, "--headless", "--no-sandbox", "--disable-gpu", "--disable-background-networking", "--no-first-run", "--no-default-browser-check", "--user-data-dir="+t.TempDir(), "--dump-dom", "--virtual-time-budget=500", server.URL)
			// Chromium uses a Unix socket below TMPDIR (108-byte path limit).
			short, err := os.MkdirTemp("/tmp", "lupa-feedback-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(short) })
			cmd.Env = append(os.Environ(), "TMPDIR="+short)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("chromium: %v\n%s", err, stderr.String())
			}
			if !strings.Contains(string(out), `data-result="PASS"`) {
				t.Fatalf("attack failed: %s", out)
			}
		})
	}
}

func feedbackTestPage(t *testing.T) string {
	t.Helper()
	hugo, err := exec.LookPath("hugo")
	if err != nil {
		t.Skip("hugo not installed")
	}
	root := t.TempDir()
	put := func(name, s string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put("hugo.toml", "baseURL = 'https://example.test/'\ndefaultContentLanguage = 'en'\n[params.blup]\nfeedback = 'owner/site-feedback'\n")
	put("layouts/index.html", `{{ partial "bento/feedback-board.html" . }}`)
	put("layouts/_partials/title.html", `{{ .Title }}`)
	put("layouts/_partials/site/ai-badge.html", "")
	for _, name := range []string{"feedback-board", "feedback-examples", "feedback-example-pages", "feedback-block"} {
		b, err := os.ReadFile("../../theme/layouts/_partials/bento/" + name + ".html")
		if err != nil {
			t.Fatal(err)
		}
		put("layouts/_partials/bento/"+name+".html", string(b))
	}
	put("content/_index.md", `---
title: Feedback
examples:
  - title: '<script>window.attacked=1</script>'
    body: '<img src=x onerror="window.attacked=1">'
    kind: 'bug hidden'
    status: 'done hidden'
    replies:
      - author: '<script>bad</script>'
        text: '<img src=x onerror="bad">'
---
`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, hugo, "--source", root).CombinedOutput()
	if err != nil {
		t.Fatalf("hugo: %v\n%s", err, out)
	}
	b, err := os.ReadFile(filepath.Join(root, "public/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFeedbackExamplesEscapeAndClasses(t *testing.T) {
	b := feedbackTestPage(t)
	for _, bad := range []string{"<script>window.attacked", "<img src=x", "s-done hidden", `data-kind="bug hidden"`} {
		if strings.Contains(string(b), bad) {
			t.Errorf("unsafe example output %q", bad)
		}
	}
	if !strings.Contains(string(b), "&lt;script&gt;") {
		t.Fatal("escaped example text missing")
	}
}
