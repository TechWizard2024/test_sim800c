package excel

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
)

type ExcelWriter struct {
	basePath string
	logger   *logrus.Logger
}

func NewExcelWriter(basePath string, logger *logrus.Logger) *ExcelWriter {
	return &ExcelWriter{
		basePath: basePath,
		logger:   logger,
	}
}

func (w *ExcelWriter) CreateNewVersion(newCodes []USSDCode) (string, error) {
	if len(newCodes) == 0 {
		return "", nil
	}

	w.logger.Infof("Création nouvelle version Excel avec %d nouveaux codes", len(newCodes))

	// Trouver le fichier existant le plus récent
	reader := NewExcelReader(w.basePath, "Codes_USSD_CI*.xlsx", w.logger)
	latestFile, err := reader.findLatestFile()
	if err != nil {
		// Créer un nouveau fichier
		return w.createNewFile(newCodes)
	}

	// Ouvrir le fichier existant
	filePath := filepath.Join(w.basePath, latestFile)
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", fmt.Errorf("erreur ouverture fichier existant: %w", err)
	}
	defer f.Close()

	// Lire les données existantes
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("aucune feuille trouvée")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return "", err
	}

	// Trouver le prochain ID disponible
	maxID := 0
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) > 0 {
			var id int
			fmt.Sscanf(rows[i][0], "%d", &id)
			if id > maxID {
				maxID = id
			}
		}
	}

	// Ajouter les nouveaux codes
	nextRow := len(rows) + 1
	for i, code := range newCodes {
		rowNum := nextRow + i
		code.ID = maxID + i + 1

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), code.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), code.Carrier)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), code.Action)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), code.Target)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), code.Operation)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), code.USSDCode)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), code.InformationINPUT)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), code.InformationOUTPUT)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), "In")
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNum), code.Comment)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNum), code.ParentUSSDID)
	}

	// Générer le nouveau nom de fichier
	timestamp := time.Now().Format("02012006-150405")
	newFilename := fmt.Sprintf("Codes_USSD_CI-v%s.xlsx", timestamp)
	newFilePath := filepath.Join(w.basePath, newFilename)

	// Sauvegarder le nouveau fichier
	if err := f.SaveAs(newFilePath); err != nil {
		return "", fmt.Errorf("erreur sauvegarde nouveau fichier: %w", err)
	}

	w.logger.Infof("Nouveau fichier Excel créé: %s", newFilename)
	return newFilename, nil
}

func (w *ExcelWriter) createNewFile(newCodes []USSDCode) (string, error) {
	f := excelize.NewFile()
	sheetName := "Codes USSD CI"
	f.SetSheetName("Sheet1", sheetName)

	// En-têtes
	headers := []string{"ID", "Carrier", "Action", "Target", "Operation", "USSD_Code", "Information_INPUT", "Information_OUTPUT", "Scope", "Comment", "Parent_USSD_ID"}
	for i, header := range headers {
		col := string(rune('A' + i))
		f.SetCellValue(sheetName, fmt.Sprintf("%s1", col), header)
	}

	// Ajouter les codes
	for i, code := range newCodes {
		rowNum := i + 2
		code.ID = i + 1

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), code.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), code.Carrier)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), code.Action)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), code.Target)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), code.Operation)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowNum), code.USSDCode)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowNum), code.InformationINPUT)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowNum), code.InformationOUTPUT)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowNum), "In")
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowNum), code.Comment)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowNum), code.ParentUSSDID)
	}

	timestamp := time.Now().Format("02012006-150405")
	newFilename := fmt.Sprintf("Codes_USSD_CI-v%s.xlsx", timestamp)
	newFilePath := filepath.Join(w.basePath, newFilename)

	if err := f.SaveAs(newFilePath); err != nil {
		return "", err
	}

	return newFilename, nil
}
