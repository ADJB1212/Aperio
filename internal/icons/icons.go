package icons

import (
	"path/filepath"
	"strings"
)

const (
	GenericCode = ""
	GearConfig  = ""
	Terminal    = ""
)

var nameToIcon = map[string]string{
	"makefile":       GearConfig,
	"cmakelists.txt": GearConfig,
	"dockerfile":     "",
	".bashrc":        Terminal,
	".zshrc":         Terminal,
	".profile":       Terminal,
	".bash_profile":  Terminal,
}

var extToIcon = map[string]string{

	".go":  "",
	".rs":  "",
	".zig": "",

	".js":  "",
	".jsx": "",
	".ts":  "",
	".tsx": "",
	".mjs": "",
	".cjs": "",

	".py":  "",
	".pyw": "",

	".c":   "",
	".h":   "",
	".hpp": "",
	".hh":  "",
	".hxx": "",
	".cc":  "",
	".cpp": "",
	".cxx": "",
	".m":   "",
	".mm":  "",

	".java":  "",
	".kt":    "",
	".kts":   "",
	".scala": "",
	".swift": "",

	".rb":  "",
	".erb": "",
	".php": "",

	".lua": "",
	".hs":  "",

	".vim": "",

	".html": "",
	".htm":  "",
	".css":  "",
	".scss": "",
	".sass": "",
	".less": "",

	".json":     "",
	".jsonc":    "",
	".jsonl":    "",
	".json5":    "",
	".yaml":     GearConfig,
	".yml":      GearConfig,
	".toml":     GearConfig,
	".ini":      GearConfig,
	".conf":     GearConfig,
	".md":       "",
	".markdown": "",

	".sh":   Terminal,
	".bash": Terminal,
	".zsh":  Terminal,
	".ksh":  Terminal,
	".fish": Terminal,
}

func Icon(path string) string {
	base := filepath.Base(path)

	lowerBase := strings.ToLower(base)
	if ic, ok := nameToIcon[lowerBase]; ok {
		return ic
	}

	if ic, ok := nameToIcon[base]; ok {
		return ic
	}

	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {
		return ""
	}
	if ic, ok := extToIcon[ext]; ok {
		return ic
	}
	return ""
}

func IconOr(name, fallback string) string {
	if ic := Icon(name); ic != "" {
		return ic
	}
	return fallback
}

func IsCode(name string) bool {

	lowerBase := strings.ToLower(filepath.Base(name))
	if _, ok := nameToIcon[lowerBase]; ok {
		return true
	}
	if _, ok := nameToIcon[filepath.Base(name)]; ok {
		return true
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	_, ok := extToIcon[ext]
	return ok
}

func KnownExtensions() []string {
	out := make([]string, 0, len(extToIcon))
	for ext := range extToIcon {
		out = append(out, ext)
	}
	return out
}
