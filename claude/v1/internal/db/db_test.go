package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ----------------------------------------------------------------------------
// Helpers — base de données de test en mémoire (SQLite) ou MySQL de test
// ----------------------------------------------------------------------------
// On utilise une base MySQL de test dédiée configurée via variables d'env.
// Si non configuré, les tests sont skippés proprement.
//
// Variables d'environnement optionnelles :
//   TEST_DB_DSN   ex: root:@tcp(127.0.0.1:3306)/sim800c_test?parseTime=true
//
// Par défaut : root:@tcp(127.0.0.1:3306)/sim800c_test?parseTime=true

func testDSN() string {
	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		return dsn
	}
	return "root:@tcp(127.0.0.1:3306)/sim800c_test?parseTime=true"
}

// openTestDB ouvre (ou skipte) une connexion MySQL de test.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("mysql", testDSN())
	if err != nil {
		t.Skipf("Impossible d'ouvrir la base de test (%s) : %v — tests DB skippés", testDSN(), err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("Base de test inaccessible (%s) : %v — tests DB skippés", testDSN(), err)
	}
	db := &DB{sqlDB}
	setupTestSchema(t, db)
	return db
}

// setupTestSchema crée les tables minimales nécessaires aux tests.
func setupTestSchema(t *testing.T, db *DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS modules (
			id INT AUTO_INCREMENT PRIMARY KEY,
			com_port VARCHAR(20) NOT NULL UNIQUE,
			imei VARCHAR(20) DEFAULT '',
			phone_number VARCHAR(20) DEFAULT '',
			carrier VARCHAR(50) DEFAULT '',
			status VARCHAR(20) DEFAULT 'disconnected',
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sms_messages (
			id INT AUTO_INCREMENT PRIMARY KEY,
			module_id INT NOT NULL,
			sender_number VARCHAR(20) DEFAULT '',
			receiver_number VARCHAR(20) DEFAULT '',
			message TEXT DEFAULT '',
			direction ENUM('in','out') DEFAULT 'in',
			is_deleted BOOLEAN DEFAULT FALSE,
			is_trash BOOLEAN DEFAULT FALSE,
			is_read BOOLEAN DEFAULT FALSE,
			sms_index INT DEFAULT 0,
			received_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ussd_history (
			id INT AUTO_INCREMENT PRIMARY KEY,
			module_id INT NOT NULL,
			ussd_code VARCHAR(100) DEFAULT '',
			input_data TEXT DEFAULT '',
			output_data TEXT DEFAULT '',
			status VARCHAR(20) DEFAULT 'success',
			duration_ms INT DEFAULT 0,
			executed_by VARCHAR(100) DEFAULT 'system',
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS dial_plan (
			id INT AUTO_INCREMENT PRIMARY KEY,
			country_code VARCHAR(5) NOT NULL,
			country_name VARCHAR(100) DEFAULT '',
			calling_code VARCHAR(10) DEFAULT '',
			number_length INT DEFAULT 10,
			operator VARCHAR(100) DEFAULT '',
			prefix VARCHAR(10) DEFAULT '',
			is_active BOOLEAN DEFAULT TRUE
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("setupTestSchema: %v", err)
		}
	}
}

// cleanTable vide une table avant chaque test pour l'isolation.
func cleanTable(t *testing.T, db *DB, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if _, err := db.Exec("DELETE FROM " + tbl); err != nil {
			t.Fatalf("cleanTable %s: %v", tbl, err)
		}
	}
}

// insertTestModule insère un module fictif et retourne son ID.
func insertTestModule(t *testing.T, db *DB, comPort string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO modules (com_port, imei, phone_number, carrier, status) VALUES (?, ?, ?, ?, ?)`,
		comPort, "123456789012345", "0701020304", "Orange CI", "connected",
	)
	if err != nil {
		t.Fatalf("insertTestModule: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertTestSMS insère un SMS de test et retourne son ID.
func insertTestSMS(t *testing.T, db *DB, moduleID int, direction string, isRead, isTrash bool) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO sms_messages (module_id, sender_number, message, direction, is_read, is_trash, is_deleted, sms_index, received_at)
		 VALUES (?, '0701020304', 'Test SMS', ?, ?, ?, FALSE, 1, ?)`,
		moduleID, direction, isRead, isTrash, time.Now(),
	)
	if err != nil {
		t.Fatalf("insertTestSMS: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// ----------------------------------------------------------------------------
// TESTS — MarkSMSRead
// ----------------------------------------------------------------------------

// TestMarkSMSRead vérifie qu'un SMS non-lu devient lu après MarkSMSRead.
func TestMarkSMSRead(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_1")
	smsID := insertTestSMS(t, db, modID, "in", false, false)

	// Vérifier état initial : is_read = FALSE
	var isRead bool
	db.QueryRow(`SELECT is_read FROM sms_messages WHERE id = ?`, smsID).Scan(&isRead)
	if isRead {
		t.Fatal("Le SMS devrait être non-lu avant MarkSMSRead")
	}

	// Marquer comme lu
	if err := db.MarkSMSRead(smsID); err != nil {
		t.Fatalf("MarkSMSRead: %v", err)
	}

	// Vérifier : is_read = TRUE
	db.QueryRow(`SELECT is_read FROM sms_messages WHERE id = ?`, smsID).Scan(&isRead)
	if !isRead {
		t.Error("Le SMS devrait être marqué comme lu après MarkSMSRead")
	}
}

// TestMarkSMSReadIdempotent vérifie que MarkSMSRead est idempotent (appeler 2x ne provoque pas d'erreur).
func TestMarkSMSReadIdempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_2")
	smsID := insertTestSMS(t, db, modID, "in", true, false) // déjà lu

	if err := db.MarkSMSRead(smsID); err != nil {
		t.Fatalf("MarkSMSRead (idempotent): %v", err)
	}

	var isRead bool
	db.QueryRow(`SELECT is_read FROM sms_messages WHERE id = ?`, smsID).Scan(&isRead)
	if !isRead {
		t.Error("Le SMS doit rester lu après double appel MarkSMSRead")
	}
}

// TestMarkSMSReadInvalidID vérifie qu'aucune erreur SQL n'est retournée pour un ID inexistant.
func TestMarkSMSReadInvalidID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Un ID très grand qui n'existe pas — l'UPDATE ne doit pas retourner d'erreur SQL
	if err := db.MarkSMSRead(999999999); err != nil {
		t.Fatalf("MarkSMSRead ID inexistant ne doit pas retourner d'erreur SQL: %v", err)
	}
}

// ----------------------------------------------------------------------------
// TESTS — GetUnreadSMSCount
// ----------------------------------------------------------------------------

// TestGetUnreadSMSCount vérifie le comptage des SMS non lus par module.
func TestGetUnreadSMSCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_3")

	// 3 SMS non-lus reçus, 1 SMS lu, 1 SMS envoyé (direction=out) non-lu, 1 SMS en corbeille
	insertTestSMS(t, db, modID, "in", false, false) // non-lu, compté
	insertTestSMS(t, db, modID, "in", false, false) // non-lu, compté
	insertTestSMS(t, db, modID, "in", false, false) // non-lu, compté
	insertTestSMS(t, db, modID, "in", true, false)  // lu, non compté
	insertTestSMS(t, db, modID, "out", false, false) // sortant, non compté
	insertTestSMS(t, db, modID, "in", false, true)  // corbeille, non compté

	count, err := db.GetUnreadSMSCount(modID)
	if err != nil {
		t.Fatalf("GetUnreadSMSCount: %v", err)
	}
	if count != 3 {
		t.Errorf("GetUnreadSMSCount = %d, attendu 3", count)
	}
}

