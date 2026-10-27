package cfg

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
	"github.com/samber/lo"

	"m3u-merge-astra/util/logger"
	"m3u-merge-astra/util/simplify"
)

func init() {
	// Treat regexp.Regexp as its pattern string on encoding.
	yaml.RegisterCustomMarshaler(func(rx regexp.Regexp) ([]byte, error) {
		return []byte(rx.String()), nil
	})
}

// Root represents root settings of the program.
type Root struct {
	General General `yaml:"general"`
	M3U     M3U     `yaml:"m3u"`
	Streams Streams `yaml:"streams"`
}

// General represents general settings of the program.
type General struct {
	// FullTranslit specifies if name transliteration should be used to detect which M3U channel corresponds a stream.
	FullTranslit bool `yaml:"full_translit"`

	// FullTranslitMap represents source to destination character mapping.
	//
	// All symbols are lowercase as comparsion function will convert every character in a name to lowercase.
	//
	// Key: From. Value: To.
	FullTranslitMap map[string]string `yaml:"full_translit_map"`

	// SimilarTranslit specifies if name transliteration between visually similar characters should be used to detect
	// which M3U channel corresponds a stream.
	SimilarTranslit bool `yaml:"similar_translit"`

	// SimilarTranslitMap represents source to destination character mapping.
	//
	// All symbols are lowercase as comparsion function will convert every character in a name to lowercase.
	//
	// Key: From. Value: To.
	SimilarTranslitMap map[string]string `yaml:"similar_translit_map"`

	// NameAliases specifies if name aliases should be used to detect which M3U channel corresponds a stream.
	NameAliases bool `yaml:"name_aliases"`

	// NameAliasList represents the list of lists.
	//
	// Names defined here will be considered identical to any other name in the same nested group.
	//
	// During comparsion, names will be simplified (lowercase, no special characters except the '+' sign), but not
	// transliterated.
	NameAliasList [][]string `yaml:"name_alias_list"`

	// NameAliasList represents simplified version of NameAliasList.
	//
	// This field will not be included into config and used to improve performance of util/compare.IsNameSame().
	SimpleNameAliasList [][]string `yaml:"-"`

	// AstraAPIRespTimeout represents astra API response timeout.
	AstraAPIRespTimeout time.Duration `yaml:"astra_api_resp_timeout"`

	// MergeCategories specifies if duplicated categories should be removed with unique groups combined per category.
	MergeCategories bool `yaml:"merge_categories"`
}

// SimplifyAliases returns simplified alias list in `c`.
//
// Made to improve performance of util/compare.IsNameSame().
func (c General) SimplifyAliases() (out [][]string) {
	for _, set := range c.NameAliasList {
		out = append(out, lo.Map(set, func(alias string, _ int) string {
			return simplify.Name(alias)
		}))
	}
	return
}

// M3U represents M3U related settings of the program.
type M3U struct {
	// RespTimeout represents M3U playlist URL response timeout.
	RespTimeout time.Duration `yaml:"resp_timeout"`

	// ChannNameBlacklist represens the list of regular expressions.
	//
	// If any expression match name of a channel, this channel will be removed from M3U input before merging.
	ChannNameBlacklist []regexp.Regexp `yaml:"chann_name_blacklist"`

	// ChannGroupBlacklist represens the list of regular expressions.
	//
	// If any expression match group of a channel, this channel will be removed from M3U input before merging.
	//
	// It runs after replacing groups by ChannGroupMap so enter the appropriate values.
	ChannGroupBlacklist []regexp.Regexp `yaml:"chann_group_blacklist"`

	// ChannURLBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a channel, this channel will be removed from M3U input before merging.
	ChannURLBlacklist []regexp.Regexp `yaml:"chann_url_blacklist"`

	// ChannGroupMap represents invalid to valid M3U channel group mapping.
	//
	// Key: From. Value: To.
	ChannGroupMap map[string]string `yaml:"chann_group_map"`
}

