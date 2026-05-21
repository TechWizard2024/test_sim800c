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
		return &MenuNode{
			Code:        fmt.Sprintf("%s%s", parentCode, option.Number),
			Description: option.Text,
			Depth:       depth,
			ParentID:    parentID,
		}, nil, nil
	}

	// Construire le code USSD complet
	fullCode := fmt.Sprintf("%s%s", parentCode, option.Number)

	// Vérifier si le code existe déjà dans l'Excel
	exists, existingCode := e.excelReader.GetByUSSDCode(fullCode)

	var discoveredCodes []excel.USSDCode

	if !exists {
		// Nouveau code découvert
		newCode := excel.USSDCode{
			USSDCode:     fullCode,
			Operation:    option.Text,
			Action:       "Services_N2",
			Target:       "Interne",
			Scope:        "In",
			ParentUSSDID: parentID,
		}
		discoveredCodes = append(discoveredCodes, newCode)
	}

	node := &MenuNode{
		Code:        fullCode,
		Description: option.Text,
		Depth:       depth,
		ParentID:    parentID,
	}

	// Exécuter le sous-menu
	req := &USSDRequest{
		Module: module,
		Code:   fullCode,
	}

	response, err := e.executor.Execute(req)
	if err != nil {
		e.logger.Warnf("Erreur exécution sous-menu %s: %v", fullCode, err)
		node.Options = []MenuOption{}
		return node, discoveredCodes, nil
	}

	// Analyser les options du sous-menu
	subOptions := e.executor.ParseMenuResponse(response.Result)
	node.Options = subOptions

	// Explorer récursivement
	for _, subOption := range subOptions {
		childNode, childDiscovered, err := e.exploreSubMenu(module, subOption, fullCode, depth+1, existingCode.ID)
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