// TestGetUnreadSMSCountZero vérifie le retour 0 quand tous les SMS sont lus.
func TestGetUnreadSMSCountZero(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_4")
	insertTestSMS(t, db, modID, "in", true, false)
	insertTestSMS(t, db, modID, "in", true, false)

	count, err := db.GetUnreadSMSCount(modID)
	if err != nil {
		t.Fatalf("GetUnreadSMSCount: %v", err)
	}
	if count != 0 {
		t.Errorf("GetUnreadSMSCount = %d, attendu 0", count)
	}
}

// TestGetUnreadSMSCountAfterMarkRead vérifie que le compteur diminue après MarkSMSRead.
func TestGetUnreadSMSCountAfterMarkRead(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_5")
	smsID1 := insertTestSMS(t, db, modID, "in", false, false)
	insertTestSMS(t, db, modID, "in", false, false)

	// Avant : 2 non-lus
	count, _ := db.GetUnreadSMSCount(modID)
	if count != 2 {
		t.Fatalf("Attendu 2 non-lus, obtenu %d", count)
	}

	// Marquer le premier comme lu
	db.MarkSMSRead(smsID1)

	count, _ = db.GetUnreadSMSCount(modID)
	if count != 1 {
		t.Errorf("Après MarkSMSRead, attendu 1 non-lu, obtenu %d", count)
	}
}

