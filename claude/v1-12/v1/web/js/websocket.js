// Gestionnaire WebSocket pour la communication temps réel
class WebSocketManager {
    constructor() {
        this.socket = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 3000;
        this.messageHandlers = [];
        this.isConnecting = false;
    }
    
    connect() {
        if (this.isConnecting) return;
        this.isConnecting = true;
        
        const wsUrl = this.getWebSocketUrl();
        console.log(`Connexion WebSocket à ${wsUrl}`);
        
        try {
            this.socket = new WebSocket(wsUrl);
            
            this.socket.onopen = () => {
                console.log('WebSocket connecté');
                this.isConnecting = false;
                this.reconnectAttempts = 0;
                this.updateConnectionStatus(true);
            };
            
            this.socket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleMessage(data);
                } catch (e) {
                    console.error('Erreur parsing message:', e);
                }
            };
            
            this.socket.onerror = (error) => {
                console.error('Erreur WebSocket:', error);
                this.updateConnectionStatus(false);
            };
            
            this.socket.onclose = () => {
                console.log('WebSocket déconnecté');
                this.updateConnectionStatus(false);
                this.isConnecting = false;
                this.attemptReconnect();
            };
        } catch (error) {
            console.error('Erreur connexion WebSocket:', error);
            this.isConnecting = false;
            this.attemptReconnect();
        }
    }
    
    getWebSocketUrl() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.hostname;
        // Use same port as the page (avoids hardcoded mismatch)
        const port = window.location.port || '8082';
        return `${protocol}//${host}:${port}/api/ws`;
    }
    
    attemptReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Nombre maximum de tentatives de reconnexion atteint');
            return;
        }
        
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1);
        
        console.log(`Tentative de reconnexion ${this.reconnectAttempts}/${this.maxReconnectAttempts} dans ${delay}ms`);
        
        setTimeout(() => {
            this.connect();
        }, delay);
    }
    
    handleMessage(message) {
        this.messageHandlers.forEach(handler => {
            try {
                handler(message);
            } catch (e) {
                console.error('Erreur dans le handler de message:', e);
            }
        });
    }
    
    onMessage(handler) {
        this.messageHandlers.push(handler);
    }
    
    send(data) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify(data));
        } else {
            console.warn('WebSocket non connecté, message non envoyé');
        }
    }
    
    updateConnectionStatus(connected) {
        const statusDot = document.getElementById('ws-status');
        const statusText = document.getElementById('ws-status-text');
        
        if (statusDot) {
            statusDot.className = `status-dot ${connected ? 'connected' : 'disconnected'}`;
        }
        
        if (statusText) {
            statusText.textContent = connected ? 'Connecté' : 'Déconnecté';
        }
    }
    
    disconnect() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
    }
}