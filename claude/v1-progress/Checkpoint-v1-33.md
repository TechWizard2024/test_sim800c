Voici les problèmes identifiés et les corrections à apporter :

**Résumé des bugs :**

1.  **Thème clair** — 
    ```
    main.css
    ```
     n'a pas de variables CSS pour le thème clair → contenu invisible
2.  **WebSocket** — URL 
    ```
    ws://.../ws
    ```
     au lieu de 
    ```
    ws://.../api/ws
    ```
    
3.  **Timestamps** — événements WebSocket de reconnexion sans horodatage
4.  **Dashboard** — onglet masqué au départ (
    ```
    .tab-content.active
    ```
     ignoré car 
    ```
    showTab
    ```
     initialise tout à 
    ```
    none
    ```
    )
5.  **Statut système** — utilise 
    ```
    localStorage.getItem('auth_token')
    ```
     mais le token est stocké sous 
    ```
    token
    ```
    
6.  **USSD output** (Fonction 4) — 
    ```
    <pre id="ussd-output" class="result">
    ```
     avec 
    ```
    .result { display: none }
    ```
     → jamais visible
7.  **Auto-Discovery erreur** — 
    ```
    apiCall
    ```
     non utilisé dans 
    ```
    autoStatusDiscovery
    ```
     (pas de Bearer token)
8.  **Doublons boutons** — 
    ```
    GetConsultCodes
    ```
     retourne Universel + carrier spécifique → duplications
9.  **COM7 MTN** — si le carrier retourné est "MTN" au lieu de "MTN CI" → aucun bouton
10.  **SMS envoi** — validation frontend exige 10 chiffres exactement mais numéros CI peuvent avoir indicatif
11.  **Cards reposition** — 
    ```
    .result { display: none }
    ```
     puis 
    ```
    display:block
    ```
     fait sauter la grille
12.  **Module COM non-connecté** — 
    ```
    GetAllModules()
    ```
     renvoie tous modules même déconnectés

Commençons les corrections :

Exécuté 3 commandes

Exécuté 3 commandes

Copy project to v1-34

Script

Create light theme CSS

bash

```bash
cat> /home/claude/v1-34/web/css/theme-light.css <<'EOF'
/* ─── Thème clair — variables CSS ─────────────────────────────────────── */
:root {
    --bg-primary: #f0f2f5;
    --text-primary: #1a1a2e;
    --text-secondary: #555577;
    --card-bg: #ffffff;
    --border-color: #d0d4dd;
    --hover-bg: rgba(102,126,234,0.08);
    --primary-color: #667eea;
    --secondary-color: #764ba2;
    --info-bg: #f0f4ff;
    --result-bg: #f4f7ff;
    --input-bg: #ffffff;
    --text: #1a1a2e;
    --subtext: #666699;
    --border: #d0d4dd;
    --heading: #333366;
    --success: #28a745;
    --warning: #ffc107;
    --danger: #dc3545;
    --info: #17a2b8;
    --surface: #ffffff;
    --text-muted: #888899;
    --unread-bg: #e8f0ff;
    --trash-bg: #fff5f5;
    --primary: #667eea;
    --bg: #ffffff;
}
body {
    background-color: var(--bg-primary);
    color: var(--text-primary);
}
EOFecho"Done"
```

Sortie

```
Done

```

Update dark theme with all CSS vars

bash

```bash
cat> /tmp/theme_dark_patch.css <<'EOF'
/* ─── Thème sombre — variables CSS ─────────────────────────────────────── */
:root {
    --bg-primary: #1a1a2e;
    --text-primary: #f0f0f0;
    --text-secondary: #b0b0b0;
    --card-bg: #16213e;
    --border-color: #0f3460;
    --hover-bg: #1f2a4a;
    --primary-color: #e94560;
    --secondary-color: #533483;
    --info-bg: #1a2a4a;
    --result-bg: #0f1a2e;
    --input-bg: #0f1a2e;
    --text: #f0f0f0;
    --subtext: #b0b0c0;
    --border: #0f3460;
    --heading: #e0e0f0;
    --success: #4caf50;
    --warning: #ffc107;
    --danger: #f44336;
    --info: #29b6f6;
    --surface: #16213e;
    --text-muted: #8080a0;
    --unread-bg: #1a2a4a;
    --trash-bg: #2a1a1a;
    --primary: #e94560;
    --bg: #16213e;
}
body {
    background-color: var(--bg-primary);
    color: var(--text-primary);
}
EOFcp /tmp/theme_dark_patch.css /home/claude/v1-34/web/css/theme-dark.css
echo"Done"
```

Sortie

```
Done

```

Terminé

Now let's fix the main 
```
index.html
```
 — multiple fixes needed:

Patch index.html with all fixes

Patch index.html with all fixes

Patch index.html with all fixes

bash