// Streams represents astra streams related settings of the program.
type Streams struct {
	// AddedPrefix represents new stream name prefix.
	AddedPrefix string `yaml:"added_prefix"`

	// AddNew specifies if new astra streams should be added if streams does not contain M3U channel name.
	AddNew bool `yaml:"add_new"`

	// AddGroupsToNew specifies if groups should be added to new astra streams.
	AddGroupsToNew bool `yaml:"add_groups_to_new"`

	// GroupsCategoryForNew represents category name to use for groups of new astra streams.
	GroupsCategoryForNew string `yaml:"groups_category_for_new"`

	// AddNewWithKnownInputs specifies if new astra streams should be added if streams contain M3U channel URL.
	AddNewWithKnownInputs bool `yaml:"add_new_with_known_inputs"`

	// MakeNewEnabled specifies if new streams should be enabled.
	MakeNewEnabled bool `yaml:"make_new_enabled"`

	// NewType represents new stream type, can be one of two types:
	//
	// SPTS - Single-Program Transport Stream. Streaming channels to the end users over IP network.
	//
	// MPTS - Multi-Program Transport Stream. Preparing multiplexes to DVB modulators.
	NewType StreamType `yaml:"new_type"`

	// NewKeepActive represents delay before stop stream if no active connections for new streams.
	NewKeepActive int `yaml:"new_keep_active"`

	// DisabledPrefix represents disabled stream name prefix.
	DisabledPrefix string `yaml:"disabled_prefix"`

	// RemoveWithoutInputs specifies if streams without inputs should be removed.
	//
	// It has priority over DisableWithoutInputs.
	RemoveWithoutInputs bool `yaml:"remove_without_inputs"`

	// DisableWithoutInputs specifies if streams without inputs should be disabled.
	DisableWithoutInputs bool `yaml:"disable_without_inputs"`

	// EnableOnInputUpdate specifies if streams should be enabled if they got new inputs or inputs were updated
	// (but not removed).
	EnableOnInputUpdate bool `yaml:"enable_on_input_update"`

	// Rename specifies if astra streams should be renamed as M3U channels if their standartized names are equal.
	Rename bool `yaml:"rename"`

	// AddNewInputs specifies if new inputs of astra streams should be added if such found in M3U channels.
	AddNewInputs bool `yaml:"add_new_inputs"`

	// UniteInputs specifies if inputs of streams with the same names should be moved to the first stream found.
	UniteInputs bool `yaml:"unite_inputs"`

	// HashCheckOnAddNewInputs specifies if new inputs of astra streams should be added even if M3U channel and
	// stream input only differ by hash (everything after #).
	HashCheckOnAddNewInputs bool `yaml:"hash_check_on_add_new_inputs"`

	// SortInputs specifies if inputs of astra streams should be sorted.
	SortInputs bool `yaml:"sort_inputs"`

	// InputWeightToTypeMap represents Mapping of how high stream input should appear in the list after sorting.
	//
	// Any unspecified input will have weight of maximum - 1 (right before the last entry).
	InputWeightToTypeMap map[int]regexp.Regexp `yaml:"input_weight_to_type_map"`

	// UnknownInputWeight represents Default weight of unknown inputs.
	UnknownInputWeight int `yaml:"unknown_input_weight"`

	// InputBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a stream's input, this input will be removed from astra streams before adding new
	// ones.
	InputBlacklist []regexp.Regexp `yaml:"input_blacklist"`

	// RemoveDuplicatedInputs specifies if inputs of astra streams which already present in config should be
	// removed.
	RemoveDuplicatedInputs bool `yaml:"remove_duplicated_inputs"`

	// RemoveDuplicatedInputsByRxList represens the list of regular expressions.
	//
	// If any first capture group (anything surrounded by the first '()') of regular expression match URL of input of a
	// stream, any other inputs of that stream which first capture group is the same will be removed from stream.
	//
	// This setting is not controlled by 'remove_duplicated_inputs'.
	RemoveDuplicatedInputsByRxList []regexp.Regexp `yaml:"remove_duplicated_inputs_by_rx_list"`

	// RemoveDisabledInputs specifies if disabled inputs should be removed.
	RemoveDisabledInputs bool `yaml:"remove_disabled_inputs"`

	// DisableAllButOneInputByRxList represens the list of regular expressions.
	//
	// If any expression match URL of a stream's input, only this input will be kept and all other will be disabled.
	//
	// If list has multiple entries, only input matching first found expression will be kept.
	DisableAllButOneInputByRxList []regexp.Regexp `yaml:"disable_all_but_one_input_by_rx_list"`

	// RemoveDeadInputs specifies if inputs of astra streams which do not respond or give invalid response should be
	// removed.
	//
	// Supports HTTP(S), enable 'use_analyzer' option for more.
	//
	// It has priority over DisableDeadInputs.
	RemoveDeadInputs bool `yaml:"remove_dead_inputs"`

	// DisableDeadInputs specifies if inputs of astra streams which do not respond or give invalid response should be
	// disabled.
	//
	// Supports HTTP(S), enable 'use_analyzer' option for more.
	DisableDeadInputs bool `yaml:"disable_dead_inputs"`

	// DeadInputsCheckBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a stream's input, this input will not be checked for availability.
	DeadInputsCheckBlacklist []regexp.Regexp `yaml:"dead_inputs_check_blacklist"`

	// InputMaxConns represents maximum amount of simultaneous connections to validate inputs of astra streams.
	//
	// Use more than 1 with caution. It may result in false positives if server consider frequent requests as spam.
	InputMaxConns int `yaml:"input_max_conns"`

	// InputRespTimeout represents astra stream input response timeout.
	InputRespTimeout time.Duration `yaml:"input_resp_timeout"`

	// UseAnalyzer specifies if astra analyzer (astra --analyze -p `port`) should be used to check for dead inputs.
	//
	// Supports HTTP(S), UDP, RTP, RTSP.
	UseAnalyzer bool `yaml:"use_analyzer"`

	// AnalyzerAddr represents astra analyzer address in format of 'host:port'.
	AnalyzerAddr string `yaml:"analyzer_addr"`

	// AnalyzerWatchTime represents amount of time per attempt that astra analyzer should spend collecting results.
	AnalyzerWatchTime time.Duration `yaml:"analyzer_watch_time"`

	// AnalyzerMaxAttempts represents maximum amount of attempts that astra analyzer should perform trying to get
	// response from stream input.
	AnalyzerMaxAttempts int `yaml:"analyzer_max_attempts"`

	// AnalyzerWatchTime represents average bitrate threshold in kbit/s for stream inputs.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerBitrateThreshold int `yaml:"analyzer_bitrate_threshold"`

	// AnalyzerVideoOnlyBitrateThreshold represents average bitrate threshold in kbit/s for stream inputs without audio.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerVideoOnlyBitrateThreshold int `yaml:"analyzer_video_only_bitrate_threshold"`

	// AnalyzerAudioOnlyBitrateThreshold represents average bitrate threshold in kbit/s for stream inputs without video.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerAudioOnlyBitrateThreshold int `yaml:"analyzer_audio_only_bitrate_threshold"`

	// AnalyzerCCErrorsThreshold represents average amount of CC errors for stream inputs.
	//
	// If astra analyzer will return amount of CC errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerCCErrorsThreshold int `yaml:"analyzer_cc_errors_threshold"`

	// AnalyzerPCRErrorsThreshold represents average amount of PCR errors for stream inputs.
	//
	// If astra analyzer will return amount of PCR errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerPCRErrorsThreshold int `yaml:"analyzer_pcr_errors_threshold"`

	// AnalyzerPESErrorsThreshold represents average amount of PES errors for stream inputs.
	//
	// If astra analyzer will return amount of PES errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerPESErrorsThreshold int `yaml:"analyzer_pes_errors_threshold"`

	// InputUpdateMap represens list of regular expression pairs.
	//
	// If any `From` expression match URL of astra stream's input, it will be replaced with URL from according M3U
	// channel if it matches the `To` expression.
	//
	// In most cases specified `From` and `To` should be identical.
	//
	// Using InputBlacklist with AddNewInputs instead will have almost the same end result but since old
	// URL's will be removed beforehand, original hash (#...) will be lost. Also it will be less clear which input was
	// being replaced.
	InputUpdateMap []UpdateRecord `yaml:"input_update_map"`

	// UpdateInputs specifies if inputs of astra streams should be updated with M3U channels according to
	// InputUpdateMap.
	UpdateInputs bool `yaml:"update_inputs"`

	// KeepInputHash specifies if old URL hash should be kept on updating inputs of astra streams.
	KeepInputHash bool `yaml:"keep_input_hash"`

	// RemoveInputsByUpdateMap specifies if inputs of astra streams which match at least one InputUpdateMap.From
	// expression but not in M3U channels should be removed.
	RemoveInputsByUpdateMap bool `yaml:"remove_inputs_by_update_map"`

	// NameToInputHashMap represents mapping of stream name regular expression to stream input hash which should
	// be added.
	NameToInputHashMap []HashAddRule `yaml:"name_to_input_hash_map"`

	// GroupToInputHashMap represents mapping of stream group regular expression to stream input hash which should
	// be added.
	// Stream groups should be defined to match expressions in the form of 'Category: Group'.
	GroupToInputHashMap []HashAddRule `yaml:"group_to_input_hash_map"`

	// InputToInputHashMap represents mapping of stream input regular expression to stream input hash which should
	// be added.
	InputToInputHashMap []HashAddRule `yaml:"input_to_input_hash_map"`

	// NameToKeepActiveMap represents mapping of stream name regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	NameToKeepActiveMap []KeepActiveAddRule `yaml:"name_to_keep_active_map"`

	// GroupToKeepActiveMap represents mapping of stream group regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	GroupToKeepActiveMap []KeepActiveAddRule `yaml:"group_to_keep_active_map"`

	// InputToKeepActiveMap represents mapping of stream input regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	//
	// Setting will be set if at least one input matches the `By` expression.
	InputToKeepActiveMap []KeepActiveAddRule `yaml:"input_to_keep_active_map"`
}

