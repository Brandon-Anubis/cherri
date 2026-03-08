/*
 * Copyright (c) Cherri
 */

package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/electrikmilk/args-parser"
)

var actionCategories = []string{"basic"}
var currentCategory string

type selfDoc struct {
	title       string
	description string
	warning     string
	category    string
	subcategory string
}

type actionCategory struct {
	title         string
	actions       []string
	subcategories map[string][]string
}

func generateDocs() {
	defineToggleSetActions()
	defineRawAction()
	loadActionsByCategory()
	var cat = args.Value("docs")

	// If a pack-specific docs request is made via --docs-pack, show that pack's actions.
	if args.Using("docs-pack") {
		generatePackDocs(args.Value("docs-pack"))
		return
	}

	for i, category := range actionCategories {
		if cat != "" && cat != category {
			continue
		}
		var actionCategory = generateCategory(category)
		fmt.Println("#", actionCategory.title)

		if category == "basic" {
			fmt.Println("\nActions in this category are automatically included.")
		} else {
			fmt.Print("\nTo use actions in this category, use this include statement:\n\n")

			if args.Using("no-ansi") {
				fmt.Printf("```\n#include 'actions/%s'\n```\n", category)
			} else {
				fmt.Println(ansi(fmt.Sprintf("#include 'actions/%s'", category), red))
			}
		}

		slices.Sort(actionCategory.actions)
		fmt.Println(strings.Join(actionCategory.actions, "\n\n---\n"))

		if actionCategory.subcategories != nil {
			printCategories(actionCategory.subcategories)
		}

		if cat == "" && i != 0 {
			fmt.Print("---\n")
		}
	}
}

func printCategories(categories map[string][]string) {
	var keys []string
	for k := range categories {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		var category = categories[k]
		fmt.Printf("\n## %s\n", k)
		slices.Sort(category)
		fmt.Println(strings.Join(category, "\n\n---\n"))
	}
}

func generateCategory(category string) actionCategory {
	var categoryTitle = category
	switch categoryTitle {
	case "pdf":
		categoryTitle = "PDF"
	case "a11y":
		categoryTitle = "Accessibility"
	case "intelligence":
		categoryTitle = "Apple Intelligence"
	}
	var cat = actionCategory{
		title: fmt.Sprintf("%s Actions", capitalize(categoryTitle)),
	}
	var subcat = args.Value("subcat")
	for name, def := range actions {
		if def.doc.category != category || (subcat != "" && def.doc.subcategory != subcat) {
			continue
		}

		currentAction = *def
		currentActionIdentifier = name
		var definition = generateActionDefinition(parameterDefinition{}, true)

		if def.doc.subcategory != "" {
			if cat.subcategories == nil {
				cat.subcategories = make(map[string][]string)
			}
			cat.subcategories[def.doc.subcategory] = append(cat.subcategories[def.doc.subcategory], definition)
			continue
		}

		cat.actions = append(cat.actions, definition)
	}
	return cat
}

// generatePackDocs renders documentation for all actions provided by the named function pack.
func generatePackDocs(packID string) {
	loadPackRegistry()
	var entry, found = packRegistry.byID[packID]
	if !found {
		fmt.Printf("Function pack '%s' not found.\n", packID)
		return
	}

	fmt.Printf("# %s Actions\n\n", entry.manifest.DisplayName)
	fmt.Printf("Pack: `%s`  Version: %s  Author: %s\n", entry.manifest.ID, entry.manifest.Version, entry.manifest.Author)
	if entry.manifest.Homepage != "" {
		fmt.Printf("Homepage: %s\n", entry.manifest.Homepage)
	}
	fmt.Printf("\nTo use actions from this pack, add to your source file:\n\n")
	if args.Using("no-ansi") {
		fmt.Printf("```\n#pack '%s'\n```\n\n", packID)
	} else {
		fmt.Println(ansi(fmt.Sprintf("#pack '%s'", packID), red))
		fmt.Print("\n")
	}

	// Load pack actions so we can document them.
	resolvePackDirective(packID, "", "", "")
	resetParse()
	handleActionDefinitions()

	var packActions []string
	for name, def := range actions {
		if def.packID != packID {
			continue
		}
		currentAction = *def
		currentActionIdentifier = name
		packActions = append(packActions, generateActionDefinition(parameterDefinition{}, true))
	}

	if len(packActions) == 0 {
		fmt.Println("No documented actions found in this pack.")
		return
	}

	slices.Sort(packActions)
	fmt.Println(strings.Join(packActions, "\n\n---\n"))
}
