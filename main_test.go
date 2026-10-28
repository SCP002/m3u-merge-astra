package main

import (
	"m3u-merge-astra/astra"
	"m3u-merge-astra/astra/api"
	"m3u-merge-astra/astra/api/apitest"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/network"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMainFunc(t *testing.T) {
	programCfgPath := filepath.Join(t.TempDir(), "m3u-merge-astra-main-test.yaml")

	// Read initial astra config.
	log := logger.New(logger.DebugLevel, os.Stderr)

	// Create program config file.
	_, _, err := cfg.Init(log, programCfgPath)
	assert.NoError(t, err)

	// Create M3U file.
	m3uPath := filepath.Join(t.TempDir(), "m3u-merge-astra-main-test.m3u")
	m3uBytes := []byte(`#EXTM3U
	#EXTINF:-1 group-title="Group 3",Channel 3
	http://url/3
	#EXTINF:-1 group-title="Group 4",Channel 4
	http://url/4`)
	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err)

	apiHttpClient := network.NewHttpClient(time.Second * 10)
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
	})

	// Run program.
	os.Args = []string{"", "-n", "-l", "2", "-c", programCfgPath, "-m", m3uPath, "-a", astraAddr,
		"-u", apitest.User, "-p", apitest.Password}
	main()

	// Read modified config.
	astraCfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err)

	// Check modified config.
	expectedCategories := []astra.Category{
		{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
	}
	assert.Exactly(t, expectedCategories, astraCfg.Categories)

	// Checking fields separately as new streams contains random ID.
	assert.Exactly(t, "Channel 1", astraCfg.Streams[0].Name)
	assert.Exactly(t, "Channel 2", astraCfg.Streams[1].Name)
	assert.Exactly(t, "_ADDED: Channel 3", astraCfg.Streams[2].Name)
	assert.Exactly(t, "_ADDED: Channel 4", astraCfg.Streams[3].Name)
}
