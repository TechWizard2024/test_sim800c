// Gestionnaire des commandes USSD — v1-19 (MICRO-BLOC B3)
// Nouveautés: raccourcis codes récents (5 derniers codes cliquables sous le champ USSD)
class USSDManager {
    constructor() {
        this.currentModuleId = null;
        this.favorites = [];
        this.recentCodes = [];   // MICRO-BLOC B3
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

        // MICRO-BLOC B3 — Charger les codes récents quand le module change
        const moduleSelect = document.getElementById('ussd-module');
        if (moduleSelect) {
            moduleSelect.addEventListener('change', () => {
                const id = moduleSelect.value;
                this.currentModuleId = id || null;
                if (id) {
                    this.loadRecentCodes(id);
                } else {
                    this.renderRecentCodes([]);
                }
            });
        }
    }
    
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            
            const select = document.getElementById('ussd-module');
            if (select) {
                select.innerHTML = '<option value="">Sélectionner un module</option>';
                modules.forEach(module => {
                    const id = module.module_id || module.id || module.port;
                    select.innerHTML += `<option value="${id}">${module.port || id} - ${module.phone_number || 'No SIM'}</option>`;
                });
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }

    // MICRO-BLOC B3 — Charger les 5 derniers codes USSD du module
    async loadRecentCodes(moduleId) {
        if (!moduleId) {
            this.renderRecentCodes([]);
            return;
        }
        try {
            const resp = await fetch(`/api/modules/${moduleId}/ussd/recent?limit=5`);
            if (!resp.ok) { this.renderRecentCodes([]); return; }
            const data = await resp.json();
            this.recentCodes = data.codes || [];
            this.renderRecentCodes(this.recentCodes);
        } catch (e) {
            this.renderRecentCodes([]);
        }
    }