```bash
cd /home/claude/v1-34 && python3 <<'PYEOF'
with open('web/index.html', 'r', encoding='utf-8') as f:
    html = f.read()
# Fix 1: Add theme-light CSS link + keep theme-dark
old_links = '''    <link rel="stylesheet" id="theme-light" href="/css/main.css">
    <link rel="stylesheet" id="theme-dark" href="/css/theme-dark.css" disabled>'''
new_links = '''    <link rel="stylesheet" href="/css/main.css">
    <link rel="stylesheet" id="theme-light" href="/css/theme-light.css">
    <link rel="stylesheet" id="theme-dark" href="/css/theme-dark.css" disabled>'''
html = html.replace(old_links, new_links)
# Fix 2: WebSocket URL - fix ws path /ws -> /api/ws
old_ws = 'const wsUrl = `ws://${window.location.host}/ws`;'
new_ws = '''const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${wsProtocol}//${window.location.host}/api/ws`;'''
html = html.replace(old_ws, new_ws)
# Fix 3: addRTEvent - add timestamp when missing
old_addRT = '''        function addRTEvent(text) {
            const log = document.getElementById('rt-log');
            const div = document.createElement('div');
            div.className = 'rt-event';
            div.textContent = text;'''
new_addRT = '''        function addRTEvent(text) {
            const log = document.getElementById('rt-log');
            const div = document.createElement('div');
            div.className = 'rt-event';
            // Add timestamp if not already present
            const ts = new Date().toLocaleTimeString('fr-FR');
            const hasTs = /\\[\\d{2}:\\d{2}:\\d{2}\\]/.test(text);
            div.textContent = hasTs ? text : `[${ts}] ${text}`;'''
html = html.replace(old_addRT, new_addRT)
# Fix 4: Dashboard tab - show by default (remove display:none from tab-content CSS is done in CSS)
# Actually the first tab should be visible - the CSS .tab-content.active handles it
# The dashboard tab has class "tab-content" but NOT "active" initially - we need to add it
html = html.replace(
    '<div id="dashboard-tab" class="tab-content">',
    '<div id="dashboard-tab" class="tab-content active">'
)
# Fix 5: System status - auth token key fix (localStorage 'auth_token' -> 'token')
# Already correct in loadSystemStatus in app.js, but index.html also uses it
# The index.html loadSystemStatus is in app.js, let's check the loadSystemStatus call in index.html
# Actually app.js uses localStorage.getItem('auth_token') - but token is stored as 'token'
# This is fixed in app.js below
# Fix 6: USSD output - remove display:none (class="result" has display:none)
# Change the pre#ussd-output to not use class="result"
html = html.replace(
    '<pre id="ussd-output" class="result" style="margin-top:10px; white-space: pre-wrap; word-break: break-word;"></pre>',
    '<div id="ussd-output" style="margin-top:10px; padding:10px; background:var(--result-bg,#f0f4ff); border-radius:8px; white-space:pre-wrap; word-break:break-word; font-size:0.85rem; min-height:40px; color:var(--text,#333); display:none;"></div>'
)
# Fix 7: auto-discovery-result and auto-menu-result - also remove class="result"
html = html.replace(
    '<div id="auto-discovery-result" class="result" style="margin-top:10px;"></div>',
    '<div id="auto-discovery-result" style="margin-top:10px; padding:10px; background:var(--result-bg,#f0f4ff); border-radius:8px; display:none;"></div>'
)
html = html.replace(
    '<div id="auto-menu-result" class="result" style="margin-top:10px;"></div>',
    '<div id="auto-menu-result" style="margin-top:10px; padding:10px; background:var(--result-bg,#f0f4ff); border-radius:8px; display:none;"></div>'
)
# Fix 8: explorer-result - remove class="result"
html = html.replace(
    '<div id="explorer-result" class="result"></div>',
    '<div id="explorer-result" style="margin-top:10px; padding:10px; background:var(--result-bg,#f0f4ff); border-radius:8px; display:none; white-space:pre-wrap; font-size:0.85rem;"></div>'
)
# Fix 9: SMS number validation - allow 8-15 digits (local or with prefix)
old_sms_validate = "if (!/^\\\\d{10}$/.test(number)) { alert('Le numéro doit contenir exactement 10 chiffres (sans indicatif)'); return; }"
new_sms_validate = "if (!/^[+\\\\d]{8,15}$/.test(number)) { alert('Numéro invalide (8-15 chiffres, indicatif international autorisé)'); return; }"
html = html.replace(old_sms_validate, new_sms_validate)
# Fix 10: executeUSSD - show ussd-output when result arrives
old_exec_ussd = '''        async function executeUSSD(moduleId, code, inputData = '') {
            const resultDiv = document.getElementById(`result-${moduleId}`);
            if (resultDiv) {
                resultDiv.style.display = 'block';
                resultDiv.innerHTML = '<pre>⏳ Exécution de ' + escapeHtml(code) + '...</pre>';
            }
            const response = await apiCall(`/api/modules/${moduleId}/ussd/execute`, {
                method: 'POST',
                body: JSON.stringify({ ussd_code: code, input_data: inputData })
            });
            const result = await response.json();
            if (resultDiv) resultDiv.innerHTML = '<pre>' + escapeHtml(result.result || result.error || 'Aucun résultat') + '</pre>';
            showToast(result.success ? 'Succès' : 'Erreur', result.success ? 'success' : 'error');
        }'''
