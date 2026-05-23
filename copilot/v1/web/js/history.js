// Gestionnaire d'historique
class HistoryManager {
    constructor() {
        this.history = [];
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.loadHistory();
        
        // Rafraîchir périodiquement
        setInterval(() => this.loadHistory(), 30000);
    }
    
    setupEventListeners() {
        const moduleSelect = document.getElementById('history-module-select');
        if (moduleSelect) {
            moduleSelect.addEventListener('change', () => this.loadHistory());
        }
        
        const dateInput = document.getElementById('history-date');
        if (dateInput) {
            dateInput.addEventListener('change', () => this.loadHistory());
        }
        
        const clearBtn = document.getElementById('clear-history-btn');
        if (clearBtn) {
            clearBtn.addEventListener('click', () => this.clearHistory());
        }
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            const select = document.getElementById('history-module-select');
            if (select) {
                select.innerHTML = '<option value="all">Tous les modules</option>';
                modules.forEach(module => {
                    const id = module.module_id || module.port;
                    select.innerHTML += `<option value="${id}">${module.port || id}</option>`;
                });
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    async loadHistory() {
        const moduleSelect = document.getElementById('history-module-select');
        const moduleId = moduleSelect?.value || 'all';
        const date = document.getElementById('history-date')?.value || '';
        
        try {
            let url = '/api/ussd/history';
            if (moduleId !== 'all') {
                url += `?module_id=${moduleId}`;
            }
            if (date) {
                url += `${moduleId !== 'all' ? '&' : '?'}date=${date}`;
            }
            
            const response = await fetch(url);
            this.history = await response.json();
            this.render();
        } catch (error) {
            console.error('Erreur chargement historique:', error);
        }
    }
    
    render() {
        const container = document.getElementById('history-list');
        if (!container) return;
        
        if (this.history.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>Aucun historique</p></div>';
            return;
        }
        
        let html = '<table class="history-table"><thead><tr>';
        html += '<th>Date</th><th>Module</th><th>Code USSD</th><th>Résultat</th><th>Durée</th><th>Statut</th>';
        html += '</tr></thead><tbody>';
        
        for (const item of this.history) {
            const statusClass = item.status === 'success' ? 'status-success' : 'status-error';
            html += `
                <tr>
                    <td>${new Date(item.executed_at).toLocaleString()}</td>
                    <td>Module ${item.module_id}</td>
                    <td><code>${item.ussd_code}</code></td>
                    <td class="history-output" title="${this.escapeHtml(item.output_data || '')}">
                        ${this.truncate(item.output_data, 50)}
                    </td>
                    <td>${item.duration_ms}ms</td>
                    <td><span class="status-badge ${statusClass}">${item.status}</span></td>
                </tr>
            `;
        }
        
        html += '</tbody></table>';
        container.innerHTML = html;
    }
    
    async clearHistory() {
        if (!confirm('⚠️ Vider tout l\'historique ? Cette action est irréversible.')) return;
        
        try {
            const response = await fetch('/api/ussd/history', { method: 'DELETE' });
            if (response.ok) {
                this.history = [];
                this.render();
                alert('✅ Historique vidé');
            }
        } catch (error) {
            console.error('Erreur vidage historique:', error);
            alert('❌ Erreur lors du vidage');
        }
    }
    
    truncate(text, maxLength) {
        if (!text) return '';
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength) + '...';
    }
    
    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialiser le gestionnaire d'historique
document.addEventListener('DOMContentLoaded', () => {
    window.historyManager = new HistoryManager();
});