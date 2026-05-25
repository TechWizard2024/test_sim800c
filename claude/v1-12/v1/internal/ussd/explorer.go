package ussd

import (
	"fmt"
	"strings"
	"time"

	"sim800c-supervisor/internal/excel"
	"sim800c-supervisor/internal/serial"

	"github.com/sirupsen/logrus"
)

type USSDExplorer struct {
	executor    *USSDExecutor
	excelReader *excel.ExcelReader
	excelWriter *excel.ExcelWriter
	logger      *logrus.Logger
	maxDepth    int
}

type MenuOption struct {
	Number   string
	Text     string
	FullText string
	USSDCode string
	ParentID int
}

type ExplorationResult struct {
	DiscoveredCodes []excel.USSDCode
	MenuTree        *MenuNode
	Duration        time.Duration
}

type MenuNode struct {
	Code        string
	Description string
	Options     []MenuOption
	Children    []*MenuNode
	Depth       int
	ParentID    int
}

func NewUSSDExplorer(executor *USSDExecutor, excelReader *excel.ExcelReader, excelWriter *excel.ExcelWriter, logger *logrus.Logger, maxDepth int) *USSDExplorer {
	return &USSDExplorer{
		executor:    executor,
		excelReader: excelReader,
		excelWriter: excelWriter,
		logger:      logger,
		maxDepth:    maxDepth,
	}
}

// SetMaxDepth met à jour dynamiquement la profondeur max d'exploration de menu USSD.
// Appelé depuis PUT /api/config/advanced pour que le changement prenne effet immédiatement
// sans redémarrage de l'application.
func (e *USSDExplorer) SetMaxDepth(depth int) {
	if depth > 0 {
		e.maxDepth = depth
		e.logger.Infof("USSDExplorer: max_menu_depth mis à jour → %d", depth)
	}
}

func (e *USSDExplorer) ExploreMenu(module *serial.SIM800C, startCode string, parentID int) (*ExplorationResult, error) {
	startTime := time.Now()

	e.logger.Infof("Exploration du menu USSD: %s (parent ID: %d)", startCode, parentID)

	result := &ExplorationResult{
		DiscoveredCodes: []excel.USSDCode{},
		MenuTree: &MenuNode{
			Code:     startCode,
			Depth:    0,
			ParentID: parentID,
		},
	}

	// Exécuter le code USSD initial
	req := &USSDRequest{
		Module:   module,
		Code:     startCode,
		ModuleID: 0, // Sera rempli par l'appelant
	}

	response, err := e.executor.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("erreur exécution code initial: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("échec exécution: %s", response.Error)
	}

	// Analyser le menu
	options := e.executor.ParseMenuResponse(response.Result)
	result.MenuTree.Options = options

	// Explorer récursivement
	for _, option := range options {
		childNode, discovered, err := e.exploreSubMenu(module, option, startCode, 1, parentID)
		if err != nil {
			e.logger.Warnf("Erreur exploration sous-menu %s: %v", option.Number, err)
			continue
		}
		result.MenuTree.Children = append(result.MenuTree.Children, childNode)
		result.DiscoveredCodes = append(result.DiscoveredCodes, discovered...)
	}

	result.Duration = time.Since(startTime)

	// Sauvegarder les nouveaux codes découverts
	if len(result.DiscoveredCodes) > 0 {
		_, err := e.excelWriter.CreateNewVersion(result.DiscoveredCodes)
		if err != nil {
			e.logger.Errorf("Erreur sauvegarde nouveaux codes: %v", err)
		}
	}

	return result, nil
}

func (e *USSDExplorer) exploreSubMenu(module *serial.SIM800C, option MenuOption, parentCode string, depth int, parentID int) (*MenuNode, []excel.USSDCode, error) {
	if depth >= e.maxDepth {
		// Dans le mode B, le choix n'est pas concaténé au code USSD.
		return &MenuNode{
			Code:        parentCode,
			Description: option.Text,
			Depth:       depth,
			ParentID:    parentID,
		}, nil, nil
	}

	// Mode B : le module attend le choix séparé ("1", "2", ...)
	// - on exécute le menu parentCode puis on envoie l'entrée option.Number.
	// Ici on explore le sous-menu en envoyant option.Number comme choice.
	var discoveredCodes []excel.USSDCode

	// On utilise comme "USSDCode" la séquence parentCode + "[choice]" pour repérer
	// des items distincts sans fabriquer une concaténation invalide.
	// Comme la spec dit USSD_Code structure, on conserve parentCode et on stocke le choix dans Operation.
	usCodeKey := parentCode
	// Déduplication basée sur USSDCode existant.
	exists, existingCode := e.excelReader.GetByUSSDCode(usCodeKey)

	if !exists {
		newCode := excel.USSDCode{
			USSDCode:     usCodeKey,
			Operation:    option.Text,
			Action:       "Services_N2",
			Target:       "Interne",
			Scope:        "In",
			ParentUSSDID: parentID,
		}
		discoveredCodes = append(discoveredCodes, newCode)
		_ = existingCode
	}

	node := &MenuNode{
		Code:        parentCode,
		Description: option.Text,
		Depth:       depth,
		ParentID:    parentID,
	}

	// Exécuter le sous-menu en envoyant le choix.
	response, err := e.executor.ExecuteWithMenu(&USSDRequest{Module: module, Code: parentCode, ModuleID: module.ModuleID}, option.Number)
	if err != nil {
		e.logger.Warnf("Erreur exécution sous-menu choice=%s parent=%s: %v", option.Number, parentCode, err)
		node.Options = []MenuOption{}
		return node, discoveredCodes, nil
	}

	subOptions := e.executor.ParseMenuResponse(response.Result)
	node.Options = subOptions

	// Explorer récursivement : le parentCode reste identique (mode B navigue dans la même session)
	for _, subOption := range subOptions {
		childNode, childDiscovered, err := e.exploreSubMenu(module, subOption, parentCode, depth+1, existingCode.ID)
		if err != nil {
			continue
		}
		node.Children = append(node.Children, childNode)
		discoveredCodes = append(discoveredCodes, childDiscovered...)
	}

	return node, discoveredCodes, nil
}

func (e *USSDExplorer) ExploreAllModules(modules []*serial.SIM800C, startCodes []string) map[int]*ExplorationResult {
	results := make(map[int]*ExplorationResult)

	for _, module := range modules {
		for _, startCode := range startCodes {
			result, err := e.ExploreMenu(module, startCode, 0)
			if err != nil {
				e.logger.Errorf("Erreur exploration module %s: %v", module.Port, err)
				continue
			}
			results[module.ModuleID] = result
		}
	}

	return results
}

func (e *USSDExplorer) FormatMenuTree(node *MenuNode, indent int) string {
	var result strings.Builder

	prefix := strings.Repeat("  ", indent)
	result.WriteString(fmt.Sprintf("%s📁 Code: %s\n", prefix, node.Code))
	if node.Description != "" {
		result.WriteString(fmt.Sprintf("%s   Description: %s\n", prefix, node.Description))
	}

	if len(node.Options) > 0 {
		result.WriteString(fmt.Sprintf("%s   Options:\n", prefix))
		for _, opt := range node.Options {
			result.WriteString(fmt.Sprintf("%s     %s. %s\n", prefix, opt.Number, opt.Text))
		}
	}

	for _, child := range node.Children {
		result.WriteString(e.FormatMenuTree(child, indent+1))
	}

	return result.String()
}
