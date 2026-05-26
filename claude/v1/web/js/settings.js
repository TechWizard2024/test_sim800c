// Gestionnaire des paramètres — v1-9
class SettingsManager {
    constructor() {
        this.dialPlan = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadConfig();
        this.loadModulesConfig();
        this.loadDialPlan();
        this.loadAdvancedSettings();
        this.loadPortsWhitelist();
    }

    setupEventListeners() {
        const exportLogsBtn = document.getElementById('export-logs-btn');
        if (exportLogsBtn) exportLogsBtn.addEventListener('click', () => this.exportLogs());

        const clearLogsBtn = document.getElementById('clear-logs-btn');
        if (clearLogsBtn) clearLogsBtn.addEventListener('click', () => this.clearLogs());

        const backupDbBtn = document.getElementById('backup-db-btn');
        if (backupDbBtn) backupDbBtn.addEventListener('click', () => this.backupDatabase());

        const addDialPlanBtn = document.getElementById('add-dialplan-btn');
        if (addDialPlanBtn) addDialPlanBtn.addEventListener('click', () => this.showAddDialPlanModal());

        const saveDialPlanBtn = document.getElementById('save-dialplan-btn');
        if (saveDialPlanBtn) saveDialPlanBtn.addEventListener('click', () => this.saveDialPlanEntry());

        const cancelDialPlanBtn = document.getElementById('cancel-dialplan-btn');
        if (cancelDialPlanBtn) cancelDialPlanBtn.addEventListener('click', () => this.hideDialPlanModal());

        const saveDelaysBtn = document.getElementById('save-delays-btn');
        if (saveDelaysBtn) saveDelaysBtn.addEventListener('click', () => this.saveDelays());

        const reloadDialPlanBtn = document.getElementById('reload-dialplan-btn');
        if (reloadDialPlanBtn) reloadDialPlanBtn.addEventListener('click', () => this.reloadDialPlan());

        const saveAdvancedBtn = document.getElementById('save-advanced-btn');
        if (saveAdvancedBtn) saveAdvancedBtn.addEventListener('click', () => this.saveAdvancedSettings());

        const saveComWhitelistBtn = document.getElementById('save-com-whitelist-btn');
        if (saveComWhitelistBtn) saveComWhitelistBtn.addEventListener('click', () => this.savePortsWhitelist());
    }

    // ── Config display ──────────────────────────────────────────────────────
    async loadConfig() {
        try {
            const response = await fetch('/api/config');
            if (response.ok) {
                const config = await response.json();
                this.currentConfig = config;
                this.displayConfig(config);
                this.displayDelays(config);
            }
        } catch (error) {
            console.error('Erreur chargement configuration:', error);
        }
    }

    displayConfig(config) {
        const container = document.getElementById('config-display');
        if (!container) return;
        container.innerHTML = `
            <div class="config-section">
                <h4>Serveur</h4>
                <p>Port: ${config.server?.port || window.location.port || 8082}</p>
                <p>WebSocket: ${config.server?.websocket_path || '/ws'}</p>
            </div>
            <div class="config-section">
                <h4>Série</h4>
                <p>Ports: ${config.serial?.ports?.join(', ') || 'Auto-détection'}</p>
                <p>Baud rate: ${config.serial?.baud_rate || 9600}</p>
            </div>
            <div class="config-section">
                <h4>Base de données</h4>
                <p>Host: ${config.mysql?.host || 'localhost'}</p>
                <p>Database: ${config.mysql?.database || 'sim800c_manager'}</p>
            </div>`;
    }

    displayDelays(config) {
        const expEl = document.getElementById('explore-delay-input');
        const navEl = document.getElementById('nav-delay-input');
        if (expEl) expEl.value = config.ussd?.explore_delay_ms ?? 3000;
        if (navEl) navEl.value = config.ussd?.nav_delay_ms ?? 500;
    }

