/*
 * Copyright (c) Cherri
 */

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/electrikmilk/args-parser"
)

// TestPackManifestParsing verifies that pack.json files are correctly parsed into packManifest structs.
func TestPackManifestParsing(t *testing.T) {
	var tmpDir = t.TempDir()
	var manifest = `{
		"id": "my-pack",
		"displayName": "My Pack",
		"version": "1.2.3",
		"author": "Test Author",
		"prefix": "mp",
		"sources": ["my-pack.cherri"],
		"identifierIndex": {
			"com.example.mypack.hello": "mpHello"
		}
	}`
	var manifestPath = filepath.Join(tmpDir, "pack.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}

	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	registerPackFromPath(manifestPath)

	var entry, found = packRegistry.byID["my-pack"]
	if !found {
		t.Fatal("pack 'my-pack' was not registered")
	}
	if entry.manifest.DisplayName != "My Pack" {
		t.Errorf("displayName: got %q, want %q", entry.manifest.DisplayName, "My Pack")
	}
	if entry.manifest.Version != "1.2.3" {
		t.Errorf("version: got %q, want %q", entry.manifest.Version, "1.2.3")
	}
	if entry.manifest.Prefix != "mp" {
		t.Errorf("prefix: got %q, want %q", entry.manifest.Prefix, "mp")
	}
	if entry.rootDir != tmpDir {
		t.Errorf("rootDir: got %q, want %q", entry.rootDir, tmpDir)
	}
	if packRegistry.shortcutIdentifierToPackID["com.example.mypack.hello"] != "my-pack" {
		t.Error("shortcutIdentifierToPackID not populated correctly")
	}
	if packRegistry.actionToPack["mpHello"] != "my-pack" {
		t.Error("actionToPack not populated correctly")
	}
}

// TestPackManifestInvalidJSON verifies that malformed manifests are silently ignored.
func TestPackManifestInvalidJSON(t *testing.T) {
	var tmpDir = t.TempDir()
	var badPath = filepath.Join(tmpDir, "pack.json")
	if err := os.WriteFile(badPath, []byte(`{invalid json`), 0600); err != nil {
		t.Fatal(err)
	}

	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	registerPackFromPath(badPath)

	if len(packRegistry.byID) != 0 {
		t.Error("expected no packs to be registered for invalid JSON")
	}
}

// TestPackManifestMissingID verifies that manifests without an id field are ignored.
func TestPackManifestMissingID(t *testing.T) {
	var tmpDir = t.TempDir()
	var manifestPath = filepath.Join(tmpDir, "pack.json")
	if err := os.WriteFile(manifestPath, []byte(`{"displayName":"No ID"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	registerPackFromPath(manifestPath)

	if len(packRegistry.byID) != 0 {
		t.Error("expected no packs registered when id is missing")
	}
}

// TestPackDuplicateIDConflict verifies that registering the same pack ID from two different
// directories records both paths for later conflict detection.
func TestPackDuplicateIDConflict(t *testing.T) {
	var dir1 = t.TempDir()
	var dir2 = t.TempDir()

	var manifest = `{"id":"dup-pack","displayName":"Dup","version":"1.0.0","sources":[]}`
	if err := os.WriteFile(filepath.Join(dir1, "pack.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "pack.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}

	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	registerPackFromPath(filepath.Join(dir1, "pack.json"))
	registerPackFromPath(filepath.Join(dir2, "pack.json"))

	var entry = packRegistry.byID["dup-pack"]
	if entry == nil {
		t.Fatal("pack was not registered at all")
	}
	if len(entry.paths) != 2 {
		t.Errorf("expected 2 conflicting paths, got %d", len(entry.paths))
	}
}

// TestPackIdempotentRegistration verifies that registering the same pack from the same path twice is a no-op.
func TestPackIdempotentRegistration(t *testing.T) {
	var tmpDir = t.TempDir()
	var manifest = `{"id":"idempotent-pack","displayName":"Idempotent","version":"1.0.0","sources":[]}`
	var manifestPath = filepath.Join(tmpDir, "pack.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}

	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	registerPackFromPath(manifestPath)
	registerPackFromPath(manifestPath)

	var entry = packRegistry.byID["idempotent-pack"]
	if entry == nil {
		t.Fatal("pack was not registered")
	}
	if len(entry.paths) != 1 {
		t.Errorf("expected 1 path after idempotent re-registration, got %d", len(entry.paths))
	}
}

// TestPackVersionConstraintEqual verifies == version constraint checking.
func TestPackVersionConstraintEqual(t *testing.T) {
	if !checkPackVersionConstraint("1.2.3", "==", "1.2.3") {
		t.Error("expected == constraint to match identical versions")
	}
	if checkPackVersionConstraint("1.2.4", "==", "1.2.3") {
		t.Error("expected == constraint to reject different versions")
	}
}

// TestPackVersionConstraintGreaterOrEqual verifies >= version constraint checking.
func TestPackVersionConstraintGreaterOrEqual(t *testing.T) {
	if !checkPackVersionConstraint("1.2.3", ">=", "1.2.3") {
		t.Error("expected >= to pass for equal version")
	}
	if !checkPackVersionConstraint("2.0.0", ">=", "1.9.9") {
		t.Error("expected >= to pass for newer installed version")
	}
	if checkPackVersionConstraint("1.0.0", ">=", "1.2.3") {
		t.Error("expected >= to fail for older installed version")
	}
}

// TestCompareVersionStrings verifies compareVersionStrings returns correct ordering.
func TestCompareVersionStrings(t *testing.T) {
	if compareVersionStrings("1.2.3", "1.2.3") != 0 {
		t.Error("equal versions should compare as 0")
	}
	if compareVersionStrings("2.0.0", "1.9.9") <= 0 {
		t.Error("2.0.0 should be greater than 1.9.9")
	}
	if compareVersionStrings("1.0.0", "1.0.1") >= 0 {
		t.Error("1.0.0 should be less than 1.0.1")
	}
}

// TestPackLockFileRoundtrip verifies that writing and reading cherri.lock preserves entries.
func TestPackLockFileRoundtrip(t *testing.T) {
	var tmpDir = t.TempDir()
	lockFilePath = filepath.Join(tmpDir, "cherri.lock")

	var lockIn = packLockFile{
		Packs: []packLockEntry{
			{ID: "toolboxpro", Version: "3.1.2", Hash: "sha256:abc123"},
		},
	}
	writeLockFile(lockIn)

	// Override relativePath so loadLockFile finds the right file.
	var savedRelPath = relativePath
	relativePath = tmpDir + string(filepath.Separator)
	defer func() { relativePath = savedRelPath }()

	var lockOut, exists = loadLockFile()
	if !exists {
		t.Fatal("expected cherri.lock to exist after writing")
	}
	if len(lockOut.Packs) != 1 {
		t.Fatalf("expected 1 pack entry, got %d", len(lockOut.Packs))
	}
	if lockOut.Packs[0].ID != "toolboxpro" {
		t.Errorf("ID: got %q, want %q", lockOut.Packs[0].ID, "toolboxpro")
	}
	if lockOut.Packs[0].Version != "3.1.2" {
		t.Errorf("Version: got %q, want %q", lockOut.Packs[0].Version, "3.1.2")
	}
}

// TestFindPackByShortcutIdentifier verifies the shortcut-identifier-to-pack lookup.
func TestFindPackByShortcutIdentifier(t *testing.T) {
	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	packRegistry.byID["example"] = &packEntry{
		manifest: packManifest{
			ID:      "example",
			Version: "1.0.0",
			IdentifierIndex: map[string]string{
				"com.example.action.foo": "exFoo",
			},
		},
		rootDir: "/tmp/example",
		paths:   []string{"/tmp/example"},
	}
	packRegistry.shortcutIdentifierToPackID["com.example.action.foo"] = "example"

	var pid, funcName, found = findPackByShortcutIdentifier("com.example.action.foo")
	if !found {
		t.Fatal("expected to find pack for known shortcut identifier")
	}
	if pid != "example" {
		t.Errorf("packID: got %q, want %q", pid, "example")
	}
	if funcName != "exFoo" {
		t.Errorf("functionName: got %q, want %q", funcName, "exFoo")
	}

	_, _, notFoundResult := findPackByShortcutIdentifier("com.unknown.action")
	if notFoundResult {
		t.Error("expected lookup to fail for unknown identifier")
	}
}

// TestSuggestPackForAction verifies that unimported pack actions are correctly surfaced.
func TestSuggestPackForAction(t *testing.T) {
	packRegistry = newPackRegistry()
	packRegistryLoaded = false
	packRegistry.byID["mypkg"] = &packEntry{
		manifest: packManifest{ID: "mypkg", Version: "1.0.0"},
		rootDir:  "/tmp/mypkg",
		paths:    []string{"/tmp/mypkg"},
	}
	packRegistry.actionToPack["myAction"] = "mypkg"

	var suggestion = suggestPackForAction("myAction")
	if suggestion == "" {
		t.Error("expected non-empty suggestion for known pack action")
	}

	var noSuggestion = suggestPackForAction("unknownAction")
	if noSuggestion != "" {
		t.Error("expected empty suggestion for unknown action")
	}
}

// TestPackImportCompile verifies that a .cherri file using #pack compiles successfully.
func TestPackImportCompile(t *testing.T) {
	args.Args["no-ansi"] = ""
	args.Args["skip-sign"] = ""

	currentTest = "tests/pack-import.cherri"
	os.Args[1] = currentTest

	compile()

	resetParser()

	// Clean up generated lock file so it does not affect other tests.
	_ = os.Remove("tests/cherri.lock")
}
