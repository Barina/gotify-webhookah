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
		Version:     "1.0.0",
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
