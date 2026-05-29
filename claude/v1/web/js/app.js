// Application principale SIM800C Supervisor
class SIM800CApp {
    constructor() {
        this.wsManager = null;
        this.currentTab = 'dashboard';
        this.modules = new Map();
        this.init();
    }
    
    async init() {
        console.log('Initialisation de SIM800C Supervisor');
        
        // Initialiser WebSocket
        this.initWebSocket();
        
        // Configurer les événements UI
        this.setupEventListeners();
        
        // Charger les données initiales
        await this.loadModules();
        this.loadSystemStatus();
        
        // MICRO-BLOC A4 — Charger le nombre de SMS non-lus au démarrage
        await this.loadInitialUnreadCount();
        
        // Démarrer les rafraîchissements périodiques
        this.startPeriodicRefresh();
    }
    
    initWebSocket() {
        this.wsManager = new WebSocketManager();
        this.wsManager.connect();
        
        this.wsManager.onMessage((event) => {
            this.handleWebSocketEvent(event);
        });
    }
    
    setupEventListeners() {
        // Navigation par onglets
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = btn.dataset.tab;
                this.switchTab(tab);
            });
        });
        
        // Boutons d'action globaux
        document.getElementById('auto-status-btn')?.addEventListener('click', () => this.runAutoStatus());
        document.getElementById('auto-menu-btn')?.addEventListener('click', () => this.runAutoMenu());
        document.getElementById('discover-modules-btn')?.addEventListener('click', () => this.discoverModules());
        document.getElementById('refresh-dashboard')?.addEventListener('click', () => this.refreshDashboard());
        
        // Thème
        document.getElementById('theme-toggle')?.addEventListener('click', () => this.toggleTheme());
    }
    
    switchTab(tabName) {
        // Mettre à jour les onglets
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        
        // Mettre à jour le contenu
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-tab`);
        });
        
        this.currentTab = tabName;
        
        // Recharger les données spécifiques à l'onglet
        if (tabName === 'sms') {
            if (window.smsManager) window.smsManager.loadSMS();
        } else if (tabName === 'history') {
            if (window.historyManager) window.historyManager.loadHistory();
        }
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            this.modules.clear();
            modules.forEach(module => {
                this.modules.set(module.module_id || module.port, module);
            });
            
            this.renderModules();
            this.updateModuleSelectors();
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    renderModules() {
        const container = document.getElementById('modules-container');
        if (!container) return;
        
        if (this.modules.size === 0) {
            container.innerHTML = '<div class="loading">Aucun module détecté. Vérifiez les connexions USB.</div>';
            return;
        }
        
        let html = '';
        for (const [id, module] of this.modules) {
            html += `
                <div class="module-card" data-module-id="${id}">
                    <div class="card-header">
                        <h3>📡 Module ${module.port || id}</h3>
                        <span class="status-badge ${module.status || 'connected'}">${module.status || 'Connecté'}</span>
                    </div>
                    <div class="sim-info">
                        <p><strong>IMEI:</strong> ${module.imei || 'Non détecté'}</p>
                        <p><strong>Numéro:</strong> ${module.phone_number || 'Non détecté'}</p>
                        <p><strong>Opérateur:</strong> ${module.carrier || 'Inconnu'}</p>
                    </div>
                    <div class="actions">
                        <button class="btn-sm btn-status" data-module="${id}">📊 SIM Status</button>
                        <button class="btn-sm btn-menu" data-module="${id}">🌲 Explorer Menu</button>
                        <button class="btn-sm btn-ussd" data-module="${id}">🔧 USSD Custom</button>
                    </div>
                    <div class="results" id="results-${id}" style="display: none;">
                        <pre></pre>
                    </div>
                </div>
            `;
        }
        
        container.innerHTML = html;
        
        // Attacher les événements
        document.querySelectorAll('.btn-status').forEach(btn => {
            btn.addEventListener('click', (e) => this.runModuleStatus(btn.dataset.module));
        });
        document.querySelectorAll('.btn-menu').forEach(btn => {
            btn.addEventListener('click', (e) => this.runModuleMenu(btn.dataset.module));
        });
        document.querySelectorAll('.btn-ussd').forEach(btn => {
            btn.addEventListener('click', (e) => this.showUSSDModal(btn.dataset.module));
        });
    }
    
    updateModuleSelectors() {
        const selectors = ['#ussd-module-select', '#modal-sms-module', '#sms-module-select', '#history-module-select'];
        
        selectors.forEach(selector => {
            const select = document.querySelector(selector);
            if (select) {
                let options = '<option value="">Sélectionner un module</option>';
                for (const [id, module] of this.modules) {
                    options += `<option value="${id}">${module.port || id} - ${module.phone_number || 'No SIM'}</option>`;
                }
                select.innerHTML = options;
            }
        });
    }
    
    async runAutoStatus() {
        this.showNotification('Exécution de SIM Status Auto-Discovery...', 'info');
        
        try {
            const response = await fetch('/api/ussd/auto-status', { method: 'POST' });
            const results = await response.json();
            
            this.showNotification('SIM Status Auto-Discovery terminé', 'success');
            this.displayResults(results);
        } catch (error) {
            this.showNotification('Erreur: ' + error.message, 'error');
        }
    }
    
    async runAutoMenu() {
        this.showNotification('Exécution de USSD Menu Auto-Discovery...', 'info');
        
        try {
            const response = await fetch('/api/ussd/auto-menu', { method: 'POST' });
            const results = await response.json();
            
            this.showNotification('USSD Menu Auto-Discovery terminé', 'success');
            this.displayResults(results);
        } catch (error) {
            this.showNotification('Erreur: ' + error.message, 'error');
        }
    }
    
    async runModuleStatus(moduleId) {
        const resultsDiv = document.getElementById(`results-${moduleId}`);
        const pre = resultsDiv?.querySelector('pre');
        
        if (resultsDiv) {
            resultsDiv.style.display = 'block';
            if (pre) pre.textContent = 'Exécution en cours...';
        }
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ module_id: parseInt(moduleId), ussd_code: '#122#' })
            });
            
            const result = await response.json();
            if (pre) pre.textContent = result.result || JSON.stringify(result, null, 2);
        } catch (error) {
            if (pre) pre.textContent = 'Erreur: ' + error.message;
        }
    }
    
    async runModuleMenu(moduleId) {
        const resultsDiv = document.getElementById(`results-${moduleId}`);
        const pre = resultsDiv?.querySelector('pre');
        
        if (resultsDiv) {
            resultsDiv.style.display = 'block';
            if (pre) pre.textContent = 'Exploration du menu en cours...';
        }
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ module_id: parseInt(moduleId), ussd_code: '#144#' })
            });
            
            const result = await response.json();
            if (pre) pre.textContent = result.result || JSON.stringify(result, null, 2);
        } catch (error) {
            if (pre) pre.textContent = 'Erreur: ' + error.message;
        }
    }
    
    showUSSDModal(moduleId) {
        const code = prompt('Entrez le code USSD à exécuter (ex: #122#):');
        if (!code) return;
        
        const inputData = prompt('Données d\'entrée (optionnel):', '');
        
        this.executeCustomUSSD(moduleId, code, inputData);
    }
    
    async executeCustomUSSD(moduleId, code, inputData) {
        const resultsDiv = document.getElementById(`results-${moduleId}`);
        const pre = resultsDiv?.querySelector('pre');
        
        if (resultsDiv) {
            resultsDiv.style.display = 'block';
            if (pre) pre.textContent = 'Exécution en cours...';
        }
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    module_id: parseInt(moduleId), 
                    ussd_code: code,
                    input_data: inputData
                })
            });
            
            const result = await response.json();
            if (pre) pre.textContent = result.result || JSON.stringify(result, null, 2);
        } catch (error) {
            if (pre) pre.textContent = 'Erreur: ' + error.message;
        }
    }
    
    async discoverModules() {
        this.showNotification('Recherche des modules en cours...', 'info');
        
        try {
            await fetch('/api/discover', { method: 'POST' });
            await this.loadModules();
            this.showNotification('Modules détectés avec succès', 'success');
        } catch (error) {
            this.showNotification('Erreur: ' + error.message, 'error');
        }
    }
    
    // MICRO-BLOC A4 — Chargement initial du total de SMS non-lus pour tous les modules
    async loadInitialUnreadCount() {
        try {
            const moduleIds = Array.from(this.modules.keys());
            if (moduleIds.length === 0) return;

            let totalUnread = 0;
            const fetches = moduleIds.map(async (id) => {
                try {
                    const resp = await fetch(`/api/modules/${id}/sms/unread-count`, {
                        headers: { 'Authorization': 'Bearer ' + (localStorage.getItem('token') || '') }
                    });
                    if (!resp.ok) return 0;
                    const data = await resp.json();
                    return data.unread_count || 0;
                } catch {
                    return 0;
                }
            });

            const counts = await Promise.all(fetches);
            totalUnread = counts.reduce((sum, c) => sum + c, 0);

            if (window.smsManager) {
                window.smsManager.setUnreadBadge(totalUnread);
            } else {
                this._updateUnreadBadgeDirect(totalUnread);
            }
        } catch (error) {
            console.warn('loadInitialUnreadCount:', error);
        }
    }

    // MICRO-BLOC A4 — Mise à jour directe du badge sans passer par smsManager
    _updateUnreadBadgeDirect(count) {
        const badgeTab    = document.getElementById('sms-unread-badge');
        const badgeInline = document.getElementById('sms-unread-badge-inline');
        [badgeTab, badgeInline].forEach(el => {
            if (!el) return;
            if (count > 0) {
                el.textContent = count > 99 ? '99+' : count;
                el.style.display = 'inline-block';
            } else {
                el.textContent = '0';
                el.style.display = 'none';
            }
        });
    }

    async refreshDashboard() {
        await this.loadModules();
        this.showNotification('Dashboard actualisé', 'success');
    }
    
    handleWebSocketEvent(event) {
        console.log('Événement WebSocket:', event);
        
        switch(event.type) {
            case 'module_update':
            case 'module_connected':
            case 'module_initialized':
                this.loadModules();
                this.loadSystemStatus();
                break;
            case 'module_disconnected':
                this.loadModules();
                this.loadSystemStatus();
                this.showNotification(`⚠️ Module déconnecté: ${event.data?.port || 'module ' + event.module_id}`, 'error');
                break;
            case 'pin_unlocked':
                this.showNotification(`🔓 PIN déverrouillé sur ${event.data?.port || 'module ' + event.module_id}`, 'success');
                this.loadModules(); // refresh card to show PIN status
                break;
            case 'pin_failed':
                this.showNotification(`🔒 Échec déverrouillage PIN sur ${event.data?.port || 'module ' + event.module_id}`, 'error');
                this.loadModules();
                break;
            case 'auto_status_progress': {
                const d = event.data;
                const liveDiv = document.getElementById('auto-discovery-result');
                if (liveDiv) {
                    const entry = document.createElement('div');
                    entry.className = 'live-progress-item';
                    entry.innerHTML = `<span class="live-port">[${d.port}]</span> <span class="live-op">${d.operation}</span> <code>${d.ussd_code}</code><br><pre class="live-result">${d.result || ''}</pre>`;
                    const existing = liveDiv.querySelector('pre');
                    if (existing) existing.remove();
                    liveDiv.appendChild(entry);
                }
                break;
            }
            case 'auto_menu_progress': {
                const d = event.data;
                const liveDiv = document.getElementById('auto-menu-result');
                if (liveDiv) {
                    if (d.status === 'exploring') {
                        const entry = document.createElement('div');
                        entry.className = 'live-progress-item exploring';
                        entry.id = `menu-progress-${event.module_id}-${d.ussd_code.replace(/[^a-z0-9]/gi,'_')}`;
                        entry.innerHTML = `<span class="live-port">[${d.port}]</span> ⏳ Exploration <code>${d.ussd_code}</code> — ${d.operation}...`;
                        const existing = liveDiv.querySelector('pre');
                        if (existing) existing.remove();
                        liveDiv.appendChild(entry);
                    } else if (d.status === 'done') {
                        const entryId = `menu-progress-${event.module_id}-${d.ussd_code.replace(/[^a-z0-9]/gi,'_')}`;
                        const existing = document.getElementById(entryId);
                        const entry = existing || document.createElement('div');
                        entry.className = 'live-progress-item done';
                        const tree = d.result?.menu_tree || d.result?.error || '';
                        entry.innerHTML = `<span class="live-port">[${d.port}]</span> ✅ <code>${d.ussd_code}</code> — ${d.operation} (${d.result?.discovered_codes || 0} codes)<pre class="live-result">${tree}</pre>`;
                        if (!existing) liveDiv.appendChild(entry);
                    }
                }
                // Indicateur "exploration en cours" sur la carte module
                const moduleCard = document.getElementById(`module-${event.module_id}`);
                if (moduleCard) {
                    let badge = moduleCard.querySelector('.exploring-badge');
                    if (d.status === 'exploring') {
                        if (!badge) {
                            badge = document.createElement('span');
                            badge.className = 'exploring-badge';
                            badge.style.cssText = 'display:inline-block;background:#f59e0b;color:#fff;border-radius:4px;padding:2px 8px;font-size:0.75rem;margin-left:8px;animation:pulse 1.2s infinite;';
                            badge.innerHTML = '⏳ Exploration...';
                            const cardHeader = moduleCard.querySelector('.module-header, h3, .card-title');
                            if (cardHeader) cardHeader.appendChild(badge);
                        }
                        moduleCard.querySelectorAll('.btn-auto-menu, .btn-menu-explore').forEach(btn => {
                            btn.disabled = true;
                            btn.dataset.wasDisabledByExploration = '1';
                        });
                    } else if (d.status === 'done') {
                        if (badge) badge.remove();
                        moduleCard.querySelectorAll('[data-was-disabled-by-exploration]').forEach(btn => {
                            btn.disabled = false;
                            delete btn.dataset.wasDisabledByExploration;
                        });
                    }
                }
                break;
            }
            case 'signal_update':
                // Update signal display without full reload
                if (event.data) {
                    const d = event.data;
                    const sigEl = document.querySelector(`#module-${event.module_id} .info-row .info-value[class*="signal-"]`);
                    if (sigEl) {
                        const cls = typeof getSignalClass === 'function' ? getSignalClass(d.signal_quality) : 'unknown';
                        sigEl.className = `info-value signal-${cls}`;
                        sigEl.textContent = `${d.signal_rssi} (CSQ:${d.signal_quality})`;
                    }
                }
                break;
            case 'ussd_result':
                this.displayUSSDResult(event.module_id, event.data);
                break;
            case 'sms_received':
                if (window.smsManager) window.smsManager.addSMS(event.data);
                this.showNotification(`Nouveau SMS reçu sur module ${event.module_id}`, 'info');
                // MICRO-BLOC A4 — Incrémenter le badge de 1 lors d'un nouveau SMS entrant
                if (window.smsManager && event.data && event.data.direction === 'in') {
                    window.smsManager.incrementUnreadBadge(1);
                }
                break;
            case 'sms_auto_trash': {
                const d = event.data;
                const preview = d.preview ? ` — "${d.preview}"` : '';
                this.showNotification(`📂 SMS auto-corbeille (module ${event.module_id}) de ${d.sender || '?'}${preview}`, 'warning');
                if (window.smsManager) window.smsManager.loadSMS();
                break;
            }
            case 'sms_moved_to_trash':
            case 'sms_restored':
            case 'sms_deleted_permanent':
                if (window.smsManager) window.smsManager.loadSMS();
                break;
            // MICRO-BLOC A4 — Mise à jour badge non-lus via WebSocket sms_unread_count
            case 'sms_unread_count':
                if (event.data) {
                    const count = event.data.unread_count || 0;
                    // Mettre à jour via smsManager si disponible
                    if (window.smsManager) {
                        window.smsManager.setUnreadBadge(count);
                    } else {
                        // Fallback direct si smsManager pas encore initialisé
                        this._updateUnreadBadgeDirect(count);
                    }
                }
                break;
            case 'config_updated': {
                const changedStr = (event.data.changed || []).join(', ');
                this.showNotification(`⚙️ Configuration mise à jour: ${changedStr}`, 'success');
                if (window.settingsManager) window.settingsManager.loadAdvancedSettings();
                break;
            }

            case 'dialplan_reloaded':
                this.showNotification(`🔄 Plan de numérotation rechargé (${event.data?.count || '?'} entrées)`, 'success');
                // Refresh settings dialplan table if visible
                if (window.settingsManager) window.settingsManager.loadDialPlan();
                break;
            case 'discovery_scan_complete': {
                const d = event.data;
                if (d && d.new_found > 0) {
                    this.showNotification(`🔍 Scan terminé: ${d.new_found} nouveau(x) module(s) détecté(s) (total: ${d.modules_total})`, 'success');
                    this.loadModules();
                    this.loadSystemStatus();
                }
                break;
            }
        }
    }
    
    displayUSSDResult(moduleId, result) {
        const resultsDiv = document.getElementById(`results-${moduleId}`);
        const pre = resultsDiv?.querySelector('pre');
        if (pre) {
            pre.textContent = typeof result === 'string' ? result : JSON.stringify(result, null, 2);
        }
    }
    
    displayResults(results) {
        const container = document.getElementById('modules-container');
        if (!container) return;
        
        // Créer un modal pour afficher les résultats
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content" style="width: 80%; max-width: 800px;">
                <span class="close">&times;</span>
                <h2>Résultats de l'opération</h2>
                <pre style="max-height: 500px; overflow: auto;">${JSON.stringify(results, null, 2)}</pre>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        modal.querySelector('.close').onclick = () => modal.remove();
        modal.onclick = (e) => { if (e.target === modal) modal.remove(); };
    }
    
    showNotification(message, type = 'info') {
        // Créer une notification toast
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            padding: 12px 20px;
            background: ${type === 'error' ? '#f44336' : type === 'success' ? '#4caf50' : '#2196f3'};
            color: white;
            border-radius: 8px;
            z-index: 1000;
            animation: slideIn 0.3s ease;
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }
    
    toggleTheme() {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
        
        // Activer/désactiver les feuilles de style
        const lightSheet = document.getElementById('theme-light');
        const darkSheet = document.getElementById('theme-dark');
        
        if (newTheme === 'dark') {
            lightSheet.disabled = true;
            darkSheet.disabled = false;
        } else {
            lightSheet.disabled = false;
            darkSheet.disabled = true;
        }
    }
    
    async loadSystemStatus() {
        try {
            const resp = await fetch('/api/system/status', {
                headers: { 'Authorization': 'Bearer ' + (localStorage.getItem('token') || '') }
            });
            if (!resp.ok) return;
            const d = await resp.json();

            const setText = (id, val) => {
                const el = document.getElementById(id);
                if (el) el.textContent = val;
            };

            setText('sys-uptime', d.uptime || '—');
            setText('sys-modules', `${d.modules_total || 0} (⚠️ PIN KO: ${d.modules_pin_fail || 0})`);
            setText('sys-db', d.database?.ok ? `✅ ${d.database.database}` : `❌ ${d.database.error}`);
            setText('sys-time', d.server_time ? new Date(d.server_time).toLocaleTimeString('fr-FR') : '—');
            setText('sys-explore-delay', d.config ? `${d.config.explore_delay_ms} ms` : '—');
            setText('sys-max-depth', d.config ? `${d.config.max_menu_depth}` : '—');
        } catch (e) {
            // silencieux — pas critique
        }
    }

        startPeriodicRefresh() {
        // Rafraîchir le dashboard toutes les 30 secondes
        setInterval(() => {
            if (this.currentTab === 'dashboard') {
                this.loadModules();
                this.loadSystemStatus();
            }
        }, 30000);
    }
}

// Styles pour les animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);

// Initialiser l'application au chargement
document.addEventListener('DOMContentLoaded', () => {
    window.app = new SIM800CApp();
});

// Helpers globaux utilisés depuis les attributs onclick dans le HTML
function loadSystemStatus() { if (window.app) window.app.loadSystemStatus(); }