    async saveDelays() {
        const exploreDelay = parseInt(document.getElementById('explore-delay-input')?.value || '3000');
        const navDelay = parseInt(document.getElementById('nav-delay-input')?.value || '500');

        if (isNaN(exploreDelay) || exploreDelay < 500) {
            alert('❌ Délai exploration minimum: 500ms');
            return;
        }
        if (isNaN(navDelay) || navDelay < 100) {
            alert('❌ Délai navigation minimum: 100ms');
            return;
        }

        try {
            const response = await fetch('/api/config/delays', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ explore_delay_ms: exploreDelay, nav_delay_ms: navDelay })
            });
            if (response.ok) {
                alert('✅ Délais USSD mis à jour');
            } else {
                const txt = await response.text();
                alert('❌ Erreur: ' + txt);
            }
        } catch (error) {
            alert('❌ Erreur réseau: ' + error.message);
        }
    }

    async reloadDialPlan() {
        const btn = document.getElementById('reload-dialplan-btn');
        if (btn) btn.disabled = true;
        try {
            const response = await fetch('/api/dialplan/reload', { method: 'POST' });
            if (response.ok) {
                const result = await response.json();
                alert(`✅ ${result.message || 'Plan rechargé'}`);
                await this.loadDialPlan();
            } else {
                const txt = await response.text();
                alert('❌ Erreur: ' + txt);
            }
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        } finally {
            if (btn) btn.disabled = false;
        }
    }

    // ── Modules config ──────────────────────────────────────────────────────
    async loadModulesConfig() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            const container = document.getElementById('modules-config');
            if (!container) return;

            let html = '<div class="modules-config-list">';
            for (const module of modules) {
                const id = module.module_id || module.port;
                html += `
                    <div class="module-config-item">
                        <div class="config-header">
                            <strong>📡 ${module.port || 'Module ' + id}</strong>
                            <span class="status-badge ${module.status || 'connected'}">${module.status || 'Connecté'}</span>
                        </div>
                        <div class="config-details">
                            <p>IMEI: ${module.imei || 'Non détecté'}</p>
                            <p>Numéro: ${module.phone_number || 'Non détecté'}</p>
                            <p>Opérateur: ${module.carrier || 'Inconnu'}</p>
                        </div>
                        <div class="config-actions">
                            <button class="btn-sm btn-reset" data-port="${module.port}">🔄 Réinitialiser</button>
                            <button class="btn-sm btn-test" data-port="${module.port}">📡 Tester connexion</button>
                            <a class="btn-sm btn-export-sms" href="/api/modules/${id}/sms/export" download>📥 Export SMS</a>
                        </div>
                    </div>`;
            }
            html += '</div>';
            container.innerHTML = html;

            document.querySelectorAll('.btn-reset').forEach(btn => {
                btn.addEventListener('click', () => this.resetModule(btn.dataset.port));
            });
            document.querySelectorAll('.btn-test').forEach(btn => {
                btn.addEventListener('click', () => this.testModule(btn.dataset.port));
            });
        } catch (error) {
            console.error('Erreur chargement modules config:', error);
        }
    }

    // ── Dial Plan ────────────────────────────────────────────────────────────
    async loadDialPlan() {
        try {
            const response = await fetch('/api/dialplan');
            if (!response.ok) return;
            this.dialPlan = await response.json();
            this.renderDialPlan();
        } catch (error) {
            console.error('Erreur chargement plan de numérotation:', error);
        }
    }

    renderDialPlan() {
        const container = document.getElementById('dialplan-table-body');
        if (!container) return;

        if (!this.dialPlan || this.dialPlan.length === 0) {
            container.innerHTML = '<tr><td colspan="7" style="text-align:center;color:#999;">Aucune entrée</td></tr>';
            return;
        }

        container.innerHTML = this.dialPlan.map(entry => `
            <tr data-id="${entry.id}">
                <td>${entry.country_code}</td>
                <td>${entry.country_name}</td>
                <td>${entry.calling_code}</td>
                <td>${entry.operator}</td>
                <td><code>${entry.prefix}</code></td>
                <td>${entry.number_length}</td>
                <td>
                    <button class="btn-sm btn-edit-dp" data-id="${entry.id}" title="Modifier">✏️</button>
                    <button class="btn-sm btn-delete-dp" data-id="${entry.id}" title="Désactiver">🗑️</button>
                </td>
            </tr>`).join('');

        container.querySelectorAll('.btn-edit-dp').forEach(btn => {
            btn.addEventListener('click', () => this.showEditDialPlanModal(parseInt(btn.dataset.id)));
        });
        container.querySelectorAll('.btn-delete-dp').forEach(btn => {
            btn.addEventListener('click', () => this.deleteDialPlanEntry(parseInt(btn.dataset.id)));
        });
    }

    showAddDialPlanModal() {
        this._editingId = null;
        document.getElementById('dialplan-modal-title').textContent = 'Ajouter une entrée';
        document.getElementById('dp-country-code').value = '';
        document.getElementById('dp-country-name').value = '';
        document.getElementById('dp-calling-code').value = '';
        document.getElementById('dp-operator').value = '';
        document.getElementById('dp-prefix').value = '';
        document.getElementById('dp-number-length').value = '10';
        document.getElementById('dialplan-modal').style.display = 'flex';
    }

    showEditDialPlanModal(id) {
        const entry = this.dialPlan.find(e => e.id === id);
        if (!entry) return;
        this._editingId = id;
        document.getElementById('dialplan-modal-title').textContent = 'Modifier l\'entrée';
        document.getElementById('dp-country-code').value = entry.country_code;
        document.getElementById('dp-country-name').value = entry.country_name;
        document.getElementById('dp-calling-code').value = entry.calling_code;
        document.getElementById('dp-operator').value = entry.operator;
        document.getElementById('dp-prefix').value = entry.prefix;
        document.getElementById('dp-number-length').value = entry.number_length;
        document.getElementById('dialplan-modal').style.display = 'flex';
    }

    hideDialPlanModal() {
        const modal = document.getElementById('dialplan-modal');
        if (modal) modal.style.display = 'none';
    }

    async saveDialPlanEntry() {
        const entry = {
            country_code: document.getElementById('dp-country-code').value.trim().toUpperCase(),
            country_name: document.getElementById('dp-country-name').value.trim(),
            calling_code: document.getElementById('dp-calling-code').value.trim(),
            operator: document.getElementById('dp-operator').value.trim(),
            prefix: document.getElementById('dp-prefix').value.trim(),
            number_length: parseInt(document.getElementById('dp-number-length').value) || 10,
            is_active: true,
        };

        if (!entry.country_code || !entry.operator || !entry.prefix) {
            alert('❌ Code pays, opérateur et préfixe sont requis.');
            return;
        }

        try {
            let response;
            if (this._editingId) {
                response = await fetch(`/api/dialplan/${this._editingId}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(entry),
                });
            } else {
                response = await fetch('/api/dialplan', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(entry),
                });
            }

            if (response.ok) {
                this.hideDialPlanModal();
                await this.loadDialPlan();
            } else {
                const txt = await response.text();
                alert('❌ Erreur: ' + txt);
            }
        } catch (error) {
            alert('❌ Erreur réseau: ' + error.message);
        }
    }

    async deleteDialPlanEntry(id) {
        const entry = this.dialPlan.find(e => e.id === id);
        if (!confirm(`Désactiver l'entrée "${entry?.operator} (${entry?.prefix})" ?`)) return;
        try {
            const response = await fetch(`/api/dialplan/${id}`, { method: 'DELETE' });
            if (response.ok) {
                await this.loadDialPlan();
            } else {
                alert('❌ Erreur suppression');
            }
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        }
    }

    // ── Module actions ───────────────────────────────────────────────────────
    async resetModule(port) {
        if (!confirm(`Réinitialiser le module sur ${port} ?`)) return;
        try {
            const response = await fetch(`/api/modules/${encodeURIComponent(port)}/reset`, { method: 'POST' });
            if (response.ok) {
                alert('✅ Module réinitialisé');
                this.loadModulesConfig();
            }
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        }
    }

    async testModule(port) {
        try {
            const response = await fetch(`/api/modules/${encodeURIComponent(port)}/test`);
            if (response.ok) {
                const result = await response.json();
                alert('✅ Module OK\n' + JSON.stringify(result, null, 2));
            } else {
                alert('❌ Module non répondant');
            }
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        }
    }

    async exportLogs() {
        try {
            const response = await fetch('/api/logs/export');
            if (response.ok) {
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `logs_${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.log`;
                a.click();
                window.URL.revokeObjectURL(url);
            }
        } catch (error) {
            alert('❌ Erreur export: ' + error.message);
        }
    }

    async clearLogs() {
        if (!confirm('⚠️ Vider tous les logs ?')) return;
        try {
            const response = await fetch('/api/logs/clear', { method: 'DELETE' });
            if (response.ok) alert('✅ Logs vidés');
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        }
    }

    async backupDatabase() {
        try {
            const response = await fetch('/api/db/backup', { method: 'POST' });
            if (response.ok) {
                const result = await response.json();
                alert(`✅ Base sauvegardée: ${result.filename}`);
            }
        } catch (error) {
            alert('❌ Erreur sauvegarde: ' + error.message);
        }
    }

    // ── Paramètres avancés ──────────────────────────────────────────────────
    async loadAdvancedSettings() {
        try {
            const response = await fetch('/api/config/advanced');
            if (!response.ok) return;
            const data = await response.json();
            const kwEl = document.getElementById('adv-trash-keyword');
            const depthEl = document.getElementById('adv-max-menu-depth');
            const retryEl = document.getElementById('adv-retry-on-error');
            const maxRetryEl = document.getElementById('adv-max-retries');
            if (kwEl) kwEl.value = data.auto_trash_keyword || 'Test';
            if (depthEl) depthEl.value = data.max_menu_depth || 10;
            if (retryEl) retryEl.checked = !!data.retry_on_error;
            if (maxRetryEl) maxRetryEl.value = data.max_retries || 3;
        } catch (e) {
            console.warn('Erreur chargement paramètres avancés:', e);
        }
    }

    async saveAdvancedSettings() {
        const kwEl = document.getElementById('adv-trash-keyword');
        const depthEl = document.getElementById('adv-max-menu-depth');
        const retryEl = document.getElementById('adv-retry-on-error');
        const maxRetryEl = document.getElementById('adv-max-retries');

        const payload = {
            auto_trash_keyword: kwEl?.value?.trim() || 'Test',
            max_menu_depth: parseInt(depthEl?.value) || 10,
            retry_on_error: retryEl?.checked || false,
            max_retries: parseInt(maxRetryEl?.value) || 3,
        };

        try {
            const response = await fetch('/api/config/advanced', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (response.ok) {
                const result = await response.json();
                const changed = result.changed?.join(', ') || 'aucun';
                alert(`✅ Paramètres sauvegardés (${changed})`);
            } else {
                alert('❌ Erreur: ' + await response.text());
            }
        } catch (e) {
            alert('❌ Erreur réseau: ' + e.message);
        }
    }

    async loadPortsWhitelist() {
        try {
            const resp = await fetch('/api/config/ports');
            if (!resp.ok) return;
            const data = await resp.json();
            const el = document.getElementById('com-whitelist-input');
            if (el) el.value = (data.ports || []).join(', ');

            // Badge indicateur
            const badge = document.getElementById('com-whitelist-badge');
            if (badge) {
                const count = (data.ports || []).length;
                if (count > 0) {
                    badge.textContent = `✅ ${count} port(s) en priorité`;
                    badge.style.color = '#4caf50';
                } else {
                    badge.textContent = '⚠️ Aucun port en whitelist (scan complet COM1..COM99)';
                    badge.style.color = '#999';
                }
            }
        } catch (e) { /* silencieux */ }
    }

    async savePortsWhitelist() {
        const el = document.getElementById('com-whitelist-input');
        const statusEl = document.getElementById('com-whitelist-status');
        const csv = el ? el.value.trim() : '';
        try {
            const resp = await fetch('/api/config/ports', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ports_csv: csv })
            });
            if (resp.ok) {
                const data = await resp.json();
                if (statusEl) {
                    statusEl.style.display = 'block';
                    statusEl.textContent = '✅ ' + (data.message || 'Whitelist sauvegardée');
                    setTimeout(() => { statusEl.style.display = 'none'; }, 4000);
                }
                // Refresh badge
                this.loadPortsWhitelist();
            } else {
                alert('❌ Erreur: ' + await resp.text());
            }
        } catch (e) {
            alert('❌ Erreur réseau: ' + e.message);
        }
    }

    exportDialPlanCSV() {
        const a = document.createElement('a');
        a.href = '/api/dialplan/export';
        a.download = '';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.settingsManager = new SettingsManager();
});