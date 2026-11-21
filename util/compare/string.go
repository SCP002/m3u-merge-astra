package compare

import (
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/simplify"
	"strings"

	"github.com/samber/lo"
)

// Names compares names using transliteration settings and aliases from config.
//
// It remembers standardized forms of every name it compares, as the same names are compared many times when matching
// lists of names. Create it for a single operation, as it uses config at the time of creation. Not safe for concurrent
// use.
type Names struct {
	cfg  cfg.General
	keys map[string]nameKeys
}

// nameKeys represents standardized forms of a name, any of which being equal makes names the same.
type nameKeys struct {
	simple  string // Simplified name.
	similar string // Simplified name with similar characters transliterated, if enabled in config.
	full    string // Simplified name fully transliterated, if enabled in config.
	alias   string // First alias of simplified name, if enabled in config.
}

// NewNames returns new name comparator configured with `cfg`.
func NewNames(cfg cfg.General) *Names {
	return &Names{cfg: cfg, keys: map[string]nameKeys{}}
}

// Same returns true if standardized `lName` is equal to standardized `rName`.
func (n *Names) Same(lName, rName string) bool {
	if lName == rName {
		return true
	}
	l, r := n.keysOf(lName), n.keysOf(rName)
	return l.simple == r.simple ||
		n.cfg.SimilarTranslit && l.similar == r.similar ||
		n.cfg.FullTranslit && l.full == r.full ||
		n.cfg.NameAliases && l.alias == r.alias
}

// keysOf returns standardized forms of `name`, computing them on the first call.
func (n *Names) keysOf(name string) nameKeys {
	if keys, ok := n.keys[name]; ok {
		return keys
	}
	keys := nameKeys{simple: simplify.Name(name)}
	if n.cfg.SimilarTranslit {
		keys.similar = remap(keys.simple, n.cfg.SimilarTranslitMap)
	}
	if n.cfg.FullTranslit {
		keys.full = remap(keys.simple, n.cfg.FullTranslitMap)
	}
	if n.cfg.NameAliases {
		keys.alias = firstAlias(keys.simple, n.cfg.SimpleNameAliasList)
	}
	n.keys[name] = keys
	return keys
}

// IsNameSame returns true if standardized `lName` is equal to standardized `rName` using transliteration settings and
// aliases from `cfg`.
//
// Use Names to compare many names.
func IsNameSame(cfg cfg.General, lName, rName string) bool {
	return NewNames(cfg).Same(lName, rName)
}

// remap returns remapped `inp` using `dict`.
func remap(inp string, dict map[string]string) string {
	var sb strings.Builder
	for _, oldChar := range inp {
		newChar := dict[string(oldChar)]
		if newChar != "" {
			sb.WriteString(newChar)
		} else {
			sb.WriteRune(oldChar)
		}
	}
	return sb.String()
}

// firstAlias returns first alias for `name` from `aliases` or `name` if not found.
func firstAlias(name string, aliases [][]string) string {
	for _, set := range aliases {
		if lo.Contains(set, name) {
			return set[0]
		}
	}
	return name
}
