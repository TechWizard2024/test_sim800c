package excel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
)

type USSDCode struct {
	ID                int    `json:"id"`
	Carrier           string `json:"carrier"`
	Action            string `json:"action"`
	Target            string `json:"target"`
	Operation         string `json:"operation"`
	USSDCode          string `json:"ussd_code"`
	InformationINPUT  string `json:"information_input"`
	InformationOUTPUT string `json:"information_output"`
	Scope             string `json:"scope"`
	Comment           string `json:"comment"`
	ParentUSSDID      int    `json:"parent_ussd_id"`
}

type ExcelReader struct {
	basePath        string
	filenamePattern string
	logger          *logrus.Logger
	cache           map[int]USSDCode
	cacheByCode     map[string]USSDCode
	mu              sync.RWMutex
	lastLoadTime    time.Time
}

func NewExcelReader(basePath, filenamePattern string, logger *logrus.Logger) *ExcelReader {
	return &ExcelReader{
		basePath:        basePath,
		filenamePattern: filenamePattern,
		logger:          logger,
		cache:           make(map[int]USSDCode),
		cacheByCode:     make(map[string]USSDCode),
	}
}

func (r *ExcelReader) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Trouver le fichier Excel le plus récent
	filename, err := r.findLatestFile()
	if err != nil {
		return fmt.Errorf("fichier Excel non trouvé: %w", err)
	}

	filePath := filepath.Join(r.basePath, filename)
	r.logger.Infof("Chargement du fichier Excel: %s", filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture fichier: %w", err)
	}
	defer f.Close()

	// Lire la première feuille
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("aucune feuille trouvée")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return fmt.Errorf("erreur lecture lignes: %w", err)
	}

	if len(rows) < 2 {
		return fmt.Errorf("fichier vide")
	}

	// Trouver les colonnes
	headers := rows[0]
	colIndex := make(map[string]int)
	for i, header := range headers {
		colIndex[header] = i
	}

	// Vider le cache
	r.cache = make(map[int]USSDCode)
	r.cacheByCode = make(map[string]USSDCode)

	// Lire les données
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}

		code := USSDCode{}

		// Lire chaque colonne
		if idx, ok := colIndex["ID"]; ok && idx < len(row) {
			code.ID, _ = strconv.Atoi(row[idx])
		}
		if idx, ok := colIndex["Carrier"]; ok && idx < len(row) {
			code.Carrier = row[idx]
		}
		if idx, ok := colIndex["Action"]; ok && idx < len(row) {
			code.Action = row[idx]
		}
		if idx, ok := colIndex["Target"]; ok && idx < len(row) {
			code.Target = row[idx]
		}
		if idx, ok := colIndex["Operation"]; ok && idx < len(row) {
			code.Operation = row[idx]
		}
		if idx, ok := colIndex["USSD_Code"]; ok && idx < len(row) {
			code.USSDCode = row[idx]
		}
		if idx, ok := colIndex["Information_INPUT"]; ok && idx < len(row) {
			code.InformationINPUT = row[idx]
		}
		if idx, ok := colIndex["Information_OUTPUT"]; ok && idx < len(row) {
			code.InformationOUTPUT = row[idx]
		}
		if idx, ok := colIndex["Scope"]; ok && idx < len(row) {
			code.Scope = row[idx]
		}
		if idx, ok := colIndex["Comment"]; ok && idx < len(row) {
			code.Comment = row[idx]
		}
		if idx, ok := colIndex["Parent_USSD_ID"]; ok && idx < len(row) {
			code.ParentUSSDID, _ = strconv.Atoi(row[idx])
		}

		// Ne garder que les codes avec Scope = "In"
		if code.Scope == "In" && code.USSDCode != "" {
			r.cache[code.ID] = code
			r.cacheByCode[code.USSDCode] = code
		}
	}

	r.lastLoadTime = time.Now()
	r.logger.Infof("Chargé %d codes USSD depuis %s", len(r.cache), filename)

	return nil
}

func (r *ExcelReader) findLatestFile() (string, error) {
	files, err := filepath.Glob(filepath.Join(r.basePath, r.filenamePattern))
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("aucun fichier trouvé")
	}

	// Trier par date de modification
	sort.Slice(files, func(i, j int) bool {
		infoI, _ := os.Stat(files[i])
		infoJ, _ := os.Stat(files[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return filepath.Base(files[0]), nil
}

func (r *ExcelReader) GetByID(id int) (USSDCode, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	code, ok := r.cache[id]
	return code, ok
}

func (r *ExcelReader) GetByUSSDCode(ussdCode string) (bool, USSDCode) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	code, ok := r.cacheByCode[ussdCode]
	return ok, code
}

func (r *ExcelReader) GetByCarrier(carrier string) []USSDCode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []USSDCode
	for _, code := range r.cache {
		if code.Carrier == carrier {
			result = append(result, code)
		}
	}
	return result
}

func (r *ExcelReader) GetByCriteria(carrier, action, target string) []USSDCode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []USSDCode
	for _, code := range r.cache {
		if carrier == "" || code.Carrier == carrier {
			if action == "" || code.Action == action {
				if target == "" || code.Target == target {
					result = append(result, code)
				}
			}
		}
	}
	return result
}

func (r *ExcelReader) GetConsultCodes(carrier string) []USSDCode {
	return r.GetByCriteria(carrier, "Consulter", "Interne")
}

func (r *ExcelReader) GetServiceNCodes(carrier string) []USSDCode {
	return r.GetByCriteria(carrier, "Services_N1", "Interne")
}

func (r *ExcelReader) GetAllCodes() []USSDCode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]USSDCode, 0, len(r.cache))
	for _, code := range r.cache {
		result = append(result, code)
	}
	return result
}

func (r *ExcelReader) ReloadIfNeeded(maxAgeMinutes int) error {
	if time.Since(r.lastLoadTime) > time.Duration(maxAgeMinutes)*time.Minute {
		return r.Load()
	}
	return nil
}