// UpdateRecord represents astra stream input update rule.
type UpdateRecord struct {
	From regexp.Regexp `yaml:"from"`
	To   regexp.Regexp `yaml:"to"`
}

// HashAddRule represents astra stream input hash adding rule.
type HashAddRule struct {
	By   regexp.Regexp `yaml:"by"`
	Hash string        `yaml:"hash"`
}

// KeepActiveAddRule represents astra stream 'keep active' adding rule.
type KeepActiveAddRule struct {
	By         regexp.Regexp `yaml:"by"`
	KeepActive int           `yaml:"keep_active"`
}

// StreamType represents astra stream type.
type StreamType string

const (
	SPTS StreamType = "spts"
	MPTS StreamType = "mpts"
)

// BadRegexpError represents error thrown if program config has invalid regular expression.
type BadRegexpError struct {
	Reason string
	Regexp regexp.Regexp
}

// Error is used to satisfy golang error interface.
func (e BadRegexpError) Error() string {
	return fmt.Sprintf("%v; Regular expression: %v", e.Reason, e.Regexp.String())
}

// Init returns config instance and false if config at `cfgFilePath` already exist.
//
// If config does not exist, creates a default, returns empty instance and true.
//
// Builds simplified version of name aliases to Root.General.SimpleNameAliasList.
//
// Can return errors defined in this package: BadRegexpError.
func Init(log *logger.Logger, cfgFilePath string) (Root, bool, error) {
	log.Info("Reading program config")

	var root Root
	commentMap := yaml.CommentMap{}
	defCfg := NewDefCfg()
	defComments := defCommentMap()

	readConfig := func() error {
		bytes, err := os.ReadFile(cfgFilePath)
		if err != nil {
			return err
		}
		return errors.Wrap(yaml.UnmarshalWithOptions(bytes, &root, yaml.CommentToMap(commentMap)),
			"Decode config file")
	}

	writeConfig := func(cfg Root, comments yaml.CommentMap) error {
		bytes, err := yaml.MarshalWithOptions(cfg, yaml.WithComment(comments),
			yaml.UseLiteralStyleIfMultiline(true), yaml.UseSingleQuote(true))
		if err != nil {
			return errors.Wrap(err, "Encode config file")
		}
		return os.WriteFile(cfgFilePath, bytes, 0644)
	}

	validateConfig := func() error {
		for _, rx := range root.Streams.RemoveDuplicatedInputsByRxList {
			if rx.NumSubexp() < 1 {
				msg := "Expecting at least one capture group"
				return errors.Wrap(BadRegexpError{Regexp: rx, Reason: msg}, "Validate config")
			}
		}
		return nil
	}

	// Read config or create a new if not exist.
	if err := readConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info("Config file not found, creating a default")
			if err := writeConfig(defCfg, defComments); err != nil {
				return root, false, errors.Wrap(err, "Write default config")
			}
			return root, true, nil
		} else {
			return root, false, errors.Wrap(err, "Read config")
		}
	}

	if err := validateConfig(); err != nil {
		return root, false, errors.Wrap(err, "Validate config")
	}

	// Build simple aliases list.
	if root.General.NameAliases {
		root.General.SimpleNameAliasList = root.General.SimplifyAliases()
	}

	return root, false, nil
}

