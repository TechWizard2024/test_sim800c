package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sim800c-supervisor/internal/config"
	"sim800c-supervisor/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type AuthManager struct {
	db        *db.DB
	cfg       *config.Config
	logger    *logrus.Logger
	jwtSecret []byte
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthManager(dbConn *db.DB, cfg *config.Config, logger *logrus.Logger) *AuthManager {
	return &AuthManager{
		db:        dbConn,
		cfg:       cfg,
		logger:    logger,
		jwtSecret: []byte(cfg.Security.JWTSecret),
	}
}

func (a *AuthManager) CreateDefaultAdmin() {
	// Vérifier si l'admin existe déjà
	exists, _ := a.db.UserExists("admin")
	if exists {
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), a.cfg.Security.BcryptCost)

	user := &db.User{
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		CreatedAt:    time.Now(),
	}

	if err := a.db.CreateUser(user); err != nil {
		a.logger.Warnf("Erreur création admin: %v", err)
	} else {
		a.logger.Info("Utilisateur admin créé (mot de passe: admin123)")
	}
}

func (a *AuthManager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// Vérifier les identifiants
	user, err := a.db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
		return
	}

	// Générer le token JWT
	token, err := a.generateToken(user)
	if err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	// Journaliser la connexion
	a.db.SaveAuditLog(user.ID, "login", "user", user.ID, map[string]string{"ip": r.RemoteAddr}, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (a *AuthManager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Pour JWT, le logout est géré côté client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "déconnecté"})
}

func (a *AuthManager) generateToken(user *db.User) (string, error) {
	expirationTime := time.Now().Add(time.Duration(a.cfg.Security.JWTExpirationHours) * time.Hour)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *AuthManager) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.cfg.Security.EnableAuth {
			next(w, r)
			return
		}

		// Extraire le token du header
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token manquant", http.StatusUnauthorized)
			return
		}

		// Enlever le préfixe "Bearer "
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// Vérifier le token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return a.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Token invalide", http.StatusUnauthorized)
			return
		}

		// Ajouter les infos utilisateur dans le contexte
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-User-Role", claims.Role)

		next(w, r)
	}
}

func (a *AuthManager) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	user, err := a.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

func (a *AuthManager) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	// Vérifier l'ancien mot de passe
	user, err := a.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		http.Error(w, "Ancien mot de passe incorrect", http.StatusUnauthorized)
		return
	}

	// Changer le mot de passe
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), a.cfg.Security.BcryptCost)
	if err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	if err := a.db.UpdateUserPassword(userID, string(newHash)); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	a.db.SaveAuditLog(userID, "change_password", "user", userID, nil, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "Mot de passe modifié"})
}

func generateSessionToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ValidateToken - Valide un token JWT et retourne les claims
func (a *AuthManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("token invalide")
	}

	return claims, nil
}
