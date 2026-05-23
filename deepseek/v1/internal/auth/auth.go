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
		return
	}
	a.logger.Info("Utilisateur admin créé (mot de passe: admin123)")
}

func (a *AuthManager) LoginHandler(w http.ResponseWriter, r *http.Request) {
	a.logger.Infof("Login handler hit method=%s path=%s content-type=%s", r.Method, r.URL.Path, r.Header.Get("Content-Type"))

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.logger.Warnf("Login: JSON decode erreur: %v", err)
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}
	a.logger.Infof("Login: username=%s from=%s", req.Username, r.RemoteAddr)


	user, err := a.db.GetUserByUsername(req.Username)
	if err != nil {
		a.logger.Warnf("Login: GetUserByUsername erreur: %v", err)
		http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
		return
	}


	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		a.logger.Warnf("Login: bcrypt compare échoué pour username=%s: %v", req.Username, err)
		http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
		return
	}


	token, err := a.generateToken(user)
	if err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	// target_id est un int dans la DB (schema actuel). On log avec 0.
	_ = a.db.SaveAuditLog(user.ID, "login", "user", 0, map[string]string{"ip": r.RemoteAddr}, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
	})
	a.logger.Infof("Login: success username=%s user_id=%s role=%s", user.Username, user.ID, user.Role)

}

func (a *AuthManager) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "déconnecté"})
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

// AuthMiddlewareMux est compatible avec router.Use(...) de Gorilla/mux.
func (a *AuthManager) AuthMiddlewareMux(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.cfg.Security.EnableAuth {
			next.ServeHTTP(w, r)
			return
		}

		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token manquant", http.StatusUnauthorized)
			return
		}

		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return a.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token invalide", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-User-Role", claims.Role)

		next.ServeHTTP(w, r)
	})
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	user, err := a.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		http.Error(w, "Ancien mot de passe incorrect", http.StatusUnauthorized)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), a.cfg.Security.BcryptCost)
	if err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	if err := a.db.UpdateUserPassword(userID, string(newHash)); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	_ = a.db.SaveAuditLog(userID, "change_password", "user", 0, nil, r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "Mot de passe modifié"})
}

func generateSessionToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

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