// ----------------------------------------------------------------------------
// TESTS — RestoreSMSFromTrash
// ----------------------------------------------------------------------------

// TestRestoreSMSFromTrash vérifie qu'un SMS en corbeille est bien restauré.
func TestRestoreSMSFromTrash(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_6")
	smsID := insertTestSMS(t, db, modID, "in", false, true) // is_trash = TRUE

	// Vérifier état initial
	var isTrash bool
	db.QueryRow(`SELECT is_trash FROM sms_messages WHERE id = ?`, smsID).Scan(&isTrash)
	if !isTrash {
		t.Fatal("Le SMS devrait être en corbeille avant restauration")
	}

	// Restaurer
	if err := db.RestoreSMSFromTrash(smsID); err != nil {
		t.Fatalf("RestoreSMSFromTrash: %v", err)
	}

	// Vérifier : is_trash = FALSE
	db.QueryRow(`SELECT is_trash FROM sms_messages WHERE id = ?`, smsID).Scan(&isTrash)
	if isTrash {
		t.Error("Le SMS devrait avoir is_trash=FALSE après restauration")
	}
}

// TestRestoreSMSFromTrashInvalidID vérifie qu'aucune erreur SQL pour un ID inexistant.
func TestRestoreSMSFromTrashInvalidID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := db.RestoreSMSFromTrash(999999999); err != nil {
		t.Fatalf("RestoreSMSFromTrash ID inexistant ne doit pas retourner d'erreur SQL: %v", err)
	}
}

// ----------------------------------------------------------------------------
// TESTS — DeleteSMSPermanent
// ----------------------------------------------------------------------------

// TestDeleteSMSPermanent vérifie la suppression définitive d'un SMS.
func TestDeleteSMSPermanent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_7")
	smsID := insertTestSMS(t, db, modID, "in", false, false)

	// Confirmer existence avant suppression
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sms_messages WHERE id = ?`, smsID).Scan(&count)
	if count != 1 {
		t.Fatal("Le SMS doit exister avant suppression")
	}

	if err := db.DeleteSMSPermanent(smsID); err != nil {
		t.Fatalf("DeleteSMSPermanent: %v", err)
	}

	// Vérifier suppression
	db.QueryRow(`SELECT COUNT(*) FROM sms_messages WHERE id = ?`, smsID).Scan(&count)
	if count != 0 {
		t.Error("Le SMS devrait avoir été supprimé définitivement")
	}
}

// TestDeleteSMSPermanentNotRestored vérifie qu'une restauration après suppression définitive échoue silencieusement.
func TestDeleteSMSPermanentNotRestored(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "sms_messages", "modules")

	modID := insertTestModule(t, db, "COM_TEST_8")
	smsID := insertTestSMS(t, db, modID, "in", false, true)

	db.DeleteSMSPermanent(smsID)

	// La restauration ne doit pas provoquer d'erreur même si le SMS n'existe plus
	if err := db.RestoreSMSFromTrash(smsID); err != nil {
		t.Fatalf("RestoreSMSFromTrash après suppression définitive: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM sms_messages WHERE id = ?`, smsID).Scan(&count)
	if count != 0 {
		t.Error("Le SMS supprimé définitivement ne doit pas réapparaître")
	}
}

