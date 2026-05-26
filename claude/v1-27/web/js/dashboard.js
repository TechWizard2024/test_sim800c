// Gestion du tableau de bord
class DashboardManager {
    constructor() {
        this.modules = new Map();
        this.charts = {};
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.startMonitoring();
    }
    
    setupEventListeners() {
        // Rafraîchissement manuel
        const refreshBtn = document.getElementById('refresh-dashboard');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refresh());
        }
    }
    
    async refresh() {
        await this.loadModules();
        this.updateCharts();
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            this.modules.clear();
            modules.forEach(module => {
                this.modules.set(module.module_id || module.port, module);
            });
            
            this.render();
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    render() {
        const container = document.getElementById('modules-container');
        if (!container) return;
        
        if (this.modules.size === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">🔌</div>
                    <h3>Aucun module détecté</h3>
                    <p>Vérifiez que les modules SIM800C sont correctement connectés aux ports COM5, COM6 et COM7.</p>
                    <button class="btn-primary" onclick="app.discoverModules()">🔄 Détecter les modules</button>
                </div>
            `;
            return;
        }
        
        let html = '';
        for (const [id, module] of this.modules) {
            html += this.renderModuleCard(id, module);
        }
        
        container.innerHTML = html;
        this.attachModuleEvents();
    }
    
    renderModuleCard(id, module) {
        return `
            <div class="module-card" data-module-id="${id}">
                <div class="card-header">
                    <div>
                        <h3>📡 ${module.port || 'Module ' + id}</h3>
                        <small>${module.imei || 'IMEI non détecté'}</small>
                    </div>
                    <span class="status-badge ${module.status || 'connected'}">
                        ${module.status === 'connected' ? '● Connecté' : '○ Déconnecté'}
                    </span>
                </div>
                
                <div class="sim-info">
                    <div class="info-row">
                        <span class="info-label">📱 Numéro:</span>
                        <span class="info-value">${module.phone_number || 'Non détecté'}</span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">📶 Opérateur:</span>
                        <span class="info-value">${module.carrier || 'Inconnu'}</span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">🔒 PIN:</span>
                        <span class="info-value ${module.pin_unlocked ? 'pin-ok' : (module.pin_failed ? 'pin-error' : 'pin-locked')}">
                            ${module.pin_unlocked ? '✅ Déverrouillé' : (module.pin_failed ? '❌ Échec PIN' : '⏳ En attente...')}
                        </span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">📶 Signal:</span>
                        <span class="info-value signal-${getSignalClass(module.signal_quality)}">
                            ${getSignalIcon(module.signal_quality)} ${module.signal_rssi || 'N/A'} (CSQ:${module.signal_quality ?? '?'})
                        </span>
                    </div>
                    <!-- MICRO-BLOC C6 — Sparkline signal -->
                    <div class="info-row" id="sparkline-row-${id}">
                        <span class="info-label">📈 Tendance:</span>
                        <span class="info-value">
                            <svg id="sparkline-${id}" width="120" height="24" viewBox="0 0 120 24" style="vertical-align:middle; overflow:visible;">
                                <text x="60" y="16" text-anchor="middle" font-size="10" fill="var(--subtext,#999)">Chargement…</text>
                            </svg>
                        </span>
                    </div>
                    <div class="info-row">
                        <span class="info-label">🌐 Réseau:</span>
                        <span class="info-value network-${module.network_status || 'unknown'}">
                            ${getNetworkStatusLabel(module.network_status)}
                        </span>
                    </div>
                </div>
                
                <div class="quick-actions">
                    <button class="btn-quick" data-action="status" data-module="${id}" title="Consulter le crédit">
                        💰 Crédit
                    </button>
                    <button class="btn-quick" data-action="menu" data-module="${id}" title="Explorer le menu">
                        📋 Menu
                    </button>
                    <button class="btn-quick" data-action="ussd" data-module="${id}" title="Exécuter USSD personnalisé">
                        🔧 USSD
                    </button>
                    <button class="btn-quick" data-action="sms" data-module="${id}" title="Envoyer un SMS">
                        ✉️ SMS
                    </button>
                    <button class="btn-quick" data-action="refresh_signal" data-module="${id}" title="Rafraîchir le signal et l'état réseau">
                        📡 Signal
                    </button>
                    <button class="btn-quick btn-auto-status" data-action="auto_status" data-module="${id}" title="SIM Status Auto-Discovery pour ce module uniquement">
                        🚀 Auto-Status
                    </button>
                    <button class="btn-quick btn-auto-menu" data-action="auto_menu" data-module="${id}" title="USSD Menu Auto-Discovery pour ce module uniquement">
                        🌲 Auto-Menu
                    </button>
                </div>

                
                <div class="module-results" id="results-${id}" style="display: none;">
                    <div class="result-header">
                        <span>Résultat</span>
                        <button class="close-results" data-module="${id}">✕</button>
                    </div>
                    <pre class="result-content"></pre>
                </div>
            </div>
        `;
    }
    
    attachModuleEvents() {
        // Boutons d'action rapide
        document.querySelectorAll('.btn-quick').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = btn.dataset.action;
                const moduleId = btn.dataset.module;
                this.handleQuickAction(action, moduleId);
            });
        });
        
        // Fermeture des résultats
        document.querySelectorAll('.close-results').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const moduleId = btn.dataset.module;
                const resultsDiv = document.getElementById(`results-${moduleId}`);
                if (resultsDiv) resultsDiv.style.display = 'none';
            });
        });
    }
    
    async handleQuickAction(action, moduleId) {
        const resultsDiv = document.getElementById(`results-${moduleId}`);
        const resultContent = resultsDiv?.querySelector('.result-content');
        
        if (resultsDiv) {
            resultsDiv.style.display = 'block';
            if (resultContent) resultContent.textContent = 'Exécution en cours...';
        }
        
        try {
            let response;
            switch(action) {
                case 'status':
                    response = await this.executeUSSD(moduleId, '#122#');
                    break;
                case 'menu':
                    response = await this.executeUSSD(moduleId, '#144#');
                    break;
                case 'ussd':
                    const code = prompt('Entrez le code USSD:');
                    if (!code) return;
                    response = await this.executeUSSD(moduleId, code);
                    break;
                case 'sms':
                    this.showSendSMSModal(moduleId);
                    return;
                case 'refresh_signal':
                    await this.refreshSignal(moduleId, resultsDiv, resultContent);
                    return;
                case 'auto_status':
                    await this.runModuleAutoStatus(moduleId, resultsDiv, resultContent);
                    return;
                case 'auto_menu':
                    await this.runModuleAutoMenu(moduleId, resultsDiv, resultContent);
                    return;
            }

            
            if (resultContent) {
                resultContent.textContent = typeof response === 'string' ? response : JSON.stringify(response, null, 2);
            }
        } catch (error) {
            if (resultContent) {
                resultContent.textContent = `Erreur: ${error.message}`;
            }
        }
    }
    
    async runModuleAutoStatus(moduleId, resultsDiv, resultContent) {
        if (resultsDiv) resultsDiv.style.display = 'block';
        if (resultContent) resultContent.textContent = '🚀 Auto-Status en cours... (résultats en temps réel via WebSocket)';
        try {
            const resp = await fetch(`/api/modules/${moduleId}/ussd/auto-status`, { method: 'POST' });
            if (!resp.ok) throw new Error(await resp.text());
            const data = await resp.json();

            const lines = [];
            if (data.results) {
                for (const [op, res] of Object.entries(data.results)) {
                    lines.push(`✔ ${op}:\n${res}`);
                }
            }

            if (resultContent) resultContent.textContent = lines.length > 0 ? lines.join('\n\n') : JSON.stringify(data, null, 2);
        } catch (e) {
            if (resultContent) resultContent.textContent = `Erreur auto-status: ${e.message}`;
        }
    }

    async runModuleAutoMenu(moduleId, resultsDiv, resultContent) {
        if (resultsDiv) resultsDiv.style.display = 'block';
        if (resultContent) resultContent.textContent = '🌲 Auto-Menu en cours... (résultats en temps réel via WebSocket)';
        try {
            const resp = await fetch(`/api/modules/${moduleId}/ussd/auto-menu`, { method: 'POST' });
            if (!resp.ok) throw new Error(await resp.text());
            const data = await resp.json();

            const lines = [];
            if (data.results) {
                for (const [code, entry] of Object.entries(data.results)) {
                    lines.push(`📋 ${entry.operation || code}:`);
                    if (entry.error) {
                        lines.push(`  ❌ ${entry.error}`);
                    } else {
                        lines.push(entry.menu || '(vide)');
                    }
                    if (entry.new_codes && entry.new_codes.length > 0) {
                        lines.push(`  🆕 ${entry.new_codes.length} nouveau(x) code(s) découvert(s)`);
                    }
                    lines.push('');
                }
            }

            if (resultContent) resultContent.textContent = lines.length > 0 ? lines.join('\n') : JSON.stringify(data, null, 2);
        } catch (e) {
            if (resultContent) resultContent.textContent = `Erreur auto-menu: ${e.message}`;
        }
    }

    async refreshSignal(moduleId, resultsDiv, resultContent) {
        if (resultsDiv) resultsDiv.style.display = 'block';
        if (resultContent) resultContent.textContent = '📡 Actualisation du signal...';

        try {
            const resp = await fetch(`/api/modules/${moduleId}/signal`);
            if (!resp.ok) throw new Error(await resp.text());
            const data = await resp.json();
            // Update the card in place
            const signalEl = document.querySelector(`#module-${moduleId} .signal-strong, #module-${moduleId} .signal-medium, #module-${moduleId} .signal-weak, #module-${moduleId} .signal-none`);
            if (signalEl) {
                signalEl.className = `info-value signal-${getSignalClass(data.signal_quality)}`;
                signalEl.textContent = `${getSignalIcon(data.signal_quality)} ${data.signal_rssi} (CSQ:${data.signal_quality})`;
            }
            if (resultContent) resultContent.textContent = `📡 Signal: ${data.signal_rssi} (CSQ:${data.signal_quality})\n🌐 Réseau: ${getNetworkStatusLabel(data.network_status)}`;
        } catch(e) {
            if (resultContent) resultContent.textContent = `Erreur signal: ${e.message}`;
        }
    }

    async executeUSSD(moduleId, code, inputData = '') {
        const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                module_id: parseInt(moduleId), 
                ussd_code: code,
                input_data: inputData
            })
        });
        
        if (!response.ok) {
            const error = await response.text();
            throw new Error(error);
        }
        
        const result = await response.json();
        return result.result || result;
    }
    
    showSendSMSModal(moduleId) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>Envoyer un SMS</h2>
                <form id="quick-sms-form">
                    <label>Numéro destinataire:</label>
                    <input type="tel" id="sms-number" placeholder="0701010101" required>
                    <label>Message:</label>
                    <textarea id="sms-message" rows="4" required></textarea>
                    <button type="submit" class="btn-primary">Envoyer</button>
                </form>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        const closeBtn = modal.querySelector('.close');
        closeBtn.onclick = () => modal.remove();
        
        const form = modal.querySelector('#quick-sms-form');
        form.onsubmit = async (e) => {
            e.preventDefault();
            const number = document.getElementById('sms-number').value;
            const message = document.getElementById('sms-message').value;
            
            try {
                const response = await fetch(`/api/modules/${moduleId}/sms/send`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ number, message })
                });
                
                if (response.ok) {
                    alert('SMS envoyé avec succès');
                    modal.remove();
                } else {
                    alert('Erreur lors de l\'envoi');
                }
            } catch (error) {
                alert('Erreur: ' + error.message);
            }
        };
    }
    
    updateCharts() {
        // Implémenter les graphiques si nécessaire
    }
    
    startMonitoring() {
        // Rafraîchir toutes les 30 secondes
        setInterval(() => {
            this.loadModules();
        }, 30000);
        // MICRO-BLOC C6 — Rafraîchir sparklines toutes les 30 secondes
        setInterval(() => {
            this.refreshAllSparklines();
        }, 30000);
    }

    // MICRO-BLOC C6 — Charge l'historique signal et dessine la sparkline pour un module
    async loadSparkline(moduleId, dbId) {
        if (!dbId || dbId <= 0) return;
        try {
            const resp = await fetch(`/api/modules/${dbId}/signal/history?limit=20`);
            if (!resp.ok) return;
            const history = await resp.json();
            this.drawSparkline(moduleId, history);
        } catch(e) {
            console.warn('Sparkline error mod', moduleId, e);
        }
    }

    // MICRO-BLOC C6 — Dessine la sparkline SVG inline (20 dernières valeurs CSQ)
    drawSparkline(moduleId, history) {
        const svg = document.getElementById(`sparkline-${moduleId}`);
        if (!svg) return;
        // history est trié DESC (le plus récent en premier), on l'inverse pour afficher du plus vieux au plus récent
        const pts = [...history].reverse();
        if (!pts.length) {
            svg.innerHTML = `<text x="60" y="16" text-anchor="middle" font-size="10" fill="var(--subtext,#999)">Pas de données</text>`;
            return;
        }
        const W = 120, H = 24, pad = 4;
        const maxCSQ = 31;
        const n = pts.length;
        const xStep = n < 2 ? W : (W - pad*2) / (n - 1);
        const points = pts.map((p, i) => {
            const x = pad + i * xStep;
            const csq = Math.max(0, Math.min(maxCSQ, p.csq || 0));
            const y = H - pad - ((csq / maxCSQ) * (H - pad*2));
            return `${x.toFixed(1)},${y.toFixed(1)}`;
        }).join(' ');
        // Couleur selon dernière valeur
        const lastCSQ = pts[pts.length-1]?.csq || 0;
        const color = lastCSQ >= 20 ? '#22c55e' : (lastCSQ >= 10 ? '#f59e0b' : '#ef4444');
        svg.innerHTML = `
            <polyline points="${points}" fill="none" stroke="${color}" stroke-width="1.5" stroke-linejoin="round"/>
            <circle cx="${(pad + (n-1)*xStep).toFixed(1)}" cy="${(H - pad - ((Math.max(0,Math.min(maxCSQ,pts[pts.length-1]?.csq||0))/maxCSQ)*(H-pad*2))).toFixed(1)}" r="2.5" fill="${color}"/>
        `;
        svg.title = `Signal: ${pts.map(p=>p.csq).join(', ')}`;
    }

    // MICRO-BLOC C6 — Rafraîchit toutes les sparklines
    refreshAllSparklines() {
        for (const [id, module] of Object.entries(this.modules)) {
            const dbId = module.db_id || module.id;
            this.loadSparkline(id, dbId);
        }
    }
}

// Initialiser le gestionnaire de dashboard
document.addEventListener('DOMContentLoaded', () => {
    window.dashboardManager = new DashboardManager();
});
// ─── Signal / Network helpers ─────────────────────────────────────────────────
function getSignalClass(csq) {
    if (csq === undefined || csq === null || csq === 99 || csq === 0) return 'none';
    if (csq >= 20) return 'strong';
    if (csq >= 10) return 'medium';
    return 'weak';
}
function getSignalIcon(csq) {
    if (csq === undefined || csq === null || csq === 99 || csq === 0) return '📵';
    if (csq >= 20) return '📶';
    if (csq >= 10) return '📶';
    return '📶';
}
function getNetworkStatusLabel(status) {
    const labels = {
        'registered': '✅ Connecté',
        'roaming':    '🌍 Roaming',
        'searching':  '🔍 Recherche...',
        'denied':     '❌ Refusé',
        'not_registered': '⚠️ Non enregistré',
        'unknown':    '❓ Inconnu'
    };
    return labels[status] || '❓ Inconnu';
}