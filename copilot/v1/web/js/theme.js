// Gestion du thème clair/sombre
class ThemeManager {
    constructor() {
        this.init();
    }
    
    init() {
        // Charger le thème sauvegardé
        const savedTheme = localStorage.getItem('theme') || 'light';
        this.setTheme(savedTheme);
        
        // Écouter les changements de thème
        window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
            if (!localStorage.getItem('theme')) {
                this.setTheme(e.matches ? 'dark' : 'light');
            }
        });
    }
    
    setTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        
        const lightSheet = document.getElementById('theme-light');
        const darkSheet = document.getElementById('theme-dark');
        
        if (lightSheet && darkSheet) {
            if (theme === 'dark') {
                lightSheet.disabled = true;
                darkSheet.disabled = false;
            } else {
                lightSheet.disabled = false;
                darkSheet.disabled = true;
            }
        }
        
        localStorage.setItem('theme', theme);
    }
    
    toggle() {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        this.setTheme(newTheme);
    }
    
    getCurrentTheme() {
        return document.documentElement.getAttribute('data-theme') || 'light';
    }
}

// Initialiser le gestionnaire de thème
const themeManager = new ThemeManager();

// Exporter pour utilisation globale
window.themeManager = themeManager;