// DefFullTranslitMap returns default full transliteration map.
//
// Used in tests.
func DefFullTranslitMap() map[string]string {
	return map[string]string{
		"а": "a", "б": "b", "в": "v", "г": "g", "д": "d", "е": "e", "ё": "yo", "ж": "zh", "з": "z", "и": "i",
		"й": "j", "к": "k", "л": "l", "м": "m", "н": "n", "о": "o", "п": "p", "р": "r", "с": "s", "т": "t",
		"у": "u", "ф": "f", "х": "h", "ц": "c", "ч": "ch", "ш": "sh", "щ": "shh", "ъ": "", "ы": "y", "ь": "",
		"э": "eh", "ю": "yu", "я": "ya",
	}
}

// DefSimilarTranslitMap returns default similar transliteration map.
//
// Used in tests.
func DefSimilarTranslitMap() map[string]string {
	return map[string]string{
		"а": "a", "б": "6", "в": "b", "е": "e", "з": "3", "к": "k", "м": "m", "н": "h", "о": "o", "р": "p",
		"с": "c", "т": "t", "у": "y", "х": "x",
	}
}

// NewDefCfg returns default config.
func NewDefCfg() Root {
	return Root{
		General: General{
			FullTranslit:        true,
			FullTranslitMap:     DefFullTranslitMap(),
			SimilarTranslit:     true,
			SimilarTranslitMap:  DefSimilarTranslitMap(),
			NameAliases:         true,
			NameAliasList:       [][]string{},
			SimpleNameAliasList: [][]string(nil),
			AstraAPIRespTimeout: time.Second * 10,
			MergeCategories:     false,
		},
		M3U: M3U{
			RespTimeout:         time.Second * 10,
			ChannNameBlacklist:  []regexp.Regexp{},
			ChannGroupBlacklist: []regexp.Regexp{},
			ChannURLBlacklist:   []regexp.Regexp{},
			ChannGroupMap:       map[string]string{},
		},
		Streams: Streams{
			AddedPrefix:                       "_ADDED: ",
			AddNew:                            true,
			AddGroupsToNew:                    false,
			GroupsCategoryForNew:              "All",
			AddNewWithKnownInputs:             false,
			MakeNewEnabled:                    false,
			NewType:                           SPTS,
			NewKeepActive:                     0,
			DisabledPrefix:                    "_DISABLED: ",
			RemoveWithoutInputs:               false,
			DisableWithoutInputs:              true,
			EnableOnInputUpdate:               false,
			Rename:                            false,
			AddNewInputs:                      true,
			UniteInputs:                       true,
			HashCheckOnAddNewInputs:           false,
			SortInputs:                        true,
			InputWeightToTypeMap:              map[int]regexp.Regexp{},
			UnknownInputWeight:                50,
			InputBlacklist:                    []regexp.Regexp{},
			RemoveDuplicatedInputs:            true,
			RemoveDuplicatedInputsByRxList:    []regexp.Regexp{},
			RemoveDisabledInputs:              false,
			DisableAllButOneInputByRxList:     []regexp.Regexp{},
			RemoveDeadInputs:                  false,
			DisableDeadInputs:                 false,
			DeadInputsCheckBlacklist:          []regexp.Regexp{},
			InputMaxConns:                     1,
			InputRespTimeout:                  time.Second * 10,
			UseAnalyzer:                       false,
			AnalyzerAddr:                      "127.0.0.1:8001",
			AnalyzerWatchTime:                 time.Second * 20,
			AnalyzerMaxAttempts:               3,
			AnalyzerBitrateThreshold:          1,
			AnalyzerVideoOnlyBitrateThreshold: 1,
			AnalyzerAudioOnlyBitrateThreshold: 1,
			AnalyzerCCErrorsThreshold:         -1,
			AnalyzerPCRErrorsThreshold:        -1,
			AnalyzerPESErrorsThreshold:        -1,
			InputUpdateMap:                    []UpdateRecord{},
			UpdateInputs:                      false,
			KeepInputHash:                     true,
			RemoveInputsByUpdateMap:           false,
			NameToInputHashMap:                []HashAddRule{},
			GroupToInputHashMap:               []HashAddRule{},
			InputToInputHashMap:               []HashAddRule{},
			NameToKeepActiveMap:               []KeepActiveAddRule{},
			GroupToKeepActiveMap:              []KeepActiveAddRule{},
			InputToKeepActiveMap:              []KeepActiveAddRule{},
		},
	}
}

