package main

import (
	"bytes"
	"m3u-merge-astra/astra"
	"m3u-merge-astra/astra/api"
	"m3u-merge-astra/astra/api/apitest"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/cli"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/network"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)

	// Create program config file.
	programCfgPath := filepath.Join(t.TempDir(), "m3u-merge-astra-main-test.yaml")
	_, _, err := cfg.Init(log, programCfgPath)
	assert.NoError(t, err)

	// Sort inputs by weight, to get inputs of existing stream reordered.
	cfgBytes, err := os.ReadFile(programCfgPath)
	assert.NoError(t, err)
	cfgBytes = bytes.Replace(cfgBytes, []byte("  input_weight_to_type_map: []\n"),
		[]byte("  input_weight_to_type_map:\n    - by: 'fast'\n      weight: 1\n"), 1)
	err = os.WriteFile(programCfgPath, cfgBytes, 0644)
	assert.NoError(t, err)

	// Create M3U file.
	m3uPath := filepath.Join(t.TempDir(), "m3u-merge-astra-main-test.m3u")
	m3uBytes := []byte(`#EXTM3U
	#EXTINF:-1 group-title="Group 1",Channel 1
	http://url/1/new
	#EXTINF:-1 group-title="Group 3",Channel 3
	http://url/3
	#EXTINF:-1 group-title="Group 4",Channel 4
	http://url/4`)
	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err)

	apiHttpClient := network.NewHTTPClient(time.Second * 10)
	astraAddr := apitest.NewServer(t).URL
	apiHandler := api.NewHandler(log, apiHttpClient, astraAddr, apitest.User, apitest.Password)

	// Add streams.
	apiHandler.SetStreams([]astra.Stream{
		{
			ID:      "0001",
			Name:    "Channel 1",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Inputs:  []string{"http://url/1"},
			Groups:  map[string]string{"Category 1": "Group 1"},
		},
		{
			ID:      "0002",
			Name:    "Channel 2",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Inputs:  []string{"http://url/2"},
			Groups:  map[string]string{"Category 1": "Group 2"},
		},
		{
			ID:      "0003",
			Name:    "Channel 5",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Groups:  map[string]string{"Category 1": "Group 1"},
		},
		{
			ID:      "0004",
			Name:    "Channel 6",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Inputs:  []string{"http://url/6/slow", "http://url/6/fast"},
			Groups:  map[string]string{"Category 1": "Group 1"},
		},
	}, 0)

	flags := cli.Flags{
		Noninteractive: true,
		ProgramCfgPath: programCfgPath,
		M3UPath:        m3uPath,
		AstraAddr:      astraAddr,
		AstraUser:      apitest.User,
		AstraPwd:       apitest.Password,
	}

	// Run program.
	assert.NoError(t, run(log, flags))

	// Read modified astra config.
	astraCfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err)

	// Check modified astra config.
	expectedCategories := []astra.Category{
		{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
	}
	assert.Exactly(t, expectedCategories, astraCfg.Categories)

	// Fake astra API moves updated streams to the end, so find them by ID.
	streams := lo.KeyBy(astraCfg.Streams, func(s astra.Stream) string {
		return s.ID
	})
	assert.ElementsMatch(t, []string{"0001", "0002", "0003", "0004", "a001", "a002"}, lo.Keys(streams))
	assert.Exactly(t, "Channel 1", streams["0001"].Name)
	assert.Exactly(t, []string{"http://url/1/new", "http://url/1"}, streams["0001"].Inputs,
		"should send new input of existing stream")
	assert.Exactly(t, "Channel 2", streams["0002"].Name)
	assert.Exactly(t, []string{"http://url/2"}, streams["0002"].Inputs)
	assert.Exactly(t, "_DISABLED: Channel 5", streams["0003"].Name, "should send disabling of existing stream")
	assert.False(t, streams["0003"].Enabled, "should send disabling of existing stream")
	assert.Exactly(t, []string{"http://url/6/fast", "http://url/6/slow"}, streams["0004"].Inputs,
		"should send sorted inputs of existing stream")
	assert.Exactly(t, "_ADDED: Channel 3", streams["a001"].Name)
	assert.Exactly(t, "_ADDED: Channel 4", streams["a002"].Name)
	assert.Exactly(t, 466562, astraCfg.GID)

	// Run program again on the modified astra config.
	assert.NoError(t, run(log, flags))

	astraCfg2, err := apiHandler.FetchCfg()
	assert.NoError(t, err)
	assert.Exactly(t, astraCfg, astraCfg2, "should not change anything on the second run")

	// Run program again with new input of a channel added as new stream.
	m3uBytes = append(m3uBytes, []byte(`
	#EXTINF:-1 group-title="Group 3",Channel 3
	http://url/3/new`)...)
	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err)
	assert.NoError(t, run(log, flags))

	astraCfg3, err := apiHandler.FetchCfg()
	assert.NoError(t, err)
	streams = lo.KeyBy(astraCfg3.Streams, func(s astra.Stream) string {
		return s.ID
	})
	assert.ElementsMatch(t, []string{"0001", "0002", "0003", "0004", "a001", "a002"}, lo.Keys(streams),
		"should find stream with name prefix by channel name instead of adding a new one")
	assert.Exactly(t, "_ADDED: Channel 3", streams["a001"].Name)
	assert.Exactly(t, []string{"http://url/3/new", "http://url/3"}, streams["a001"].Inputs,
		"should add new input to stream with name prefix")
}
