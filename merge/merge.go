package merge

import (
	"m3u-merge-astra/astra"
	"m3u-merge-astra/m3u"
	"m3u-merge-astra/util/slice/find"
	"slices"
	"strconv"

	"github.com/samber/lo"
)

// RenameStreams returns `streams` with names taken from `channels` if their standardized names are equal.
func (p Processor) RenameStreams(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	p.log.Info("Renaming streams")

	for _, s := range streams {
		ch, _, chFound := find.Named(p.cfg.General, channels, s.Name)
		if chFound && s.Name != ch.Name {
			p.log.InfoFi("Renaming stream", "ID", s.ID, "old name", s.Name, "new name", ch.Name,
				"group", s.FirstGroup())
			s.Name = ch.Name
		}
		out = append(out, s)
	}

	return
}

// UpdateInputs returns `streams` with every first matching input of every stream replaced with matching URL's of m3u
// channels according to cfg.Streams.InputUpdateMap.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream on update.
func (p Processor) UpdateInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	p.log.Info("Updating inputs of streams")

	for _, s := range streams {
		find.EverySimilar(p.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(p.log, ch.URL, true) {
				var updated bool
				s, updated = s.UpdateInput(p.log, p.cfg.Streams, ch.URL, func(oldURL string) {
					p.log.InfoFi("Updating input of stream", "ID", s.ID, "name", s.Name, "old URL", oldURL,
						"new URL", ch.URL, "note", s.InputsUpdateNote(p.cfg.Streams))
				})
				if p.cfg.Streams.EnableOnInputUpdate && updated && !s.Enabled {
					p.log.InfoFi("Enabling the stream (updating inputs of streams, enable_on_input_update is on)",
						"ID", s.ID, "name", s.Name)
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	return
}

// RemoveInputsByUpdateMap returns `streams` without inputs which match at least one cfg.Streams.InputUpdateMap.From
// expression but none found in `channels`.
func (p Processor) RemoveInputsByUpdateMap(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	p.log.Info("Removing absent inputs from streams according the update map")

	for _, s := range streams {
		similarChannels := find.GetSimilar(p.cfg.General, channels, s.Name)
		for _, knownInp := range s.KnownInputs(p.cfg.Streams) {
			if !m3u.HasURL(p.log, similarChannels, knownInp, false) {
				s = s.RemoveInputsCb(knownInp, func() {
					p.log.InfoFi("Removing absent input from stream according the update map", "ID", s.ID,
						"name", s.Name, "group", s.FirstGroup(), "input", knownInp)
				})
			}
		}
		out = append(out, s)
	}

	return
}

// AddNewInputs returns `streams` with new inputs if such found in `channels`.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream with new inputs.
func (p Processor) AddNewInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	p.log.Info("Adding new inputs to streams")

	for _, s := range streams {
		find.EverySimilar(p.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(p.log, ch.URL, p.cfg.Streams.HashCheckOnAddNewInputs) {
				p.log.InfoFi("Adding new input to stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
					"URL", ch.URL, "note", s.InputsUpdateNote(p.cfg.Streams))
				s = s.AddInput(ch.URL)
				if p.cfg.Streams.EnableOnInputUpdate && !s.Enabled {
					p.log.InfoFi("Enabling the stream (adding new inputs to streams, enable_on_input_update is on)",
						"ID", s.ID, "name", s.Name)
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	return
}

// AddNewStreams returns `streams` with new streams generated from `channels` if no such found in `streams` and ID
// counter `gid` advanced by generated IDs.
func (p Processor) AddNewStreams(streams []astra.Stream, channels []m3u.Channel, gid int) ([]astra.Stream, int) {
	p.log.Info("Adding new streams")

	streams = slices.Clip(streams)
	for _, ch := range channels {
		if !p.cfg.Streams.AddNewWithKnownInputs && astra.HasInput(p.log, streams, ch.URL, false) {
			continue
		}
		if !find.HasAnySimilar(p.cfg.General, streams, ch.Name) {
			var id string
			id, gid = generateUID(streams, gid)
			stream := astra.NewStream(p.cfg.Streams, id, ch.Name, ch.Group, []string{ch.URL})
			p.log.InfoFi("Adding new stream", "ID", id, "name", ch.Name, "group", stream.FirstGroup(), "input", ch.URL)
			streams = append(streams, stream)
		}
	}

	return streams, gid
}

// defaultGID is initial ID counter value used by astra web interface if config has none ("a000" in base 36).
const defaultGID = 466560

// generateUID returns ID unique for `streams` and advanced ID counter `gid`, the same way as astra web interface does:
// increments `gid` until its base 36 representation is not used as ID by any of `streams`.
func generateUID(streams []astra.Stream, gid int) (string, int) {
	if gid == 0 {
		gid = defaultGID
	}
	for {
		gid++
		uid := strconv.FormatInt(int64(gid), 36)
		duplicate := lo.ContainsBy(streams, func(s astra.Stream) bool {
			return s.ID == uid
		})
		if !duplicate {
			return uid, gid
		}
	}
}
