// Gestionnaire d'historique USSD — v1-18 (MICRO-BLOC B2)
// Nouveautés: dropdown "Tous les modules", chargement initial sans filtre module
class HistoryManager {
    constructor() {
        this.history = [];
        this.filtered = [];
        this.currentPage = 1;
        this.pageSize = 50;
        this.searchTerm = '';
        this.selectedModuleId = 'all'; // MICRO-BLOC B2 : défaut = tous
        this.init();
    }

    init() {
        this.setupEventListeners();
        // MICRO-BLOC B2 : charger les modules avant l'historique
        this.loadModules().then(() => this.loadHistory());
        setInterval(() => this.loadHistory(), 30000);
    }

    setupEventListeners() {
        const moduleSelect = document.getElementById('history-module-select');
        if (moduleSelect) {
            moduleSelect.addEventListener('change', () => {
                this.selectedModuleId = moduleSelect.value || 'all';
                this.currentPage = 1;
                this.loadHistory();
            });
        }

        const dateInput = document.getElementById('history-date');
        if (dateInput) dateInput.addEventListener('change', () => { this.currentPage = 1; this.loadHistory(); });

        const clearBtn = document.getElementById('clear-history-btn');
        if (clearBtn) clearBtn.addEventListener('click', () => this.clearHistory());

        const exportBtn = document.getElementById('export-history-csv-btn');
        if (exportBtn) exportBtn.addEventListener('click', () => this.exportCSV());

        // Filtre/recherche
        const searchInput = document.getElementById('history-search');
        if (searchInput) {
            searchInput.addEventListener('input', () => {
                this.searchTerm = searchInput.value.toLowerCase().trim();
                this.currentPage = 1;
                this.applyFilterAndRender();
            });
        }

        // Filtre par statut
        const statusFilter = document.getElementById('history-status-filter');
        if (statusFilter) {
            statusFilter.addEventListener('change', () => {
                this.currentPage = 1;
                this.applyFilterAndRender();
            });
        }
    }

    // MICRO-BLOC B2 — Charge les modules et peuple le dropdown avec "Tous les modules" en premier
    async loadModules() {
        try {
            const response = await fetch('/api/modules');
            const modules = await response.json();
            const select = document.getElementById('history-module-select');
            if (select) {
                // Option "Tous les modules" toujours en premier (envoie module_id absent = tous)
                select.innerHTML = '<option value="all">🌐 Tous les modules</option>';
                modules.forEach(module => {
                    const id = module.db_id || module.module_id || module.id || module.port;
                    const label = `${module.port || id}${module.phone_number ? ' — ' + module.phone_number : ''}${module.carrier ? ' (' + module.carrier + ')' : ''}`;
                    select.innerHTML += `<option value="${id}">${label}</option>`;
                });
                // Restaurer la sélection courante
                if (this.selectedModuleId && this.selectedModuleId !== 'all') {
                    select.value = this.selectedModuleId;
                }
            }
        } catch (error) {
            console.error('Erreur chargement modules:', error);
        }
    }

    async loadHistory() {
        const moduleSelect = document.getElementById('history-module-select');
        const moduleId = moduleSelect?.value || 'all';
        this.selectedModuleId = moduleId;
        const date = document.getElementById('history-date')?.value || '';

        try {
            // MICRO-BLOC B2 : si moduleId == 'all' → pas de paramètre module_id → backend retourne tous
            let url = '/api/ussd/history?limit=2000';
            if (moduleId !== 'all') url += `&module_id=${moduleId}`;
            if (date) url += `&date=${date}`;

            const response = await fetch(url);
            const data = await response.json();
            this.history = Array.isArray(data) ? data : [];
            this.applyFilterAndRender();
        } catch (error) {
            console.error('Erreur chargement historique:', error);
        }
    }

    applyFilterAndRender() {
        const statusFilter = document.getElementById('history-status-filter')?.value || 'all';

        this.filtered = this.history.filter(item => {
            // Filtre par statut
            if (statusFilter !== 'all' && item.status !== statusFilter) return false;
            // Filtre par recherche texte
            if (this.searchTerm) {
                const haystack = [
                    item.ussd_code || '',
                    item.output_data || '',
                    item.operation || '',
                    String(item.module_id || '')
                ].join(' ').toLowerCase();
                if (!haystack.includes(this.searchTerm)) return false;
            }
            return true;
        });

        this.updateFilterStats();
        this.render();
    }

    updateFilterStats() {
        const statsEl = document.getElementById('history-filter-stats');
        if (statsEl) {
            const total = this.history.length;
            const shown = this.filtered.length;
            const moduleLabel = this.selectedModuleId === 'all' ? 'tous modules' : `module ${this.selectedModuleId}`;
            statsEl.textContent = shown === total
                ? `${total} entrée(s) — ${moduleLabel}`
                : `${shown} / ${total} entrée(s) — ${moduleLabel}`;
        }
    }

