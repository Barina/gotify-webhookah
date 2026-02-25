package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gotify/plugin-api"
)

func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		Name:        "Webhookah",
		Description: "Build and copy Gotify webhook curl commands for your apps.",
		Version:     "1.0.1",
		Author:      "Roy Barina",
		Website:     "https://github.com/barina/gotify-webhookah",
		License:     "MIT",
		ModulePath:  "github.com/barina/gotify-webhookah",
	}
}

type Config struct {
	PublicDomain string `json:"public_domain" yaml:"public_domain"`
	LocalIP      string `json:"local_ip" yaml:"local_ip"`
	LocalPort    string `json:"local_port" yaml:"local_port"`
}

type Plugin struct {
	userCtx        plugin.UserContext
	storageHandler plugin.StorageHandler
	config         Config
	enabled        bool
	basePath       string
}

func (p *Plugin) Enable() error {
	p.loadConfig()
	p.enabled = true
	return nil
}

func (p *Plugin) Disable() error {
	p.enabled = false
	return nil
}

func (p *Plugin) SetStorageHandler(h plugin.StorageHandler) {
	p.storageHandler = h
	p.loadConfig()
}

func (p *Plugin) DefaultConfig() interface{} {
	return &Config{
		PublicDomain: "",
		LocalIP:      "",
		LocalPort:    "80",
	}
}

func (p *Plugin) ValidateAndSetConfig(config interface{}) error {
	p.config = *config.(*Config)
	return p.saveConfig()
}

func (p *Plugin) SetBaseURL(baseURL *url.URL) {}

func (p *Plugin) RegisterWebhook(basePath string, mux *gin.RouterGroup) {
	p.basePath = basePath
	mux.GET("/webhookah", p.serveBuilder)
	mux.GET("/apps", p.serveApps)
}

func (p *Plugin) GetDisplay(location *url.URL) string {
	if location == nil || p.basePath == "" {
		return "Plugin initializing..."
	}
	base := fmt.Sprintf("%s://%s", location.Scheme, location.Host)
	path := strings.TrimRight(p.basePath, "/")
	return fmt.Sprintf("### Webhookah\n\n[Open Webhook Builder](%s%s/webhookah)", base, path)
}