// defCommentMap returns comments for the default config written by Init().
//
//nolint:lll // Human-readable YAML comments cannot be wrapped without changing their text.
func defCommentMap() yaml.CommentMap {
	return yaml.CommentMap{
		"$": []*yaml.Comment{
			yaml.HeadComment(
				" For details, see https://github.com/SCP002/m3u-merge-astra/blob/master/README.md#program-config-settings",
			),
		},
		"$.general": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" -------------------------------------------------------------------------------------------------------------------",
				" General settings of the program.",
			),
		},
		"$.general.full_translit": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Use name transliteration to detect which M3U channel corresponds a stream?",
			),
		},
		"$.general.full_translit_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Source to destination character mapping.",
				" All symbols are lowercase as comparsion function will convert every character in a name to lowercase.",
				" Key: From. Value: To.",
			),
		},
		"$.general.similar_translit": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Use name transliteration between visually similar characters to detect",
				" which M3U channel corresponds a stream?",
			),
		},
		"$.general.similar_translit_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Source to destination character mapping.",
				" All symbols are lowercase as comparsion function will convert every character in a name to lowercase.",
				" Key: From. Value: To.",
			),
		},
		"$.general.name_aliases": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Use name aliases list to detect which M3U channel corresponds a stream?",
			),
		},
		"$.general.name_alias_list": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of lists.",
				" Names defined here will be considered identical to any other name in the same nested group.",
				" During comparsion, names will be simplified (lowercase, no special characters except the '+' sign),",
				" but not transliterated.",
				"",
				" - - 'Sample'",
				"   - 'Sample TV'",
				"   - 'Sample Television Channel'",
				" - - 'Discovery ID'",
				"   - 'Discovery Investigation'",
			),
		},
		"$.general.astra_api_resp_timeout": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Astra API response timeout.",
			),
		},
		"$.general.merge_categories": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Should duplicated categories be removed with unique groups combined per category?",
			),
		},
		"$.m3u": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" -------------------------------------------------------------------------------------------------------------------",
				" M3U related settings of the program.",
			),
		},
		"$.m3u.resp_timeout": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" M3U playlist URL response timeout in seconds.",
			),
		},
		"$.m3u.chann_name_blacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match name of a channel, this channel will be removed from M3U input before merging.",
				"",
				" - 'Nonsense TV'",
				" - '(?i)^Test$'",
			),
		},
		"$.m3u.chann_group_blacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match group of a channel, this channel will be removed from M3U input before merging.",
				" It runs after replacing groups by 'chann_group_map' so enter the appropriate values.",
				"",
				" - '18\\+'",
				" - '(?i)^cooking$'",
			),
		},
		"$.m3u.chann_url_blacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match URL of a channel, this channel will be removed from M3U input before merging.",
				"",
				" - 'https?:\\/\\/filter_me\\.com'",
				" - '192\\.168\\.88\\.14\\/play'",
			),
		},
		"$.m3u.chann_group_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Invalid to valid M3U channel group mapping.",
				" Key: From. Value: To.",
				"",
				" '': 'General'",
				" '-': 'General'",
				" 'For kids': 'Kids'",
			),
		},
		"$.streams": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" -------------------------------------------------------------------------------------------------------------------",
				" Astra streams related settings of the program.",
			),
		},
		"$.streams.added_prefix": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" New stream name prefix.",
				" Safe to set to ''.",
			),
		},
		"$.streams.add_new": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Add new astra streams if streams does not contain M3U channel name?",
			),
		},
		"$.streams.add_groups_to_new": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Add groups to new astra streams?",
			),
		},
		"$.streams.groups_category_for_new": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Category name to use for groups of new astra streams.",
			),
		},
		"$.streams.add_new_with_known_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Add new astra streams if streams contain M3U channel URL?",
			),
		},
		"$.streams.make_new_enabled": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Make new streams enabled?",
			),
		},
		"$.streams.new_type": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" New stream type, can be one of two types:",
				" spts - Single-Program Transport Stream. Streaming channels to the end users over IP network.",
				" mpts - Multi-Program Transport Stream. Preparing multiplexes to DVB modulators.",
			),
		},
		"$.streams.new_keep_active": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Delay before stop stream if no active connections for new streams.",
			),
		},
		"$.streams.disabled_prefix": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Disabled stream name prefix.",
				" Safe to set to ''.",
			),
		},
		"$.streams.remove_without_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove streams without inputs?",
				" It has priority over 'disable_without_inputs'.",
			),
		},
		"$.streams.disable_without_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Disable streams without inputs?",
			),
		},
		"$.streams.enable_on_input_update": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Enable streams if they got new inputs or inputs were updated (but not removed)?",
			),
		},
		"$.streams.rename": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Rename astra streams as M3U channels if their standartized names are equal?",
			),
		},
		"$.streams.add_new_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Add new inputs to astra streams if such found in M3U channels?",
			),
		},
		"$.streams.unite_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Move inputs of streams with the same names to the first stream found?",
			),
		},
		"$.streams.hash_check_on_add_new_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Add new inputs to astra streams even if M3U channel and stream input only differ by hash (everything after #)?",
			),
		},
		"$.streams.sort_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Sort inputs of astra streams?",
			),
		},
		"$.streams.input_weight_to_type_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of how high stream input should appear in the list after sorting.",
				" Any unspecified input will have weight of 'unknown_input_weight'.",
				"",
				" 1: '192.\\168\\.88\\.' # Always first.",
				" 99: 'least_reliable\\.tv' # Always last.",
			),
		},
		"$.streams.unknown_input_weight": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Default weight of unknown inputs.",
			),
		},
		"$.streams.input_blacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match URL of a stream's input, this input will be removed from astra streams before adding",
				" new ones.",
				"",
				" - 'https?:\\/\\/filter_me\\.com'",
				" - '192\\.168\\.88\\.14\\/play'",
			),
		},
		"$.streams.remove_duplicated_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove inputs of astra streams which already present in config?",
			),
		},
		"$.streams.remove_duplicated_inputs_by_rx_list": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any first capture group (anything surrounded by the first '()') of regular expression match URL of input of",
				" a stream, any other inputs of that stream which first capture group is the same will be removed from stream.",
				"",
				" This setting is not controlled by 'remove_duplicated_inputs'.",
				"",
				" - '^.*:\\/\\/([^#?/]*)' # By host",
				" - '^.*:\\/\\/.*?\\/([^#?]*)' # By path",
			),
		},
		"$.streams.remove_disabled_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove disabled inputs?",
			),
		},
		"$.streams.disable_all_but_one_input_by_rx_list": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match URL of a stream's input, only this input will be kept and all other will be disabled.",
				" If list has multiple entries, only input matching first found expression will be kept.",
				"",
				" - '[#&]no_sync(&|$)'",
			),
		},
		"$.streams.remove_dead_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove inputs of astra streams which do not respond or give invalid response?",
				" Supports HTTP(S), enable 'use_analyzer' option for more.",
				" It has priority over 'disable_dead_inputs'.",
			),
		},
		"$.streams.disable_dead_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Disable inputs of astra streams which do not respond or give invalid response?",
				" Supports HTTP(S), enable 'use_analyzer' option for more.",
			),
		},
		"$.streams.dead_inputs_check_blacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expressions.",
				" If any expression match URL of a stream's input, this input will not be checked for availability.",
				"",
				" - 'https?:\\/\\/dont-check\\.com\\/play'",
				" - '192\\.168\\.88\\.'",
			),
		},
		"$.streams.input_max_conns": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Maximum amount of simultaneous connections to validate inputs of astra streams.",
				" Use more than 1 with caution. It may result in false positives if server consider frequent requests as spam.",
			),
		},
		"$.streams.input_resp_timeout": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Astra stream input response timeout.",
			),
		},
		"$.streams.use_analyzer": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Use astra analyzer (astra --analyze -p <port>) to check for dead inputs?",
				"",
				" Supports HTTP(S), UDP, RTP, RTSP.",
			),
		},
		"$.streams.analyzer_addr": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Astra analyzer address in format of 'host:port'.",
			),
		},
		"$.streams.analyzer_watch_time": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Amount of time per attempt that astra analyzer should spend collecting results.",
			),
		},
		"$.streams.analyzer_max_attempts": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Maximum amount of attempts that astra analyzer should perform trying to get response from stream input.",
			),
		},
		"$.streams.analyzer_bitrate_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average bitrate threshold in kbit/s for stream inputs.",
				"",
				" If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			),
		},
		"$.streams.analyzer_video_only_bitrate_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average bitrate threshold in kbit/s for stream inputs without audio.",
				"",
				" If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			),
		},
		"$.streams.analyzer_audio_only_bitrate_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average bitrate threshold in kbit/s for stream inputs without video.",
				"",
				" If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			),
		},
		"$.streams.analyzer_cc_errors_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average amount of CC errors for stream inputs.",
				"",
				" If astra analyzer will return amount of CC errors higher than specified threshold, input will be cosidered",
				" dead.",
				"",
				" Set to negative value to disable this check.",
			),
		},
		"$.streams.analyzer_pcr_errors_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average amount of PCR errors for stream inputs.",
				"",
				" If astra analyzer will return amount of PCR errors higher than specified threshold, input will be cosidered",
				" dead.",
				"",
				" Set to negative value to disable this check.",
			),
		},
		"$.streams.analyzer_pes_errors_threshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Average amount of PES errors for stream inputs.",
				"",
				" If astra analyzer will return amount of PES errors higher than specified threshold, input will be cosidered",
				" dead.",
				"",
				" Set to negative value to disable this check.",
			),
		},
		"$.streams.input_update_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" List of regular expression pairs.",
				" If any 'from' expression match URL of astra stream's input, it will be replaced with URL from according M3U",
				" channel if it matches the 'to' expression.",
				" Only first matching input will be updated per M3U channel.",
				"",
				" In most cases specified 'from' and 'to' should be identical.",
				"",
				" - from: '127\\.0\\.0\\.1'",
				"   to: '127\\.0\\.0\\.1'",
				" - from: 'some_url\\.com'",
				"   to: 'some_url\\.com'",
			),
		},
		"$.streams.update_inputs": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Update inputs of astra streams with M3U channels according to 'input_update_map'?",
			),
		},
		"$.streams.keep_input_hash": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Keep old input hash on updating inputs of astra streams?",
			),
		},
		"$.streams.remove_inputs_by_update_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove inputs of astra streams which match at least one 'input_update_map.from' expression but not in M3U",
				" channels?",
			),
		},
		"$.streams.name_to_input_hash_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream name regular expression to stream input hash which should be added.",
				"",
				" - by: '[- _]HD$'",
				"   hash: 'buffer_time=10'",
				" - by: '[- _]FM$'",
				"   hash: 'no_sync'",
			),
		},
		"$.streams.group_to_input_hash_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream group regular expression to stream input hash which should be added.",
				" Stream groups should be defined to match expressions in the form of 'Category: Group'.",
				"",
				" - by: '(?i)All: HD Channels$'",
				"   hash: 'buffer_time=10'",
				" - by: '(?i).*RADIO$'",
				"   hash: 'no_sync'",
			),
		},
		"$.streams.input_to_input_hash_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream input regular expression to stream input hash which should be added.",
				"",
				" - by: ':8080'",
				"   hash: 'ua=VLC/3.0.9 LibVLC/3.0.9'",
				" - by: '^rts?p:\\/\\/'",
				"   hash: 'no_reload'",
			),
		},
		"$.streams.name_to_keep_active_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream name regular expression to 'keep active' setting of stream which should be set.",
				"",
				" Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
				"",
				" - by: '[- _]HD$'",
				"   keep_active: 10",
				" - by: '[- _]FM$'",
				"   keep_active: 0",
			),
		},
		"$.streams.group_to_keep_active_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream group regular expression to 'keep active' setting of stream which should be set.",
				"",
				" Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
				"",
				" - by: '(?i)All: HD Channels$'",
				"   keep_active: 10",
				" - by: '(?i).*RADIO$'",
				"   keep_active: 0",
			),
		},
		"$.streams.input_to_keep_active_map": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Mapping of stream input regular expression to 'keep active' setting of stream which should be set.",
				"",
				" Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
				"",
				" Setting will be set if at least one input matches the 'by' expression.",
				"",
				" - by: ':8080'",
				"   keep_active: 10",
				" - by: '^rts?p:\\/\\/'",
				"   keep_active: 0",
			),
		},
	}
}