new_exec_ussd = '''        async function executeUSSD(moduleId, code, inputData = '') {
            const resultDiv = document.getElementById(`result-${moduleId}`);
            if (resultDiv) {
                resultDiv.style.display = 'block';
                resultDiv.innerHTML = '<pre>⏳ Exécution de ' + escapeHtml(code) + '...</pre>';
            }
            try {
                const response = await apiCall(`/api/modules/${moduleId}/ussd/execute`, {
                    method: 'POST',
                    body: JSON.stringify({ ussd_code: code, input_data: inputData })
                });
                const result = await response.json();
                const parsed = parseUSSDResult(result.result || result.error || 'Aucun résultat');
                if (resultDiv) resultDiv.innerHTML = '<pre>' + escapeHtml(parsed) + '</pre>';
                showToast(result.success ? 'Succès' : 'Erreur', result.success ? 'success' : 'error');
            } catch(e) {
                if (resultDiv) resultDiv.innerHTML = '<pre>Erreur: ' + escapeHtml(e.message) + '</pre>';
                showToast('Erreur', 'error');
            }
        }'''
html = html.replace(old_exec_ussd, new_exec_ussd)
# Fix 11: executeUSSDManual - show ussd-output div
old_exec_manual = '''            if (outputPre) {
                outputPre.textContent = result.result || result.error || JSON.stringify(result, null, 2);
            }'''
new_exec_manual = '''            if (outputPre) {
                outputPre.textContent = parseUSSDResult(result.result || result.error || JSON.stringify(result, null, 2));
                outputPre.style.display = 'block';
            }'''
html = html.replace(old_exec_manual, new_exec_manual)
# Fix 12: executeUSSDInSection - parse result
old_exec_section = "if (resultDiv) resultDiv.innerHTML = '<pre>' + escapeHtml(result.result || result.error || 'Aucun résultat') + '</pre>';"
new_exec_section = "if (resultDiv) resultDiv.innerHTML = '<pre>' + escapeHtml(parseUSSDResult(result.result || result.error || 'Aucun résultat')) + '</pre>';"
html = html.replace(old_exec_section, new_exec_section, 1)
# Fix 13: Add parseUSSDResult helper function before escapeHtml
old_escape = "        function escapeHtml(text) {"
new_escape = '''        // Parse AT +CUSD response to extract human-readable text
        function parseUSSDResult(raw) {
            if (!raw) return 'Aucun résultat';
            // Extract text from +CUSD: N, "message" pattern
            const cusdMatch = raw.match(/\\+CUSD:\\s*\\d+,\\s*"([^"]+)"/);
            if (cusdMatch) return cusdMatch[1].replace(/\\\\n/g, '\\n').trim();
            // For *#06# IMEI: extract from +CGSN or plain number
            const imeiMatch = raw.match(/\\+CGSN[:\\s]+(\\d{15})/);
            if (imeiMatch) return 'IMEI: ' + imeiMatch[1];
            // Already clean text
            if (!raw.includes('+CUSD') && !raw.includes('AT+CUSD')) return raw.trim();
            return raw;
        }
        function escapeHtml(text) {'''
html = html.replace(old_escape, new_escape)
# Fix 14: Add sent SMS section in SMS tab after the received messages card
old_sms_list_end = '''            <div class="module-card" style="margin-top: 20px;">
                <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;flex-wrap:wrap;gap:8px;">
                    <h3 style="margin:0;">📥 Messages reçus'''
new_sms_section = '''            <!-- SMS Envoyés -->
            <div class="module-card" style="margin-top: 20px;">
                <h3 style="margin:0 0 10px 0;">📤 SMS envoyés</h3>
                <div id="sms-sent-list" style="max-height:300px;overflow-y:auto;">
                    <div class="loading">Chargement...</div>
                </div>
            </div>
            <div class="module-card" style="margin-top: 20px;">
                <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;flex-wrap:wrap;gap:8px;">
                    <h3 style="margin:0;">📥 Messages reçus'''
html = html.replace(old_sms_list_end, new_sms_section)
with open('web/index.html', 'w', encoding='utf-8') as f:
    f.write(html)
print("index.html patched successfully")
# Verify key changes
checks = [
    ('theme-light.css', 'theme-light.css' in html),
    ('api/ws WebSocket', '/api/ws' in html),
    ('dashboard-tab active', 'dashboard-tab" class="tab-content active"' in html),
    ('ussd-output div', 'id="ussd-output"' in html and 'class="result"' not in html.split('id="ussd-output"')[1][:50]),
    ('parseUSSDResult', 'parseUSSDResult' in html),
    ('sms-sent-list', 'sms-sent-list' in html),
]
for name, ok in checks:
    print(f"  {'✅' if ok else '❌'} {name}")
PYEOF
```

Sortie

```
index.html patched successfully
  ✅ theme-light.css
  ✅ api/ws WebSocket
  ✅ dashboard-tab active
  ✅ ussd-output div
  ✅ parseUSSDResult
  ✅ sms-sent-list

```

Terminé