    render() {
        const container = document.getElementById('history-list');
        if (!container) return;

        if (this.filtered.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>Aucun historique trouvé</p></div>';
            this.renderPagination(container);
            return;
        }

        // Pagination
        const totalPages = Math.ceil(this.filtered.length / this.pageSize);
        if (this.currentPage > totalPages) this.currentPage = totalPages;
        const startIdx = (this.currentPage - 1) * this.pageSize;
        const pageItems = this.filtered.slice(startIdx, startIdx + this.pageSize);

        // MICRO-BLOC B2 : afficher la colonne Module si tous les modules
        const showModuleCol = this.selectedModuleId === 'all';

        let html = '<table class="history-table"><thead><tr>';
        html += '<th>Date</th>';
        if (showModuleCol) html += '<th>Module</th>';
        html += '<th>Code USSD</th><th>Résultat</th><th>Durée</th><th>Statut</th><th>Actions</th>';
        html += '</tr></thead><tbody>';

        for (const item of pageItems) {
            const statusClass = item.status === 'success' ? 'status-success' : 'status-error';
            const preview = this.truncate(item.output_data, 60);
            const fullResult = this.escapeHtml(item.output_data || '');
            html += `<tr>`;
            html += `<td>${new Date(item.executed_at).toLocaleString()}</td>`;
            if (showModuleCol) html += `<td><span class="module-badge">M${item.module_id}</span></td>`;
            html += `
                    <td><code>${item.ussd_code}</code></td>
                    <td class="history-output" title="${fullResult}">${preview}</td>
                    <td>${item.duration_ms}ms</td>
                    <td><span class="status-badge ${statusClass}">${item.status}</span></td>
                    <td>
                        <button class="btn-sm btn-copy-result" data-result="${fullResult}" title="📋 Copier le résultat">📋</button>
                    </td>
                </tr>`;
        }

        html += '</tbody></table>';

        // Pagination controls
        html += this.buildPaginationHTML(totalPages);

        container.innerHTML = html;

        // Attach copy buttons
        container.querySelectorAll('.btn-copy-result').forEach(btn => {
            btn.addEventListener('click', () => {
                const text = btn.dataset.result || '';
                navigator.clipboard.writeText(text).then(() => {
                    btn.textContent = '✅';
                    setTimeout(() => { btn.textContent = '📋'; }, 1500);
                }).catch(() => {
                    const ta = document.createElement('textarea');
                    ta.value = text;
                    document.body.appendChild(ta);
                    ta.select();
                    document.execCommand('copy');
                    document.body.removeChild(ta);
                    btn.textContent = '✅';
                    setTimeout(() => { btn.textContent = '📋'; }, 1500);
                });
            });
        });

        // Attach pagination buttons
        container.querySelectorAll('.pagination-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const page = parseInt(btn.dataset.page);
                if (!isNaN(page)) {
                    this.currentPage = page;
                    this.render();
                }
            });
        });
    }

    buildPaginationHTML(totalPages) {
        if (totalPages <= 1) return '';

        const cur = this.currentPage;
        let html = '<div class="pagination-controls">';
        html += `<span class="pagination-info">Page ${cur} / ${totalPages}</span>`;

        // Previous
        html += `<button class="btn-sm pagination-btn" data-page="${cur - 1}" ${cur <= 1 ? 'disabled' : ''}>◀ Préc.</button>`;

        // Page numbers (show max 7 pages around current)
        const start = Math.max(1, cur - 3);
        const end = Math.min(totalPages, cur + 3);

        if (start > 1) html += `<button class="btn-sm pagination-btn" data-page="1">1</button>`;
        if (start > 2) html += `<span class="pagination-ellipsis">…</span>`;

        for (let p = start; p <= end; p++) {
            html += `<button class="btn-sm pagination-btn ${p === cur ? 'active' : ''}" data-page="${p}">${p}</button>`;
        }

        if (end < totalPages - 1) html += `<span class="pagination-ellipsis">…</span>`;
        if (end < totalPages) html += `<button class="btn-sm pagination-btn" data-page="${totalPages}">${totalPages}</button>`;

        // Next
        html += `<button class="btn-sm pagination-btn" data-page="${cur + 1}" ${cur >= totalPages ? 'disabled' : ''}>Suiv. ▶</button>`;

        html += '</div>';
        return html;
    }

    renderPagination(container) {
        // No-op for empty state (already handled in render)
    }

    async clearHistory() {
        if (!confirm('⚠️ Vider tout l\'historique ? Cette action est irréversible.')) return;
        try {
            const response = await fetch('/api/ussd/history', { method: 'DELETE' });
            if (response.ok) {
                this.history = [];
                this.filtered = [];
                this.currentPage = 1;
                this.applyFilterAndRender();
            }
        } catch (error) {
            console.error('Erreur vidage historique:', error);
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

    exportCSV() {
        const moduleId = this.selectedModuleId || 'all';
        let url = '/api/ussd/history/export?limit=5000';
        if (moduleId !== 'all') url += `&module_id=${moduleId}`;
        const a = document.createElement('a');
        a.href = url;
        a.download = '';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.historyManager = new HistoryManager();
});