func (p *Plugin) serveApps(c *gin.Context) {
	if !p.enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "plugin not enabled"})
		return
	}

	req, err := http.NewRequest("GET", "http://localhost/application", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build request"})
		return
	}

	for _, h := range []string{"X-Gotify-Key", "Authorization", "Cookie"} {
		if v := c.GetHeader(h); v != "" {
			req.Header.Set(h, v)
		}
	}
	if t := c.Query("token"); t != "" {
		q := req.URL.Query()
		q.Set("token", t)
		req.URL.RawQuery = q.Encode()
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach Gotify API"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

func (p *Plugin) serveBuilder(c *gin.Context) {
	if !p.enabled {
		c.String(http.StatusServiceUnavailable, "Plugin not enabled")
		return
	}

	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := c.Request.Host

	publicBase := fmt.Sprintf("%s://%s", scheme, host)
	if p.config.PublicDomain != "" {
		publicBase = fmt.Sprintf("%s://%s", scheme, strings.TrimRight(p.config.PublicDomain, "/"))
	}

	localBase := ""
	if p.config.LocalIP != "" {
		localPort := p.config.LocalPort
		if localPort == "" {
			localPort = "80"
		}
		localBase = fmt.Sprintf("http://%s:%s", p.config.LocalIP, localPort)
	}

	localBaseJS := "null"
	if localBase != "" {
		localBaseJS = fmt.Sprintf("%q", localBase)
	}

	appsEndpoint := strings.TrimRight(p.basePath, "/") + "/apps"

	var v = GetGotifyPluginInfo().Version

	html := fmt.Sprintf(`<!DOCTYPE html>
  <html lang="en">
  <head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Webhookah</title>
  <link rel="icon" type="image/png" href="data:image/png;base64,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">
  <style>
    @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600;700&family=Syne:wght@400;700;800&display=swap');
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    :root {
      --bg: #0f0f13; --surface: #16161d; --surface2: #1e1e28; --border: #2a2a38;
      --accent: #7c6af7; --accent2: #f76a8c; --text: #e8e8f0; --muted: #6b6b80;
      --success: #4ade80; --warn: #fbbf24;
      --mono: 'JetBrains Mono', monospace; --sans: 'Syne', sans-serif;
    }
    body { background: var(--bg); color: var(--text); font-family: var(--sans); min-height: 100vh; padding: 2rem 1rem; display: flex; flex-direction: column; align-items: center; }
    .container { width: 100%%; max-width: 700px; }
    header { margin-bottom: 2.5rem; display: flex; align-items: baseline; gap: 0.75rem; }
    h1 { font-size: 2rem; font-weight: 800; background: linear-gradient(135deg, var(--accent), var(--accent2)); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; letter-spacing: -0.03em; }
    .version { font-family: var(--mono); font-size: 0.7rem; color: var(--muted); border: 1px solid var(--border); padding: 2px 6px; border-radius: 4px; }
    .card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; margin-bottom: 1rem; }
    .card-title { font-size: 0.65rem; font-family: var(--mono); color: var(--muted); text-transform: uppercase; letter-spacing: 0.1em; margin-bottom: 1rem; }
    .field { margin-bottom: 1rem; }
    .field:last-child { margin-bottom: 0; }
    label { display: block; font-size: 0.75rem; font-family: var(--mono); color: var(--muted); margin-bottom: 0.4rem; text-transform: uppercase; letter-spacing: 0.08em; white-space: nowrap; }
    label .optional { color: var(--border); font-size: 0.65rem; margin-left: 0.4rem; }
    input, select, textarea { width: 100%%; background: var(--surface2); border: 1px solid var(--border); border-radius: 8px; padding: 0.65rem 0.85rem; color: var(--text); font-family: var(--mono); font-size: 0.875rem; outline: none; transition: border-color 0.15s; appearance: none; }
    textarea { resize: vertical; min-height: 80px; line-height: 1.6; }
    input:focus, select:focus, textarea:focus { border-color: var(--accent); }
    input::placeholder, textarea::placeholder { color: var(--muted); }
    .row-3 { display: grid; grid-template-columns: 1fr 1fr 90px; gap: 1rem; align-items: start; }
    .status-bar { display: flex; align-items: center; gap: 0.5rem; font-family: var(--mono); font-size: 0.75rem; color: var(--muted); margin-bottom: 1rem; padding: 0.5rem 0.75rem; background: var(--surface2); border-radius: 6px; border: 1px solid var(--border); }
    .dot { width: 6px; height: 6px; border-radius: 50%%; background: var(--muted); flex-shrink: 0; transition: all 0.3s; }
    .dot.loading { background: var(--warn); box-shadow: 0 0 6px var(--warn); animation: pulse 1s infinite; }
    .dot.ok { background: var(--success); box-shadow: 0 0 6px var(--success); }
    .dot.err { background: var(--accent2); box-shadow: 0 0 6px var(--accent2); }
    @keyframes pulse { 0%%, 100%% { opacity: 1; } 50%% { opacity: 0.3; } }
    .cmd-block { background: var(--surface2); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; margin-bottom: 0.75rem; }
    .cmd-label { font-family: var(--mono); font-size: 0.65rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.1em; padding: 0.5rem 0.85rem 0.3rem; border-bottom: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; }
    .cmd-text { font-family: var(--mono); font-size: 0.78rem; color: #a78bfa; padding: 0.75rem 0.85rem; word-break: break-all; line-height: 1.6; min-height: 2.5rem; white-space: pre-wrap; }
    .cmd-text.empty { color: var(--muted); font-style: italic; font-family: var(--sans); font-size: 0.8rem; }
    .btn-row { display: flex; gap: 0.4rem; }
    .action-btn { font-family: var(--mono); font-size: 0.65rem; color: var(--muted); background: none; border: 1px solid var(--border); border-radius: 4px; padding: 3px 10px; cursor: pointer; transition: all 0.15s; text-transform: uppercase; letter-spacing: 0.05em; white-space: nowrap; }
    .action-btn:hover { color: var(--accent); border-color: var(--accent); }
    .action-btn.copied { color: var(--success); border-color: var(--success); }
    .action-btn.testing { color: var(--warn); border-color: var(--warn); }
    .action-btn.sent { color: var(--success); border-color: var(--success); }
    .action-btn.failed { color: var(--accent2); border-color: var(--accent2); }
    .note { font-family: var(--mono); font-size: 0.72rem; color: var(--muted); line-height: 1.7; padding: 0.85rem; background: var(--surface2); border-left: 3px solid var(--accent); border-radius: 0 6px 6px 0; margin-top: 1rem; }
    .note strong { color: var(--accent2); }
    .note code { color: #a78bfa; background: rgba(124,106,247,0.1); padding: 1px 5px; border-radius: 3px; }
    .toggle-row { display: flex; align-items: center; gap: 0.6rem; margin-top: 1rem; }
    .toggle { position: relative; width: 36px; height: 20px; flex-shrink: 0; }
    .toggle input { opacity: 0; width: 0; height: 0; }
    .slider { position: absolute; inset: 0; background: var(--surface2); border: 1px solid var(--border); border-radius: 20px; cursor: pointer; transition: 0.2s; }
    .slider:before { content: ''; position: absolute; width: 14px; height: 14px; left: 2px; top: 2px; background: var(--muted); border-radius: 50%%; transition: 0.2s; }
    .toggle input:checked + .slider { border-color: var(--accent); }
    .toggle input:checked + .slider:before { background: var(--accent); transform: translateX(16px); }
    .toggle-label { font-family: var(--mono); font-size: 0.72rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.08em; }
  </style>
  </head>
  <body>
  <div class="container">
    <header><h1>Webhookah</h1><span class="version">v%s</span></header>

    <div class="card">
      <div class="card-title">App Selection</div>
      <div class="status-bar">
        <div class="dot loading" id="statusDot"></div>
        <span id="statusText">Loading apps...</span>
      </div>
      <div class="field">
        <label>Application</label>
        <select id="app" onchange="save();updateCommands()">
          <option value="">Loading...</option>
        </select>
      </div>
    </div>

    <div class="card">
      <div class="card-title">Message Parameters</div>
      <div class="field">
        <label>Message <span class="optional">(required)</span></label>
        <textarea id="message" placeholder="e.g. Build #42 failed on main" rows="3" oninput="save();updateCommands()"></textarea>
      </div>
      <div class="toggle-row" style="margin-top:0;margin-bottom:1rem">
        <label class="toggle">
          <input type="checkbox" id="markdownToggle" onchange="save();updateCommands()">
          <span class="slider"></span>
        </label>
        <span class="toggle-label">Markdown <span class="optional">(sends as text/markdown)</span></span>
      </div>
      <div class="row-3">
        <div class="field">
          <label>Title <span class="optional">(optional)</span></label>
          <input id="title" placeholder="e.g. Build Failed" type="text" oninput="save();updateCommands()">
        </div>
        <div class="field">
          <label>Domain override <span class="optional">(optional)</span></label>
          <input id="domainOverride" placeholder="e.g. gotify.example.com" type="text" oninput="save();updateCommands()">
        </div>
        <div class="field">
          <label>Priority <span class="optional">(0-10)</span></label>
          <input id="priority" type="number" min="0" max="10" placeholder="5" oninput="save();updateCommands()">
        </div>
      </div>
    </div>

    <div class="card">
      <div class="card-title">Generated Commands</div>
      <div class="cmd-block">
        <div class="cmd-label">
          <span>Webhook</span>
          <div class="btn-row">
            <button class="action-btn" id="copyPublic" onclick="copyCmd('publicCmd','copyPublic')">Copy</button>
            <button class="action-btn" id="testPublic" onclick="testCmd('public')">Test</button>
          </div>
        </div>
        <div class="cmd-text empty" id="publicCmd">Select an app and enter a message to generate</div>
      </div>
      <div id="localBlock" style="display:none">
        <div class="cmd-block" style="margin-top:0.75rem">
          <div class="cmd-label">
            <span>Local</span>
            <div class="btn-row">
              <button class="action-btn" id="copyLocal" onclick="copyCmd('localCmd','copyLocal')">Copy</button>
              <button class="action-btn" id="testLocal" onclick="testCmd('local')">Test</button>
            </div>
          </div>
          <div class="cmd-text empty" id="localCmd">Select an app and enter a message to generate</div>
        </div>
      </div>
      <div class="note">
        <strong>Note:</strong> These are <code>curl</code> commands for scripts, CI/CD, or terminals.
        Gotify requires a <strong>POST</strong> request — clicking a URL in a browser sends GET and will fail.
        Use the <strong>Test</strong> button to fire a real message instantly.
        When markdown is enabled, the command sends a <strong>JSON body</strong> with the extras header.
      </div>
    </div>
  </div>

  <script>
  const serverPublicBase = %q;
  const localBase = %s;
  const appsEndpoint = %q;

  const STORAGE_KEY = 'webhookah-state';

  // Current state for test button
  let currentPublicPayload = null;
  let currentLocalPayload = null;

  function save() {
    const state = {
      appToken: document.getElementById('app').value,
      message: document.getElementById('message').value,
      title: document.getElementById('title').value,
      priority: document.getElementById('priority').value,
      domainOverride: document.getElementById('domainOverride').value,
      markdown: document.getElementById('markdownToggle').checked,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  }

  function loadSaved() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch(e) { return null; }
  }

  function restoreFields(state) {
    if (!state) return;
    if (state.message) document.getElementById('message').value = state.message;
    if (state.title) document.getElementById('title').value = state.title;
    if (state.priority !== undefined && state.priority !== '') document.getElementById('priority').value = state.priority;
    if (state.domainOverride) document.getElementById('domainOverride').value = state.domainOverride;
    if (state.markdown) document.getElementById('markdownToggle').checked = state.markdown;
  }

  function restoreAppSelection(state) {
    if (!state || !state.appToken) return;
    const sel = document.getElementById('app');
    for (let i = 0; i < sel.options.length; i++) {
      if (sel.options[i].value === state.appToken) { sel.selectedIndex = i; break; }
    }
  }

  async function loadApps() {
    const dot = document.getElementById('statusDot');
    const txt = document.getElementById('statusText');
    const token = localStorage.getItem('gotify-login-key');
    if (!token) {
      dot.className = 'dot err';
      txt.textContent = 'Not logged in to Gotify — please log in first';
      document.getElementById('app').innerHTML = '<option value="">Not authenticated</option>';
      return;
    }
    try {
      const resp = await fetch(appsEndpoint, { headers: { 'X-Gotify-Key': token } });
      if (!resp.ok) throw new Error('HTTP ' + resp.status);
      const apps = await resp.json();
      const sel = document.getElementById('app');
      if (!apps || apps.length === 0) {
        sel.innerHTML = '<option value="">No apps found</option>';
        dot.className = 'dot err';
        txt.textContent = 'No applications found in Gotify';
        return;
      }
      sel.innerHTML = '<option value="">Select an app...</option>';
      apps.forEach(app => {
        const opt = document.createElement('option');
        opt.value = app.token;
        opt.text = app.name + (app.description ? ' — ' + app.description : '');
        sel.add(opt);
      });
      dot.className = 'dot ok';
      txt.textContent = apps.length + ' app' + (apps.length !== 1 ? 's' : '') + ' loaded';
      const saved = loadSaved();
      restoreFields(saved);
      restoreAppSelection(saved);
      updateCommands();
    } catch(e) {
      dot.className = 'dot err';
      txt.textContent = 'Failed to load apps: ' + e.message;
      document.getElementById('app').innerHTML = '<option value="">Failed to load</option>';
    }
  }

  function getEffectivePublicBase() {
    const override = document.getElementById('domainOverride').value.trim();
    if (override) {
      const scheme = serverPublicBase.startsWith('https') ? 'https' : 'http';
      return scheme + '://' + override.replace(/^https?:\/\//, '').replace(/\/$/, '');
    }
    return serverPublicBase;
  }

  // Returns {url, body, isJSON} for a given base
  function buildPayload(base, token) {
    const msg = document.getElementById('message').value.trim();
    if (!token || !msg) return null;

    const isMarkdown = document.getElementById('markdownToggle').checked;
    const title = document.getElementById('title').value.trim();
    const prio = document.getElementById('priority').value.trim();
    const url = base + '/message?token=' + encodeURIComponent(token);

    if (isMarkdown) {
      // JSON body required for extras
      const body = { message: msg };
      if (title) body.title = title;
      if (prio !== '') body.priority = parseInt(prio, 10);
      body.extras = { 'client::display': { contentType: 'text/markdown' } };
      return { url, body: JSON.stringify(body), isJSON: true };
    } else {
      // Simple query params
      const params = new URLSearchParams();
      params.set('message', msg);
      if (title) params.set('title', title);
      if (prio !== '') params.set('priority', prio);
      return { url: url + '&' + params.toString(), body: null, isJSON: false };
    }
  }

  function buildCurlCmd(payload) {
    if (!payload) return null;
    if (payload.isJSON) {
      // Pretty JSON for readability, escaped for shell
      const escaped = payload.body.replace(/'/g, "'\\''");
      return "curl -X POST '" + payload.url + "' \\\n  -H 'Content-Type: application/json' \\\n  -d '" + escaped + "'";
    }
    return "curl -X POST '" + payload.url + "'";
  }

  function updateCommands() {
    const token = document.getElementById('app').value;
    const pubCmd = document.getElementById('publicCmd');

    currentPublicPayload = buildPayload(getEffectivePublicBase(), token);
    currentLocalPayload = localBase ? buildPayload(localBase, token) : null;

    if (!currentPublicPayload) {
      pubCmd.textContent = 'Select an app and enter a message to generate';
      pubCmd.className = 'cmd-text empty';
      return;
    }

    pubCmd.textContent = buildCurlCmd(currentPublicPayload);
    pubCmd.className = 'cmd-text';

    if (localBase) {
      document.getElementById('localBlock').style.display = 'block';
      const locCmd = document.getElementById('localCmd');
      locCmd.textContent = currentLocalPayload ? buildCurlCmd(currentLocalPayload) : '';
      locCmd.className = 'cmd-text';
    }
  }

  function copyCmd(id, btnId) {
    const el = document.getElementById(id);
    if (el.classList.contains('empty')) return;
    navigator.clipboard.writeText(el.textContent).then(() => {
      const btn = document.getElementById(btnId);
      btn.textContent = 'Copied!';
      btn.className = 'action-btn copied';
      setTimeout(() => { btn.textContent = 'Copy'; btn.className = 'action-btn'; }, 2000);
    });
  }

  async function testCmd(which) {
    const payload = which === 'public' ? currentPublicPayload : currentLocalPayload;
    const btnId = which === 'public' ? 'testPublic' : 'testLocal';
    if (!payload) return;

    const btn = document.getElementById(btnId);
    btn.textContent = 'Sending...';
    btn.className = 'action-btn testing';

    try {
      const fetchOpts = { method: 'POST' };
      if (payload.isJSON) {
        fetchOpts.headers = { 'Content-Type': 'application/json' };
        fetchOpts.body = payload.body;
      }
      const resp = await fetch(payload.url, fetchOpts);
      if (resp.ok) {
        btn.textContent = 'Sent!';
        btn.className = 'action-btn sent';
      } else {
        btn.textContent = 'Failed ' + resp.status;
        btn.className = 'action-btn failed';
      }
    } catch(e) {
      btn.textContent = 'Error';
      btn.className = 'action-btn failed';
    }
    setTimeout(() => { btn.textContent = 'Test'; btn.className = 'action-btn'; }, 3000);
  }

  loadApps();
  </script>
  </body>
  </html>`, v, publicBase, localBaseJS, appsEndpoint)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func (p *Plugin) loadConfig() {
	if p.storageHandler == nil {
		return
	}
	data, err := p.storageHandler.Load()
	if err != nil || data == nil {
		return
	}
	json.Unmarshal(data, &p.config)
}

func (p *Plugin) saveConfig() error {
	if p.storageHandler == nil {
		return nil
	}
	data, err := json.Marshal(p.config)
	if err != nil {
		return err
	}
	return p.storageHandler.Save(data)
}

func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	return &Plugin{userCtx: ctx}
}

func main() {
	panic("this should be built as a Go plugin")
}
