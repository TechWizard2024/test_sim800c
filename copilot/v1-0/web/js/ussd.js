// Gestionnaire des commandes USSD
class USSDManager {
    constructor() {
        this.currentModuleId = null;
        this.favorites = [];
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.loadModules();
        this.loadFavorites();
    }
    
    setupEventListeners() {
        const executeBtn = document.getElementById('execute-ussd-btn');
        if (executeBtn) {
            executeBtn.addEventListener('click', () => this.executeUSSD());
        }
        
        const ussdCodeInput = document.getElementById('ussd-code');
        if (ussdCodeInput) {
            ussdCodeInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') this.executeUSSD();
            });
        }
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            const select = document.getElementById('ussd-module-select');
            if (select) {
                select.innerHTML = '<option value="">Sélectionner un module</option>';
                modules.forEach(module => {
                    const id = module.module_id || module.port;
                    select.innerHTML += `<option value="${id}">${module.port || id} - ${module.phone_number || 'No SIM'}</option>`;
                });
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }
    
    async loadFavorites() {
        try {
            const response = await fetch('/api/ussd/favorites');
            if (response.ok) {
                this.favorites = await response.json();
                this.renderFavorites();
            }
        } catch (error) {
            console.error('Erreur chargement favoris:', error);
        }
    }
    
    renderFavorites() {
        const container = document.getElementById('favorites-list');
        if (!container) return;
        
        if (this.favorites.length === 0) {
            container.innerHTML = '<p class="empty-favorites">Aucun favori. Ajoutez vos codes USSD préférés.</p>';
            return;
        }
        
        let html = '';
        for (const fav of this.favorites) {
            html += `
                <div class="favorite-item" data-code="${fav.ussd_code}" data-carrier="${fav.carrier}">
                    <span class="fav-code">${fav.ussd_code}</span>
                    <span class="fav-name">${fav.operation || ''}</span>
                    <button class="fav-use" data-code="${fav.ussd_code}">▶</button>
                    <button class="fav-remove" data-id="${fav.id}">✕</button>
                </div>
            `;
        }
        
        container.innerHTML = html;
        
        // Attacher événements
        document.querySelectorAll('.fav-use').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const code = btn.dataset.code;
                document.getElementById('ussd-code').value = code;
                this.executeUSSD();
            });
        });
    }
    
    async executeUSSD() {
        const moduleSelect = document.getElementById('ussd-module-select');
        const moduleId = moduleSelect?.value;
        const ussdCode = document.getElementById('ussd-code')?.value;
        const inputData = document.getElementById('ussd-input-data')?.value || '';
        const outputDiv = document.getElementById('ussd-output');
        
        if (!moduleId) {
            alert('Veuillez sélectionner un module');
            return;
        }
        
        if (!ussdCode) {
            alert('Veuillez entrer un code USSD');
            return;
        }
        
        if (outputDiv) {
            outputDiv.textContent = '⏳ Exécution en cours...';
        }
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/ussd/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    module_id: parseInt(moduleId), 
                    ussd_code: ussdCode,
                    input_data: inputData
                })
            });
            
            if (!response.ok) {
                const error = await response.text();
                throw new Error(error);
            }
            
            const result = await response.json();
            
            if (outputDiv) {
                outputDiv.textContent = result.result || JSON.stringify(result, null, 2);
            }
            
            // Proposer d'ajouter aux favoris
            this.offerAddToFavorites(ussdCode);
            
        } catch (error) {
            if (outputDiv) {
                outputDiv.textContent = `❌ Erreur: ${error.message}`;
            }
        }
    }
    
    offerAddToFavorites(ussdCode) {
        // Vérifier si déjà en favori
        const exists = this.favorites.some(f => f.ussd_code === ussdCode);
        if (!exists) {
            const addBtn = document.createElement('button');
            addBtn.textContent = '⭐ Ajouter aux favoris';
            addBtn.className = 'btn-sm';
            addBtn.onclick = () => this.addToFavorites(ussdCode);
            
            const outputDiv = document.getElementById('ussd-output');
            if (outputDiv && !outputDiv.querySelector('.add-favorite')) {
                const existing = outputDiv.querySelector('.add-favorite');
                if (existing) existing.remove();
                outputDiv.appendChild(addBtn);
            }
        }
    }
    
    async addToFavorites(ussdCode) {
        try {
            const response = await fetch('/api/ussd/favorites', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ussd_code: ussdCode })
            });
            
            if (response.ok) {
                this.loadFavorites();
                alert('✅ Ajouté aux favoris');
            }
        } catch (error) {
            console.error('Erreur ajout favori:', error);
        }
    }
}

// Initialiser le gestionnaire USSD
document.addEventListener('DOMContentLoaded', () => {
    window.ussdManager = new USSDManager();
});