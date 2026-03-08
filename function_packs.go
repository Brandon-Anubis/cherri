/*
 * Copyright (c) Cherri
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/electrikmilk/args-parser"
)

// packManifest defines the structure of a function pack's manifest file (pack.json).
type packManifest struct {
	ID               string            `json:"id"`
	DisplayName      string            `json:"displayName"`
	Version          string            `json:"version"`
	Author           string            `json:"author"`
	Homepage         string            `json:"homepage"`
	MinCherriVersion string            `json:"minCherriVersion"`
	Sources          []string          `json:"sources"`
	Prefix           string            `json:"prefix"`
	IdentifierIndex  map[string]string `json:"identifierIndex"`
}

// packEntry tracks a discovered pack and its filesystem location.
type packEntry struct {
	manifest packManifest
	rootDir  string   // directory containing pack.json
	paths    []string // all root dirs that claim this ID (for duplicate detection)
}

// packLockEntry is a single entry in cherri.lock.
type packLockEntry struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Hash    string `json:"hash"` // SHA-256 hex of manifest bytes prefixed with "sha256:"
}

// packLockFile is the top-level structure of cherri.lock.
type packLockFile struct {
	Packs []packLockEntry `json:"packs"`
}

// packRegistryType is the in-memory index of discovered function packs.
type packRegistryType struct {
	byID                       map[string]*packEntry
	actionToPack               map[string]string // function name -> pack ID
	shortcutIdentifierToPackID map[string]string // full WF action identifier -> pack ID
}

// packRegistry is the global registry of discovered function packs.
var packRegistry = newPackRegistry()

func newPackRegistry() *packRegistryType {
	return &packRegistryType{
		byID:                       make(map[string]*packEntry),
		actionToPack:               make(map[string]string),
		shortcutIdentifierToPackID: make(map[string]string),
	}
}

// importedPacks tracks pack IDs explicitly imported by the current source file via #pack.
var importedPacks []string

// importedPacksAliases maps pack ID to an alias prefix (for collision resolution).
var importedPacksAliases = map[string]string{}

// currentLoadingPackID is the pack ID currently being loaded (used to tag action definitions).
var currentLoadingPackID string

// packRegistryLoaded guards against double-loading the registry per compilation run.
var packRegistryLoaded bool

// packSearchDirs returns the ordered list of directories to search for function packs.
// The --pack-dir CLI flag takes priority, then the source-relative ./function-packs/,
// then the global user directory.
func packSearchDirs() []string {
	var base = "./"
	if relativePath != "" {
		base = relativePath
	}
	var dirs []string
	if args.Using("pack-dir") {
		dirs = append(dirs, args.Value("pack-dir"))
	}
	dirs = append(dirs, filepath.Join(base, "function-packs"))
	dirs = append(dirs, filepath.Join(os.ExpandEnv("$HOME"), ".config", "cherri", "function-packs"))
	return dirs
}

// loadPackRegistry discovers and registers all available packs from search dirs.
// This is idempotent: subsequent calls within the same compilation run are no-ops.
func loadPackRegistry() {
	if packRegistryLoaded {
		return
	}
	packRegistryLoaded = true

	var manifestPaths []string
	for _, dir := range packSearchDirs() {
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && d.Name() == "pack.json" {
				manifestPaths = append(manifestPaths, path)
			}
			return nil
		})
	}
	// Deterministic registration order.
	sort.Strings(manifestPaths)
	for _, p := range manifestPaths {
		registerPackFromPath(p)
	}
}

// registerPackFromPath reads and validates a pack.json file and inserts the pack into the registry.
func registerPackFromPath(manifestPath string) {
	var manifestBytes, readErr = os.ReadFile(manifestPath)
	if readErr != nil {
		return
	}
	var manifest packManifest
	if jsonErr := json.Unmarshal(manifestBytes, &manifest); jsonErr != nil {
		return
	}
	if manifest.ID == "" {
		return
	}
	var rootDir = filepath.Dir(manifestPath)
	if existing, found := packRegistry.byID[manifest.ID]; found {
		// Same path → idempotent re-registration.
		if existing.rootDir == rootDir {
			return
		}
		// Different path → record conflict; will be reported at #pack resolution time.
		existing.paths = append(existing.paths, rootDir)
		return
	}

	var entry = &packEntry{
		manifest: manifest,
		rootDir:  rootDir,
		paths:    []string{rootDir},
	}
	packRegistry.byID[manifest.ID] = entry

	// Build reverse-lookup indexes from identifierIndex.
	for shortcutIdentifier, functionName := range manifest.IdentifierIndex {
		packRegistry.shortcutIdentifierToPackID[shortcutIdentifier] = manifest.ID
		packRegistry.actionToPack[functionName] = manifest.ID
	}
}

// resolvePackDirective resolves a #pack directive for the source file currently being compiled.
// It validates version constraints, loads pack source files into lines, tracks the imported
// pack, and regenerates or validates cherri.lock.
func resolvePackDirective(packID string, op string, version string, alias string) {
	loadPackRegistry()

	if packID == "" {
		exit("Pack directive: missing pack ID.")
	}

	var entry, found = packRegistry.byID[packID]
	if !found {
		exit(fmt.Sprintf("Function pack '%s' not found. Install it in ./function-packs/ or ~/.config/cherri/function-packs/.", packID))
	}

	// Duplicate-installation conflict check.
	if len(entry.paths) > 1 {
		exit(fmt.Sprintf(
			"Function pack '%s' is installed in multiple locations:\n  %s\nDisambiguate using --pack-dir or remove one installation.",
			packID, strings.Join(entry.paths, "\n  "),
		))
	}

	// Version constraint validation.
	if op != "" && version != "" {
		if !checkPackVersionConstraint(entry.manifest.Version, op, version) {
			exit(fmt.Sprintf(
				"Function pack '%s': installed version %s does not satisfy constraint %s '%s'.",
				packID, entry.manifest.Version, op, version,
			))
		}
	}

	// Record alias if provided.
	if alias != "" {
		importedPacksAliases[packID] = alias
	}

	// Snapshot action names before loading so we can tag newly-added actions afterwards.
	var knownActions = make(map[string]bool, len(actions))
	for k := range actions {
		knownActions[k] = true
	}

	// Prepend each source file's contents to lines so handleActionDefinitions can process them.
	for i := len(entry.manifest.Sources) - 1; i >= 0; i-- {
		var sourcePath = filepath.Join(entry.rootDir, entry.manifest.Sources[i])
		var sourceBytes, readErr = os.ReadFile(sourcePath)
		if readErr != nil {
			exit(fmt.Sprintf("Function pack '%s': failed to read source '%s': %v", packID, entry.manifest.Sources[i], readErr))
		}
		lines = append([]string{string(sourceBytes) + "\n"}, lines...)
	}

	// Trigger action definition parsing immediately so we can tag actions with their pack ID.
	resetParse()
	handleActionDefinitions()

	// Tag all newly-added action definitions with the pack ID.
	for k, def := range actions {
		if !knownActions[k] && def.packID == "" {
			def.packID = packID
		}
	}

	if !containsStr(importedPacks, packID) {
		importedPacks = append(importedPacks, packID)
	}
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// checkPackVersionConstraint validates that an installed version satisfies op+constraint.
func checkPackVersionConstraint(installed, op, constraint string) bool {
	switch op {
	case "==":
		return installed == constraint
	case ">=":
		return compareVersionStrings(installed, constraint) >= 0
	}
	return true
}

// compareVersionStrings performs a lexicographic numeric comparison of dotted version strings.
func compareVersionStrings(a, b string) int {
	var aParts = strings.Split(a, ".")
	var bParts = strings.Split(b, ".")
	var maxLen = len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &av)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bv)
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// packManifestHash returns the SHA-256 hash of the manifest file for packID.
func packManifestHash(packID string) string {
	var entry, found = packRegistry.byID[packID]
	if !found {
		return ""
	}
	var manifestPath = filepath.Join(entry.rootDir, "pack.json")
	var manifestBytes, readErr = os.ReadFile(manifestPath)
	if readErr != nil {
		return ""
	}
	var sum = sha256.Sum256(manifestBytes)
	return "sha256:" + hex.EncodeToString(sum[:])
}

var lockFilePath string

// loadLockFile reads cherri.lock from the same directory as the source file.
func loadLockFile() (lock packLockFile, exists bool) {
	lockFilePath = filepath.Join(relativePath, "cherri.lock")
	var lockBytes, readErr = os.ReadFile(lockFilePath)
	if readErr != nil {
		return packLockFile{}, false
	}
	if jsonErr := json.Unmarshal(lockBytes, &lock); jsonErr != nil {
		return packLockFile{}, false
	}
	return lock, true
}

// writeLockFile writes the provided lock data to cherri.lock.
func writeLockFile(lock packLockFile) {
	if lockFilePath == "" {
		lockFilePath = filepath.Join(relativePath, "cherri.lock")
	}
	var lockBytes, marshalErr = json.MarshalIndent(lock, "", "\t")
	if marshalErr != nil {
		return
	}
	handle(os.WriteFile(lockFilePath, lockBytes, 0600))
}

// updateLockFile validates imported packs against cherri.lock (when it exists) and writes a
// fresh lock after a successful build.  Pass updating=true when --update-packs is set to skip
// hash validation and always regenerate.
func updateLockFile(updating bool) {
	if len(importedPacks) == 0 {
		return
	}
	var existingLock, lockExists = loadLockFile()
	var newLock packLockFile
	for _, packID := range importedPacks {
		var hash = packManifestHash(packID)
		var entry = packRegistry.byID[packID]
		if entry == nil {
			continue
		}
		if lockExists && !updating {
			for _, lockedEntry := range existingLock.Packs {
				if lockedEntry.ID != packID {
					continue
				}
				if lockedEntry.Hash != hash {
					exit(fmt.Sprintf(
						"Function pack '%s' hash does not match cherri.lock (expected %s, got %s). Run with --update-packs to refresh.",
						packID, lockedEntry.Hash, hash,
					))
				}
			}
		}
		newLock.Packs = append(newLock.Packs, packLockEntry{
			ID:      packID,
			Version: entry.manifest.Version,
			Hash:    hash,
		})
	}
	if len(newLock.Packs) > 0 {
		writeLockFile(newLock)
	}
}

// listInstalledPacks discovers and prints all available function packs.
func listInstalledPacks() {
	loadPackRegistry()
	if len(packRegistry.byID) == 0 {
		fmt.Println(ansi("No function packs installed.", yellow))
		fmt.Println("Install packs in ./function-packs/ or ~/.config/cherri/function-packs/")
		return
	}
	fmt.Println(ansi("Installed function packs:\n", green))
	var ids []string
	for id := range packRegistry.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		printPackEntry(packRegistry.byID[id])
	}
}

// printPackEntry prints human-readable information about a single function pack.
func printPackEntry(entry *packEntry) {
	fmt.Printf("- %s\n", ansi(entry.manifest.ID, blue))
	fmt.Printf("  Display Name: %s\n", entry.manifest.DisplayName)
	fmt.Printf("  Version:      %s\n", entry.manifest.Version)
	fmt.Printf("  Author:       %s\n", entry.manifest.Author)
	if entry.manifest.Homepage != "" {
		fmt.Printf("  Homepage:     %s\n", entry.manifest.Homepage)
	}
	if entry.manifest.Prefix != "" {
		fmt.Printf("  Prefix:       %s\n", entry.manifest.Prefix)
	}
	fmt.Printf("  Sources:      %v\n", entry.manifest.Sources)
	fmt.Printf("  Location:     %s\n", entry.rootDir)
	if len(entry.paths) > 1 {
		fmt.Printf("  %s\n", ansi("WARNING: Multiple installations detected!", yellow))
	}
}

// packInfoByID prints information about the pack with the given id.
func packInfoByID(packID string) {
	loadPackRegistry()
	var entry, found = packRegistry.byID[packID]
	if !found {
		fmt.Println(ansi(fmt.Sprintf("Function pack '%s' not found.", packID), red))
		return
	}
	printPackEntry(entry)
}

// findPackByShortcutIdentifier looks up a pack and Cherri function name by a Shortcuts
// action identifier (WFWorkflowActionIdentifier).
func findPackByShortcutIdentifier(identifier string) (packID string, functionName string, found bool) {
	var pid, ok = packRegistry.shortcutIdentifierToPackID[identifier]
	if !ok {
		return "", "", false
	}
	var entry = packRegistry.byID[pid]
	if entry == nil {
		return "", "", false
	}
	return pid, entry.manifest.IdentifierIndex[identifier], true
}

// suggestPackForAction checks whether any installed (but not yet imported) pack declares
// the given function name and returns a human-readable suggestion string.
func suggestPackForAction(actionName string) string {
	loadPackRegistry()
	var pid, ok = packRegistry.actionToPack[actionName]
	if !ok {
		return ""
	}
	return fmt.Sprintf("Action '%s()' is provided by function pack '%s'. Add:\n\n#pack '%s'", actionName, pid, pid)
}