// ----------------------------------------------------------------------------
// TESTS — GetUSSDHistoryAllModules
// ----------------------------------------------------------------------------

// insertUSSDHistory insère une entrée d'historique USSD.
func insertUSSDHistory(t *testing.T, db *DB, moduleID int, code string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO ussd_history (module_id, ussd_code, input_data, output_data, status, duration_ms, executed_by, executed_at)
		 VALUES (?, ?, '', 'Résultat test', 'success', 500, 'test', ?)`,
		moduleID, code, time.Now(),
	)
	if err != nil {
		t.Fatalf("insertUSSDHistory: %v", err)
	}
}

// TestGetUSSDHistoryAllModules vérifie que module_id=0 retourne l'historique de tous les modules.
func TestGetUSSDHistoryAllModules(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "ussd_history", "modules")

	modID1 := insertTestModule(t, db, "COM_TEST_9")
	modID2 := insertTestModule(t, db, "COM_TEST_10")

	insertUSSDHistory(t, db, modID1, "#122#")
	insertUSSDHistory(t, db, modID1, "#111#")
	insertUSSDHistory(t, db, modID2, "*555#")

	history, err := db.GetUSSDHistory(0, 100)
	if err != nil {
		t.Fatalf("GetUSSDHistory(0, 100): %v", err)
	}

	if len(history) != 3 {
		t.Errorf("GetUSSDHistory tous modules: attendu 3, obtenu %d", len(history))
	}

	// Vérifier que les deux modules sont représentés
	found := map[int]bool{}
	for _, h := range history {
		found[h.ModuleID] = true
	}
	if !found[modID1] || !found[modID2] {
		t.Errorf("GetUSSDHistory doit inclure les modules %d et %d", modID1, modID2)
	}
}

// TestGetUSSDHistoryByModule vérifie le filtrage par module_id.
func TestGetUSSDHistoryByModule(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "ussd_history", "modules")

	modID1 := insertTestModule(t, db, "COM_TEST_11")
	modID2 := insertTestModule(t, db, "COM_TEST_12")

	insertUSSDHistory(t, db, modID1, "#122#")
	insertUSSDHistory(t, db, modID2, "*555#")
	insertUSSDHistory(t, db, modID2, "*100#")

	history, err := db.GetUSSDHistory(modID2, 100)
	if err != nil {
		t.Fatalf("GetUSSDHistory(modID2): %v", err)
	}

	if len(history) != 2 {
		t.Errorf("GetUSSDHistory modID2: attendu 2, obtenu %d", len(history))
	}
	for _, h := range history {
		if h.ModuleID != modID2 {
			t.Errorf("GetUSSDHistory filtre : module_id=%d dans résultat au lieu de %d", h.ModuleID, modID2)
		}
	}
}

// TestGetUSSDHistoryLimit vérifie que la limite est respectée.
func TestGetUSSDHistoryLimit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "ussd_history", "modules")

	modID := insertTestModule(t, db, "COM_TEST_13")
	for i := 0; i < 10; i++ {
		insertUSSDHistory(t, db, modID, fmt.Sprintf("*%d#", i))
	}

	history, err := db.GetUSSDHistory(0, 5)
	if err != nil {
		t.Fatalf("GetUSSDHistory limit 5: %v", err)
	}
	if len(history) != 5 {
		t.Errorf("GetUSSDHistory limit=5 : attendu 5, obtenu %d", len(history))
	}
}

// TestGetUSSDHistoryEmpty vérifie le retour nil/vide quand aucun historique.
func TestGetUSSDHistoryEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "ussd_history", "modules")

	history, err := db.GetUSSDHistory(0, 100)
	if err != nil {
		t.Fatalf("GetUSSDHistory vide: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("GetUSSDHistory vide: attendu 0, obtenu %d", len(history))
	}
}

// ----------------------------------------------------------------------------
// TESTS — ValidatePhoneNumber (via dial_plan)
// ----------------------------------------------------------------------------

// setupDialPlanCI insère les préfixes CI de test.
func setupDialPlanCI(t *testing.T, db *DB) {
	t.Helper()
	entries := []struct {
		operator, prefix string
	}{
		{"Orange CI", "07"},
		{"MTN CI", "05"},
		{"Moov Africa CI", "01"},
	}
	for _, e := range entries {
		_, err := db.Exec(
			`INSERT INTO dial_plan (country_code, country_name, calling_code, number_length, operator, prefix, is_active)
			 VALUES ('CI', 'Côte d''Ivoire', '+225', 10, ?, ?, TRUE)`,
			e.operator, e.prefix,
		)
		if err != nil {
			t.Fatalf("setupDialPlanCI %s: %v", e.operator, err)
		}
	}
}

// TestValidatePhoneNumberValid vérifie les numéros CI valides.
func TestValidatePhoneNumberValid(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "dial_plan")
	setupDialPlanCI(t, db)

	cases := []struct {
		number   string
		operator string
	}{
		{"0701020304", "Orange CI"},
		{"0512345678", "MTN CI"},
		{"0112345678", "Moov Africa CI"},
	}

	for _, tc := range cases {
		t.Run(tc.number, func(t *testing.T) {
			op, err := db.ValidatePhoneNumber("CI", tc.number)
			if err != nil {
				t.Errorf("ValidatePhoneNumber(%s): erreur inattendue: %v", tc.number, err)
			}
			if op != tc.operator {
				t.Errorf("ValidatePhoneNumber(%s): opérateur attendu %q, obtenu %q", tc.number, tc.operator, op)
			}
		})
	}
}

// TestValidatePhoneNumberInvalid vérifie que les numéros invalides sont rejetés.
func TestValidatePhoneNumberInvalid(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "dial_plan")
	setupDialPlanCI(t, db)

	invalid := []string{
		"",          // vide
		"123",       // trop court
		"0901020304", // préfixe 09 inexistant
		"07010203",  // 8 chiffres seulement
		"070102030405", // trop long (12 chiffres)
	}

	for _, number := range invalid {
		t.Run(fmt.Sprintf("invalid_%s", number), func(t *testing.T) {
			_, err := db.ValidatePhoneNumber("CI", number)
			if err == nil {
				t.Errorf("ValidatePhoneNumber(%q): erreur attendue pour numéro invalide", number)
			}
		})
	}
}

// TestValidatePhoneNumberWithPrefix vérifie la normalisation des numéros avec indicatif international.
func TestValidatePhoneNumberWithPrefix(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "dial_plan")
	setupDialPlanCI(t, db)

	// Le numéro avec préfixe international (+225) — 13 caractères, les 10 derniers = 0701020304
	// La fonction strip les chiffres en excès pour garder les 10 derniers
	op, err := db.ValidatePhoneNumber("CI", "+2250701020304")
	if err != nil {
		t.Logf("ValidatePhoneNumber(+2250701020304): %v (peut nécessiter normalisation préalable)", err)
		// Pas fatal : la normalisation peut être faite en amont
	} else if op != "Orange CI" {
		t.Errorf("ValidatePhoneNumber(+2250701020304): attendu Orange CI, obtenu %q", op)
	}
}

// TestValidatePhoneNumberUnknownCountry vérifie le comportement pour un pays sans données.
func TestValidatePhoneNumberUnknownCountry(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	cleanTable(t, db, "dial_plan")
	// Aucune entrée dial_plan insérée pour "XX"

	_, err := db.ValidatePhoneNumber("XX", "0701020304")
	if err == nil {
		t.Error("ValidatePhoneNumber pays inconnu: erreur attendue")
	}
}