    // MICRO-BLOC B3 — Afficher les boutons de raccourcis codes récents
    renderRecentCodes(codes) {
        const container = document.getElementById('ussd-recent-codes');
        if (!container) return;

        if (!codes || codes.length === 0) {
            container.style.display = 'none';
            container.innerHTML = '';
            return;
        }

        container.style.display = 'flex';
        container.style.flexWrap = 'wrap';
        container.style.gap = '6px';
        container.style.marginTop = '8px';
        container.style.alignItems = 'center';

        let html = '<span style="font-size:0.8rem;color:var(--text-muted,#888);white-space:nowrap;">🕐 Récents :</span>';
        codes.forEach(code => {
            const escaped = code.replace(/"/g, '&quot;');
            html += `<button class="btn-sm btn-recent-code" data-code="${escaped}" title="Cliquer pour pré-remplir le champ">${escaped}</button>`;
        });
        container.innerHTML = html;

        // Attacher événements : clic → pré-remplir le champ
        container.querySelectorAll('.btn-recent-code').forEach(btn => {
            btn.addEventListener('click', () => {
                const input = document.getElementById('ussd-code');
                if (input) {
                    input.value = btn.dataset.code;
                    input.focus();
                }
            });
        });
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
        const moduleSelect = document.getElementById('ussd-module');
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
                // Bouton copier résultat
                this.addCopyButton(outputDiv, result.result || '');
            }
            
            // MICRO-BLOC B3 — Rafraîchir les raccourcis après exécution
            this.loadRecentCodes(moduleId);
            
            // Détecter si la réponse contient un menu navigable et afficher les boutons de choix
            this.renderMenuChoices(result.result, moduleId);
            
            // Proposer d'ajouter aux favoris
            this.offerAddToFavorites(ussdCode);
            
        } catch (error) {
            if (outputDiv) {
                outputDiv.textContent = `❌ Erreur: ${error.message}`;
            }
        }
    }
    
    // Ajoute un bouton 📋 Copier à côté du résultat USSD
    addCopyButton(container, text) {
        // Supprimer ancien bouton copier s'il existe
        const existing = container.querySelector('.btn-copy-ussd');
        if (existing) existing.remove();

        if (!text) return;

        const btn = document.createElement('button');
        btn.className = 'btn-sm btn-copy-ussd';
        btn.title = 'Copier le résultat dans le presse-papiers';
        btn.innerHTML = '📋 Copier';
        btn.style.cssText = 'margin-top:8px; display:block;';
        btn.onclick = () => {
            navigator.clipboard.writeText(text).then(() => {
                btn.innerHTML = '✅ Copié!';
                setTimeout(() => { btn.innerHTML = '📋 Copier'; }, 1800);
            }).catch(() => {
                const ta = document.createElement('textarea');
                ta.value = text;
                document.body.appendChild(ta);
                ta.select();
                document.execCommand('copy');
                document.body.removeChild(ta);
                btn.innerHTML = '✅ Copié!';
                setTimeout(() => { btn.innerHTML = '📋 Copier'; }, 1800);
            });
        };
        container.appendChild(btn);
    }

    offerAddToFavorites(ussdCode) {
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
    
    /**
     * Analyse le texte de menu USSD et affiche des boutons pour chaque option.
     */
    renderMenuChoices(menuText, moduleId) {
        const container = document.getElementById('ussd-menu-choices');
        if (!container) return;
        container.innerHTML = '';

        if (this._countdownTimer) {
            clearInterval(this._countdownTimer);
            this._countdownTimer = null;
        }

        if (!menuText) return;

        const optionRe = /^\s*(\d{1,2})[:.\\-]\s*(.+)$/gm;
        const options = [];
        let match;
        const seen = new Set();
        while ((match = optionRe.exec(menuText)) !== null) {
            const num = match[1].trim();
            const label = match[2].trim();
            if (!seen.has(num) && label.length > 0) {
                seen.add(num);
                options.push({ num, label });
            }
        }

        if (options.length === 0) return;

        const countdownDiv = document.createElement('div');
        countdownDiv.className = 'menu-countdown';
        countdownDiv.id = 'menu-countdown';
        let secondsLeft = 25;
        const updateCountdown = () => {
            countdownDiv.innerHTML = `⏱ Répondez dans <span class="countdown-sec${secondsLeft <= 5 ? ' urgent' : ''}">${secondsLeft}s</span>`;
        };
        updateCountdown();
        container.appendChild(countdownDiv);

        this._countdownTimer = setInterval(() => {
            secondsLeft--;
            updateCountdown();
            if (secondsLeft <= 0) {
                clearInterval(this._countdownTimer);
                this._countdownTimer = null;
                countdownDiv.innerHTML = '⏱ Session USSD expirée — Relancez le code';
                countdownDiv.classList.add('expired');
                container.querySelectorAll('.btn-menu-choice').forEach(b => b.disabled = true);
            }
        }, 1000);

        const title = document.createElement('p');
        title.className = 'menu-choices-title';
        title.textContent = '↩ Choisir une option :';
        container.appendChild(title);

        options.forEach(opt => {
            const btn = document.createElement('button');
            btn.className = 'btn-menu-choice';
            btn.title = opt.label;
            btn.innerHTML = `<strong>${opt.num}</strong> — ${opt.label.length > 35 ? opt.label.substring(0, 35) + '…' : opt.label}`;
            btn.addEventListener('click', () => {
                if (this._countdownTimer) {
                    clearInterval(this._countdownTimer);
                    this._countdownTimer = null;
                }
                this.navigateChoice(moduleId, opt.num);
            });
            container.appendChild(btn);
        });
    }
    
    /**
     * Envoie un choix de navigation dans la session USSD en cours.
     */
    async navigateChoice(moduleId, choice) {
        const outputDiv = document.getElementById('ussd-output');
        const container = document.getElementById('ussd-menu-choices');
        
        if (outputDiv) outputDiv.textContent = `⏳ Envoi du choix "${choice}"...`;
        if (container) container.innerHTML = '';
        
        try {
            const response = await fetch(`/api/modules/${moduleId}/ussd/navigate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ choice })
            });
            
            if (!response.ok) {
                const error = await response.text();
                throw new Error(error);
            }
            
            const result = await response.json();
            
            if (outputDiv) {
                outputDiv.textContent = result.result || JSON.stringify(result, null, 2);
                this.addCopyButton(outputDiv, result.result || '');
            }
            
            this.renderMenuChoices(result.result, moduleId);
            
        } catch (error) {
            if (outputDiv) {
                outputDiv.textContent = `❌ Erreur navigation: ${error.message}`;
            }
        }
    }
}

// Initialiser le gestionnaire USSD
document.addEventListener('DOMContentLoaded', () => {
    window.ussdManager = new USSDManager();
});
