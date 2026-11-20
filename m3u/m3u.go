package m3u

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/slice"
	urlUtil "m3u-merge-astra/util/url"

	"github.com/samber/lo"
)

// Channel represents M3U channel object.
type Channel struct {
	Name  string
	Group string
	URL   string
}

// GetName used to satisfy util/slice.Named interface.
func (ch Channel) GetName() string {
	return ch.Name
}

// replaceGroup returns channel with group taken from `cfg`, running `callback` with new group on change.
func (ch Channel) replaceGroup(cfg cfg.M3U, callback func(string)) Channel {
	newGroup := cfg.ChannGroupMap[ch.Group]
	if ch.Group != newGroup && newGroup != "" {
		callback(newGroup)
		ch.Group = newGroup
	}
	return ch
}

// Parse parses `rawChannels` into []Channel.
func (p Processor) Parse(rawChannels io.ReadCloser) (out []Channel) {
	p.log.Info("Parsing M3U channels")

	nameRx := regexp.MustCompile(`^#EXTINF:.*?,(.*)`)
	groupTitleRx := regexp.MustCompile(`^#EXTINF:.*group-title="(.*?)"`)
	extGrpRx := regexp.MustCompile(`^#EXTGRP:(.*)`)

	name := ""
	groupTitle := ""
	lastExtGrp := ""

	scanner := bufio.NewScanner(rawChannels)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}
		if matchList := nameRx.FindStringSubmatch(line); len(matchList) > 1 {
			name = strings.TrimSpace(matchList[1])
		}
		if matchList := groupTitleRx.FindStringSubmatch(line); len(matchList) > 1 {
			groupTitle = strings.TrimSpace(matchList[1])
		}
		if matchList := extGrpRx.FindStringSubmatch(line); len(matchList) > 1 {
			lastExtGrp = strings.TrimSpace(matchList[1])
			continue
		}
		if !strings.HasPrefix(line, "#") && name != "" {
			ch := Channel{
				Name: name,
				// group-title have a priority over #EXTGRP.
				Group: lo.Ternary(groupTitle != "", groupTitle, lastExtGrp),
				URL:   line,
			}
			out = append(out, ch)
			name = ""
			groupTitle = ""
			// #EXTGRP applies to every subsequent channel until overriden. Not clearing lastExtGrp.
		}
	}
	if err := scanner.Err(); err != nil {
		p.log.Error(err)
	}

	return
}

// Sort returns `channels` sorted by name.
func (p Processor) Sort(channels []Channel) (out []Channel) {
	p.log.Info("Sorting M3U channels")

	out = slice.Sort(channels)

	return
}

// ReplaceGroups returns `channels` with groups taken from map in config.
func (p Processor) ReplaceGroups(channels []Channel) (out []Channel) {
	p.log.Info("Replacing groups of M3U channels")

	for _, ch := range channels {
		out = append(out, ch.replaceGroup(p.cfg.M3U, func(newGroup string) {
			p.log.InfoFi("Replacing group of M3U channel", "name", ch.Name, "old group", ch.Group,
				"new group", newGroup)
		}))
	}

	return
}

// RemoveBlocked returns `channels` without blocked ones.
func (p Processor) RemoveBlocked(channels []Channel) (out []Channel) {
	p.log.Info("Removing blocked channels")

	// getAliases returns aliases for the `name` or slice of a single `name` if not found.
	getAliases := func(name string) []string {
		aliases, found := lo.Find(p.cfg.General.NameAliasList, func(set []string) bool {
			return lo.Contains(set, name)
		})
		return lo.Ternary(found, aliases, []string{name})
	}

	out = lo.Reject(channels, func(ch Channel, _ int) bool {
		names := []string{ch.Name}
		if p.cfg.General.NameAliases {
			names = getAliases(ch.Name)
		}
		reject := slice.AnyRxMatchAny(p.cfg.M3U.ChannNameBlacklist, names...) ||
			slice.AnyRxMatch(p.cfg.M3U.ChannGroupBlacklist, ch.Group) ||
			slice.AnyRxMatch(p.cfg.M3U.ChannURLBlacklist, ch.URL)
		if reject {
			p.log.InfoFi("Removing blocked channel", "name", ch.Name, "group", ch.Group, "URL", ch.URL)
		}
		return reject
	})

	return
}

// HasURL returns true if `channels` contain `url`.
//
// If `withHash` is false, ignore hashes (everything after #) during the search.
func HasURL(log *logger.Logger, channels []Channel, url string, withHash bool) bool {
	return lo.ContainsBy(channels, func(ch Channel) bool {
		equal, err := urlUtil.Equal(ch.URL, url, withHash)
		if err != nil {
			log.Debug(err)
		}
		return equal
	})
}
