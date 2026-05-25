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
    
    async refreshDashboard() {
        await this.loadModules();
        this.showNotification('Dashboard actualisé', 'success');
    }
    
    handleWebSocketEvent(event) {
        console.log('Événement WebSocket:', event);
        
        switch(event.type) {
            case 'module_update':
            case 'module_connected':
                this.loadModules();
                break;
            case 'ussd_result':
                this.displayUSSDResult(event.module_id, event.data);
                break;
            case 'sms_received':
                if (window.smsManager) window.smsManager.addSMS(event.data);
                this.showNotification(`Nouveau SMS reçu sur module ${event.module_id}`, 'info');
                break;
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
    
    startPeriodicRefresh() {
        // Rafraîchir le dashboard toutes les 30 secondes
        setInterval(() => {
            if (this.currentTab === 'dashboard') {
                this.loadModules();
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