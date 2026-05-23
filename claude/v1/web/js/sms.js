// Gestionnaire des SMS
class SMSManager {
    constructor() {
        this.currentModuleId = null;
        this.currentTab = 'inbox';
        this.smsData = { inbox: [], trash: [] };
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.loadModules();
        
        // Rafraîchir périodiquement
        setInterval(() => this.loadSMS(), 10000);
    }
    
    setupEventListeners() {
        // Nouveau SMS
        const newSmsBtn = document.getElementById('new-sms-btn');
        if (newSmsBtn) {
            newSmsBtn.addEventListener('click', () => this.showNewSMSModal());
        }
        
        // Rafraîchir
        const refreshBtn = document.getElementById('refresh-sms-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadSMS());
        }
        
        // Filtre par module
        const moduleSelect = document.getElementById('sms-module-select');
        if (moduleSelect) {
            moduleSelect.addEventListener('change', () => this.loadSMS());
        }
        
        // Recherche
        const searchInput = document.getElementById('sms-search');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterSMS());
        }
        
        // Onglets SMS
        document.querySelectorAll('.sms-tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = btn.dataset.smsTab;
                this.switchTab(tab);
            });
        });
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            const select = document.getElementById('sms-module-select');
            if (select) {
                let options = '<option value="all">Tous les modules</option>';
                modules.forEach(module => {
                    const id = module.module_id || module.port;
                    options += `<option value="${id}">${module.port || id} - ${module.phone_number || 'No SIM'}</option>`;
                });
                select.innerHTML = options;
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    async loadSMS() {
        const moduleSelect = document.getElementById('sms-module-select');
        const moduleId = moduleSelect?.value || 'all';
        
        try {
            // Charger les SMS pour chaque module ou tous
            if (moduleId === 'all') {
                const response = await fetch('/api/modules');
                const modules = await response.json();
                
                this.smsData = { inbox: [], trash: [] };
                
                for (const module of modules) {
                    const id = module.module_id || module.port;
                    const smsResponse = await fetch(`/api/modules/${id}/sms?include_trash=true`);
                    const smsList = await smsResponse.json();
                    
                    smsList.forEach(sms => {
                        if (sms.is_trash) {
                            this.smsData.trash.push({ ...sms, module_id: id });
                        } else if (!sms.is_deleted) {
                            this.smsData.inbox.push({ ...sms, module_id: id });
                        }
                    });
                }
            } else {
                const response = await fetch(`/api/modules/${moduleId}/sms?include_trash=true`);
                const smsList = await response.json();
                
                this.smsData = { inbox: [], trash: [] };
                smsList.forEach(sms => {
                    if (sms.is_trash) {
                        this.smsData.trash.push(sms);
                    } else if (!sms.is_deleted) {
                        this.smsData.inbox.push(sms);
                    }
                });
            }
            
            this.updateCounts();
            this.render();
        } catch (error) {
            console.error('Erreur chargement SMS:', error);
        }
    }
    
    updateCounts() {
        const inboxCount = document.getElementById('inbox-count');
        const trashCount = document.getElementById('trash-count');
        
        if (inboxCount) inboxCount.textContent = this.smsData.inbox.length;
        if (trashCount) trashCount.textContent = this.smsData.trash.length;
    }
    
    render() {
        const currentData = this.currentTab === 'inbox' ? this.smsData.inbox : this.smsData.trash;
        const container = document.getElementById(this.currentTab === 'inbox' ? 'sms-inbox' : 'sms-trash');
        
        if (!container) return;
        
        if (currentData.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>Aucun message</p></div>';
            return;
        }
        
        let html = '';
        for (const sms of currentData) {
            html += `
                <div class="sms-item" data-sms-id="${sms.id}">
                    <div class="sms-header">
                        <div class="sms-sender">
                            <strong>${this.currentTab === 'inbox' ? '📩 ' + (sms.sender_number || 'Inconnu') : '🗑️ ' + (sms.sender_number || 'Inconnu')}</strong>
                            ${sms.module_id ? `<span class="sms-module">Module ${sms.module_id}</span>` : ''}
                        </div>
                        <div class="sms-date">${new Date(sms.received_at).toLocaleString()}</div>
                    </div>
                    <div class="sms-content">${this.escapeHtml(sms.message)}</div>
                    <div class="sms-actions">
                        ${this.currentTab === 'trash' ? 
                            `<button class="btn-sm btn-restore" data-id="${sms.id}">↩️ Restaurer</button>
                             <button class="btn-sm btn-delete-permanent" data-id="${sms.id}">🗑️ Supprimer définitivement</button>` :
                            `<button class="btn-sm btn-reply" data-number="${sms.sender_number}">↩️ Répondre</button>
                             <button class="btn-sm btn-trash" data-id="${sms.id}">📂 Corbeille</button>
                             <button class="btn-sm btn-delete" data-id="${sms.id}" data-index="${sms.sms_index}">❌ Supprimer</button>`
                        }
                    </div>
                </div>
            `;
        }
        
        container.innerHTML = html;
        this.attachSMSEvents();
    }
    
    attachSMSEvents() {
        if (this.currentTab === 'inbox') {
            document.querySelectorAll('.btn-reply').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const number = btn.dataset.number;
                    this.showReplyModal(number);
                });
            });
            
            document.querySelectorAll('.btn-trash').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const id = btn.dataset.id;
                    this.moveToTrash(id);
                });
            });
            
            document.querySelectorAll('.btn-delete').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const id = btn.dataset.id;
                    const index = btn.dataset.index;
                    this.deleteSMS(id, index);
                });
            });
        } else {
            document.querySelectorAll('.btn-restore').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const id = btn.dataset.id;
                    this.restoreFromTrash(id);
                });
            });
            
            document.querySelectorAll('.btn-delete-permanent').forEach(btn => {
                btn.addEventListener('click', (e) => {
                    const id = btn.dataset.id;
                    this.deletePermanent(id);
                });
            });
        }
    }
    
    filterSMS() {
        const searchTerm = document.getElementById('sms-search')?.value.toLowerCase() || '';
        const currentData = this.currentTab === 'inbox' ? this.smsData.inbox : this.smsData.trash;
        
        const filtered = currentData.filter(sms => 
            sms.message.toLowerCase().includes(searchTerm) ||
            (sms.sender_number && sms.sender_number.includes(searchTerm))
        );
        
        const container = document.getElementById(this.currentTab === 'inbox' ? 'sms-inbox' : 'sms-trash');
        if (!container) return;
        
        if (filtered.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>Aucun message trouvé</p></div>';
            return;
        }
        
        let html = '';
        for (const sms of filtered) {
            html += `
                <div class="sms-item">
                    <div class="sms-header">
                        <div class="sms-sender"><strong>${sms.sender_number || 'Inconnu'}</strong></div>
                        <div class="sms-date">${new Date(sms.received_at).toLocaleString()}</div>
                    </div>
                    <div class="sms-content">${this.escapeHtml(sms.message)}</div>
                </div>
            `;
        }
        container.innerHTML = html;
    }
    
    switchTab(tab) {
        this.currentTab = tab;
        
        // Mettre à jour les onglets
        document.querySelectorAll('.sms-tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.smsTab === tab);
        });
        
        // Mettre à jour l'affichage
        const inboxDiv = document.getElementById('sms-inbox');
        const trashDiv = document.getElementById('sms-trash');
        
        if (inboxDiv) inboxDiv.style.display = tab === 'inbox' ? 'block' : 'none';
        if (trashDiv) trashDiv.style.display = tab === 'trash' ? 'block' : 'none';
        
        this.render();
    }
    
    showNewSMSModal() {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>✏️ Nouveau SMS</h2>
                <form id="new-sms-form">
                    <label>Module:</label>
                    <select id="modal-sms-module" required></select>
                    
                    <label>Numéro destinataire:</label>
                    <input type="tel" id="modal-sms-number" placeholder="0701010101" required>
                    
                    <label>Message:</label>
                    <textarea id="modal-sms-message" rows="5" required></textarea>
                    
                    <button type="submit" class="btn-primary">📨 Envoyer</button>
                </form>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Charger les modules
        this.loadModulesIntoSelect('modal-sms-module');
        
        const closeBtn = modal.querySelector('.close');
        closeBtn.onclick = () => modal.remove();
        
        const form = modal.querySelector('#new-sms-form');
        form.onsubmit = async (e) => {
            e.preventDefault();
            const moduleId = document.getElementById('modal-sms-module').value;
            const number = document.getElementById('modal-sms-number').value;
            const message = document.getElementById('modal-sms-message').value;
            
            await this.sendSMS(moduleId, number, message);
            modal.remove();
            this.loadSMS();
        };
    }
    
    showReplyModal(number) {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.style.display = 'block';
        modal.innerHTML = `
            <div class="modal-content">
                <span class="close">&times;</span>
                <h2>↩️ Répondre à ${number}</h2>
                <form id="reply-sms-form">
                    <label>Message:</label>
                    <textarea id="reply-sms-message" rows="5" required></textarea>
                    
                    <button type="submit" class="btn-primary">📨 Envoyer</button>
                </form>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        const closeBtn = modal.querySelector('.close');
        closeBtn.onclick = () => modal.remove();
        
        const form = modal.querySelector('#reply-sms-form');
        form.onsubmit = async (e) => {
            e.preventDefault();
            const moduleSelect = document.getElementById('sms-module-select');
            const moduleId = moduleSelect?.value;
            const message = document.getElementById('reply-sms-message').value;
            
            if (!moduleId || moduleId === 'all') {
                alert('Veuillez sélectionner un module spécifique');
                return;
            }
            
            await this.sendSMS(moduleId, number, message);
            modal.remove();
            this.loadSMS();
        };
    }
    
    async loadModulesIntoSelect(selectId) {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            const select = document.getElementById(selectId);
            if (select) {
                select.innerHTML = '';
                modules.forEach(module => {
                    const id = module.module_id || module.port;
                    const option = document.createElement('option');
                    option.value = id;
                    option.textContent = `${module.port || id} - ${module.phone_number || 'No SIM'}`;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    async sendSMS(moduleId, number, message) {
        try {
            const response = await fetch(`/api/modules/${moduleId}/sms/send`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ number, message })
            });
            
            if (response.ok) {
                alert('✅ SMS envoyé avec succès');
            } else {
                const error = await response.text();
                alert('❌ Erreur: ' + error);
            }
        } catch (error) {
            alert('❌ Erreur: ' + error.message);
        }
    }
    
    async moveToTrash(smsId) {
        try {
            const response = await fetch(`/api/sms/trash/${smsId}`, { method: 'POST' });
            if (response.ok) {
                this.loadSMS();
            }
        } catch (error) {
            console.error('Erreur déplacement vers corbeille:', error);
        }
    }
    
    async deleteSMS(smsId, smsIndex) {
        if (!confirm('Supprimer définitivement ce SMS ?')) return;
        
        const moduleSelect = document.getElementById('sms-module-select');
        const moduleId = moduleSelect?.value;
        
        if (!moduleId || moduleId === 'all') return;
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/sms/${smsIndex}`, { method: 'DELETE' });
            if (response.ok) {
                this.loadSMS();
            }
        } catch (error) {
            console.error('Erreur suppression SMS:', error);
        }
    }
    
    async restoreFromTrash(smsId) {
        // Implémenter la restauration
        this.loadSMS();
    }
    
    async deletePermanent(smsId) {
        if (!confirm('Supprimer définitivement ce SMS ?')) return;
        // Implémenter la suppression définitive
        this.loadSMS();
    }
    
    addSMS(sms) {
        if (sms.is_trash) {
            this.smsData.trash.unshift(sms);
        } else {
            this.smsData.inbox.unshift(sms);
        }
        this.updateCounts();
        this.render();
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialiser le gestionnaire SMS
document.addEventListener('DOMContentLoaded', () => {
    window.smsManager = new SMSManager();
});