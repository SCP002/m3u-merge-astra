package main

import (
	"fmt"
	"os"
	"time"

	"m3u-merge-astra/astra"
	"m3u-merge-astra/astra/analyzer"
	"m3u-merge-astra/astra/api"
	"m3u-merge-astra/cfg"
	"m3u-merge-astra/cli"
	"m3u-merge-astra/m3u"
	"m3u-merge-astra/merge"
	"m3u-merge-astra/updater"
	"m3u-merge-astra/util/input"
	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/network"
	"m3u-merge-astra/util/slice"
	"m3u-merge-astra/version"

	"github.com/adampresley/sigint"
	"github.com/cockroachdb/errors"
	goFlags "github.com/jessevdk/go-flags"
	"github.com/utahta/go-openuri"
)

func main() {
	// Init default logger.
	log := logger.New(logger.FatalLevel, os.Stderr)

	// Parse command line arguments.
	flags, err := cli.Parse()
	if flags.Version {
		fmt.Println(version.Version)
		os.Exit(0)
	}
	if cli.IsErrOfType(err, goFlags.ErrHelp) {
		// Help message will be prined by go-flags.
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Setup logger.
	log.SetLevel(flags.LogLevel)
	logFile, err := log.AddFileWriter(flags.LogFile)
	if err == nil {
		// Closing nil file does not panic.
		defer func() { _ = logFile.Close() }()
	} else {
		log.Error(err)
	}

	// Register SIGINT and SIGTERM event handler.
	sigint.Listen(func() {
		log.Info("SIGINT or SIGTERM signal received, shutting down")
		os.Exit(0)
	})

	if err := run(log, flags); err != nil {
		log.Fatal(err)
	}
}

// run runs the program configured with command line `flags`, logging to `log`.
func run(log *logger.Logger, flags cli.Flags) error {
	// Update the program.
	if flags.Update {
		updaterHttpClient := network.NewVerifyingHTTPClient(time.Minute * 10)
		updater := updater.New(log, updaterHttpClient)
		return errors.Wrap(updater.Update(), "Self update failed")
	}

	log.InfoFi("Running with", "program config path", flags.ProgramCfgPath, "M3U path", flags.M3UPath, "astra address",
		flags.AstraAddr)

	// Read program config.
	cfg, isNewCfg, err := cfg.Init(log, flags.ProgramCfgPath)
	if err != nil {
		return err
	}
	if isNewCfg {
		log.InfoFi("New config is written, please verify it and start this program again", "path", flags.ProgramCfgPath)
		return nil
	}

	// Fetch astra config.
	log.Info("Fetching astra config")
	apiHttpClient := network.NewHTTPClient(cfg.General.AstraAPIRespTimeout)
	apiHandler := api.NewHandler(log, apiHttpClient, flags.AstraAddr, flags.AstraUser, flags.AstraPwd)
	astraCfg, err := apiHandler.FetchCfg()
	if err != nil {
		return err
	}

	// Fetch M3U channels.
	log.Info("Fetching M3U channels")
	m3uHttpClient := network.NewHTTPClient(cfg.M3U.RespTimeout)
	m3uResp, err := openuri.Open(flags.M3UPath, openuri.WithHTTPClient(m3uHttpClient))
	if err != nil {
		return err
	}
	defer func() { _ = m3uResp.Close() }()

	// Parse and preprocess M3U channels.
	m3uProcessor := m3u.NewProcessor(log, cfg)

	m3uChannels := m3uProcessor.Parse(m3uResp)
	m3uChannels = m3uProcessor.Sort(m3uChannels)
	if len(cfg.M3U.ChannGroupMap) > 0 {
		m3uChannels = m3uProcessor.ReplaceGroups(m3uChannels)
	}
	if !slice.IsAllEmpty(cfg.M3U.ChannNameBlacklist, cfg.M3U.ChannGroupBlacklist, cfg.M3U.ChannURLBlacklist) {
		m3uChannels = m3uProcessor.RemoveBlocked(m3uChannels)
	}

	// Update astra streams with data from M3U channels and run extra operations such as sorting or disabling streams
	// without inputs.
	astraProcessor := astra.NewProcessor(log, cfg)
	mergeProcessor := merge.NewProcessor(log, cfg)

	// Processing does not modify its arguments, so astraCfg keeps original streams and categories to find changes.
	modifiedStreams := astraCfg.Streams
	gid := astraCfg.GID
	modifiedStreams = astraProcessor.RemoveNamePrefixes(modifiedStreams)
	modifiedStreams = astraProcessor.Sort(modifiedStreams)
	if cfg.Streams.Rename {
		modifiedStreams = mergeProcessor.RenameStreams(modifiedStreams, m3uChannels)
	}
	if len(cfg.Streams.InputBlacklist) > 0 {
		modifiedStreams = astraProcessor.RemoveBlockedInputs(modifiedStreams)
	}
	if cfg.Streams.RemoveDuplicatedInputs {
		modifiedStreams = astraProcessor.RemoveDuplicatedInputs(modifiedStreams)
	}
	if len(cfg.Streams.RemoveDuplicatedInputsByRxList) > 0 {
		modifiedStreams = astraProcessor.RemoveDuplicatedInputsByRx(modifiedStreams)
	}
	if cfg.Streams.RemoveDisabledInputs {
		modifiedStreams = astraProcessor.RemoveDisabledInputs(modifiedStreams)
	}
	if cfg.Streams.UpdateInputs {
		modifiedStreams = mergeProcessor.UpdateInputs(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.RemoveInputsByUpdateMap {
		modifiedStreams = mergeProcessor.RemoveInputsByUpdateMap(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.AddNewInputs {
		modifiedStreams = mergeProcessor.AddNewInputs(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.UniteInputs {
		modifiedStreams = astraProcessor.UniteInputs(modifiedStreams)
	}
	if cfg.Streams.SortInputs {
		modifiedStreams = astraProcessor.SortInputs(modifiedStreams)
	}
	if cfg.Streams.AddNew {
		modifiedStreams, gid = mergeProcessor.AddNewStreams(modifiedStreams, m3uChannels, gid)
	}
	if cfg.Streams.RemoveDeadInputs {
		httpClient := network.NewHTTPClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(log, cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		modifiedStreams = astraProcessor.RemoveDeadInputs(httpClient, analyzer, modifiedStreams)
	} else if cfg.Streams.DisableDeadInputs {
		httpClient := network.NewHTTPClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(log, cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		modifiedStreams = astraProcessor.DisableDeadInputs(httpClient, analyzer, modifiedStreams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToInputHashMap, cfg.Streams.GroupToInputHashMap,
		cfg.Streams.InputToInputHashMap) {
		modifiedStreams = astraProcessor.AddHashes(modifiedStreams)
	}
	if len(cfg.Streams.DisableAllButOneInputByRxList) > 0 {
		modifiedStreams = astraProcessor.DisableAllButOneInputByRx(modifiedStreams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToKeepActiveMap, cfg.Streams.GroupToKeepActiveMap,
		cfg.Streams.InputToKeepActiveMap) {
		modifiedStreams = astraProcessor.SetKeepActive(modifiedStreams)
	}
	if cfg.Streams.RemoveWithoutInputs {
		modifiedStreams = astraProcessor.RemoveWithoutInputs(modifiedStreams)
	} else if cfg.Streams.DisableWithoutInputs {
		modifiedStreams = astraProcessor.DisableWithoutInputs(modifiedStreams)
	}
	modifiedStreams = astraProcessor.AddNamePrefixes(modifiedStreams)

	// Update astra categories.
	modifiedCats := astraCfg.Categories
	if cfg.General.MergeCategories {
		modifiedCats = astraProcessor.MergeCategories(modifiedCats)
	}
	modifiedCats = astraProcessor.UpdateCategories(modifiedCats, modifiedStreams)

	// Search for changes.
	changedCatMap := astraProcessor.ChangedCategories(astraCfg.Categories, modifiedCats)
	changedStreams := astraProcessor.ChangedStreams(astraCfg.Streams, modifiedStreams)

	// Sending changes to astra.
	sendChangesAllowed := true
	if !flags.Noninteractive {
		sendChangesAllowed = input.AskYesNo(log, os.Stdin, "Send changes to astra (Y/N)? ")
	}
	if sendChangesAllowed {
		apiHandler.SetCategories(changedCatMap)
		apiHandler.SetStreams(changedStreams, gid)
	}

	log.Info("Done")

	return nil
}
