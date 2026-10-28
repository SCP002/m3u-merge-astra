package api

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"m3u-merge-astra/astra"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/network"
	"m3u-merge-astra/util/rnd"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)

	expected := handler{
		log:        log,
		httpClient: httpClient,
		address:    "127.0.0.1",
		user:       "user",
		password:   "pass",
	}
	assert.Exactly(t, expected, NewHandler(log, httpClient, "127.0.0.1", "user", "pass"))
}

// Requires a running astra.
func TestSetCategories(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Remove existing categories.
	config, err := apiHandler.FetchCfg()
	assert.NoError(t, err)
	for _, category := range config.Categories {
		category.Remove = true
		err := apiHandler.SetCategory(0, category)
		assert.NoError(t, err)
	}

	// Set.
	idxCategoryMap := []lo.Entry[int, astra.Category]{
		{ // 0
			Key:   -1,
			Value: astra.Category{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
		},
		{ // 1
			Key:   -1,
			Value: astra.Category{Name: "Category 2", Groups: []astra.Group{{Name: "Group 3"}, {Name: "Group 4"}}},
		},
		{ // 2
			Key:   -1,
			Value: astra.Category{Name: "Category 3", Groups: []astra.Group{{Name: "Group 5"}, {Name: "Group 6"}}},
		},
		{ // 3
			Key:   -1,
			Value: astra.Category{Name: "Category 4", Groups: []astra.Group{{Name: "Group 7"}, {Name: "Group 8"}}},
		},
		{ // 4
			Key:   -1,
			Value: astra.Category{Name: "Category 5", Groups: []astra.Group{{Name: "Group 9"}, {Name: "Group 10"}}},
		},
		{ // 5
			Key:   1,
			Value: astra.Category{Name: "Category 2*", Groups: []astra.Group{{Name: "Group 3*"}, {Remove: true}}},
		},
		{ // 4
			Key:   -1,
			Value: astra.Category{Name: "Category 6", Groups: []astra.Group{{Name: "Group 11"}, {Name: "Group 12"}}},
		},
		{ // 6
			Key:   4,
			Value: astra.Category{Remove: true},
		},
		{ // 7
			Key:   3,
			Value: astra.Category{Remove: true},
		},
	}
	apiHandler.SetCategories(idxCategoryMap)

	// Check.
	config, err = apiHandler.FetchCfg()
	assert.NoError(t, err)
	expected := []astra.Category{
		{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
		{Name: "Category 2*", Groups: []astra.Group{{Name: "Group 3*"}, {Remove: true}}},
		{Name: "Category 3", Groups: []astra.Group{{Name: "Group 5"}, {Name: "Group 6"}}},
		{Name: "Category 6", Groups: []astra.Group{{Name: "Group 11"}, {Name: "Group 12"}}},
	}
	assert.Equal(t, expected, config.Categories)

	// Test log output.
	var out bytes.Buffer
	log = logger.New(logger.DebugLevel, &out)
	httpClient = network.NewHttpClient(time.Second * 3)
	apiHandler = NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	idxCategoryMap = []lo.Entry[int, astra.Category]{
		{
			Key:   -1,
			Value: astra.Category{Name: "Category 0", Groups: []astra.Group{{Name: "Group 0"}, {Name: "Group 01"}}},
		},
	}
	apiHandler.SetCategories(idxCategoryMap)
	assert.Contains(t, out.String(), `Successfully set category: name "Category 0", `+
		`groups "[{Name:Group 0 Remove:false} {Name:Group 01 Remove:false}]", remove "false"`)
	assert.NotContains(t, out.String(), "Failed")
}

// Requires a running astra.
func TestSetCategory(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	astraCfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err)

	// Remove old categories.
	for _, category := range astraCfg.Categories {
		category.Remove = true
		err = apiHandler.SetCategory(0, category)
		if ok := assert.NoError(t, err); !ok {
			t.FailNow()
		}
	}

	// Set new category.
	err = apiHandler.SetCategory(-1, astra.Category{
		Name:   fmt.Sprintf("Category %v", rnd.String(4, false, true)),
		Groups: []astra.Group{{Name: "Group name 1"}, {Name: "Group name 2"}, {Name: "Group name 3"}},
	})
	assert.NoError(t, err)

	// Set new stream (test for crash when changing categories with existing streams).
	err = apiHandler.SetStream("0001", astra.Stream{
		ID:      "0001",
		Name:    "Name 1",
		Enabled: true,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err)

	astraCfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err)

	modifiedCategory := astra.Category{
		Name: "Category modified",
		Groups: []astra.Group{
			{Name: "Group name 1", Remove: true},
			{Name: "Group modified"},
			{Name: "Group name 3", Remove: true},
		},
	}
	err = apiHandler.SetCategory(0, modifiedCategory)
	assert.NoError(t, err)

	astraCfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err)
	assert.Equal(t, astraCfg.Categories, []astra.Category{modifiedCategory})
}

func TestSetStreams(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Remove existing streams.
	config, err := apiHandler.FetchCfg()
	assert.NoError(t, err)
	apiHandler.SetStreams(lo.Map(config.Streams, func(s astra.Stream, _ int) astra.Stream {
		s.Remove = true
		return s
	}))

	// Set.
	streams := []astra.Stream{
		{ID: "0001", Name: "Name 1", Type: string(cfg.SPTS)},
		{ID: "0002", Name: "Name 2", Type: string(cfg.SPTS)},
		{ID: "0003", Name: "Name 3", Type: string(cfg.SPTS)},
	}
	apiHandler.SetStreams(streams)

	// Check.
	config, err = apiHandler.FetchCfg()
	assert.NoError(t, err)
	assert.Equal(t, streams, config.Streams)

	// Test log output.
	var out bytes.Buffer
	log = logger.New(logger.DebugLevel, &out)
	httpClient = network.NewHttpClient(time.Second * 3)
	apiHandler = NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	streams = []astra.Stream{
		{ID: "0000", Name: "Name 0", Type: string(cfg.SPTS)},
	}
	apiHandler.SetStreams(streams)
	assert.Contains(t, out.String(), `Successfully set stream: ID "0000", name "Name 0"`)
	assert.NotContains(t, out.String(), "Failed")
}

// Requires a running astra.
func TestSetStream(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Set.
	streamName := fmt.Sprintf("Stream %v", rnd.String(4, false, true))
	err := apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID:      "0000",
		Inputs:  []string{"http://xxx/2", "http://xxx"},
		Groups:  map[string]string{"Category 1": "Group 1"},
		Name:    streamName,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err)

	// Change.
	err = apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID:      "0000",
		Inputs:  []string{"http://xxx"},
		Name:    streamName,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err)

	// Check.
	cfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err)
	assert.True(t, lo.ContainsBy(cfg.Streams, func(s astra.Stream) bool {
		return s.ID == "0000" && s.Name == streamName && cmp.Equal(s.Inputs, []string{"http://xxx"})
	}))
}

// Requires a running astra.
func TestFetchCfg(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	astraCfg, err := apiHandler.FetchCfg()
	assert.NotEmpty(t, astraCfg)
	assert.NoError(t, err)
}

// Requires a running astra.
func TestRequest(t *testing.T) {
	log := logger.New(logger.DebugLevel, os.Stderr)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	resp, err := apiHandler.request("POST", "/control/", loadReq{Cmd: "sessions"})
	assert.Contains(t, string(resp), "sessions")
	assert.NoError(t, err)
}
