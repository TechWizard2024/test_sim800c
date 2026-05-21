// Gestionnaire des paramètres
class SettingsManager {
    constructor() {
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.loadConfig();
        this.loadModulesConfig();
    }
    
    setupEventListeners() {
        const exportLogsBtn = document.getElementById('export-logs-btn');
        if (exportLogsBtn) {
            exportLogsBtn.addEventListener('click', () => this.exportLogs());
        }
        
        const clearLogsBtn = document.getElementById('clear-logs-btn');
        if (clearLogsBtn) {
            clearLogsBtn.addEventListener('click', () => this.clearLogs());
        }
        
        const backupDbBtn = document.getElementById('backup-db-btn');
        if (backupDbBtn) {
            backupDbBtn.addEventListener('click', () => this.backupDatabase());
        }
    }
    
    async loadConfig() {
        try {
            const response = await fetch('/api/config');
            if (response.ok) {
                const config = await response.json();
                this.displayConfig(config);
            }
        } catch (error) {
            console.error('Erreur chargement configuration:', error);
        }
    }
    
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
                        </div>
                    </div>
                `;
            }
            html += '</div>';
            container.innerHTML = html;
            
            // Attacher événements
            document.querySelectorAll('.btn-reset').forEach(btn => {
                btn.addEventListener('click', (e) => this.resetModule(btn.dataset.port));
            });
            document.querySelectorAll('.btn-test').forEach(btn => {
                btn.addEventListener('click', (e) => this.testModule(btn.dataset.port));
            });
            
        } catch (error) {
            console.error('Erreur chargement modules config:', error);
        }
    }
    
    displayConfig(config) {
        const container = document.getElementById('config-display');
        if (!container) return;
        
        container.innerHTML = `
            <div class="config-section">
                <h4>Serveur</h4>
                <p>Port: ${config.server?.port || 8080}</p>
                <p>WebSocket: ${config.server?.websocket_path || '/ws'}</p>
            </div>
            <div class="config-section">
                <h4>Série</h4>
                <p>Ports: ${config.serial?.ports?.join(', ') || 'COM5, COM6, COM7'}</p>
                <p>Baud rate: ${config.serial?.baud_rate || 9600}</p>
            </div>
            <div class="config-section">
                <h4>Base de données</h4>
                <p>Host: ${config.mysql?.host || 'localhost'}</p>
                <p>Database: ${config.mysql?.database || 'sim800c_manager'}</p>
            </div>
        `;
    }
    
    async resetModule(port) {
        if (!confirm(`Réinitialiser le module sur ${port} ?`)) return;
        
        try {
            const response = await fetch(`/api/modules/${port}/reset`, { method: 'POST' });
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
            const response = await fetch(`/api/modules/${port}/test`);
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
                a.download = `logs_${new Date().toISOString().slice(0, 19)}.log`;
                a.click();
                window.URL.revokeObjectURL(url);
                alert('✅ Logs exportés');
            }
        } catch (error) {
            alert('❌ Erreur export: ' + error.message);
        }
    }
    
    async clearLogs() {
        if (!confirm('⚠️ Vider tous les logs ?')) return;
        
        try {
            const response = await fetch('/api/logs/clear', { method: 'DELETE' });
            if (response.ok) {
                alert('✅ Logs vidés');
            }
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
}

// Initialiser le gestionnaire de paramètres
document.addEventListener('DOMContentLoaded', () => {
    window.settingsManager = new SettingsManager();
});