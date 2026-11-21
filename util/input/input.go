package input

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"m3u-merge-astra/util/logger"

	"github.com/cockroachdb/errors"
)

// AskYesNo returns true if user input is 'y' or 'Y' and false if 'n' or 'N'.
//
// If user types neither 'y', 'Y', 'n' or 'N', it asks again.
//
// If reading fails, e.g. `in` is closed, it logs the error and returns false.
//
// `in` is a reader to read data from. Usually it should be 'os.Stdin'.
func AskYesNo(log *logger.Logger, in io.Reader, prompt string) bool {
	answer, err := ask(in, true, prompt, func(input string) bool {
		switch input {
		case "y", "Y", "n", "N":
			return false
		}
		return true
	})
	if err != nil {
		log.Error(errors.Wrap(err, "Read from standard input"))
		return false
	}
	return strings.EqualFold(answer, "y")
}

// ask returns user input, preliminarily printing `prompt`.
//
// It asks again while `callback` returns true. Last line without trailing newline is passed to `callback` too.
//
// If reading fails before `callback` returns false, it returns the error.
//
// If `trim` is true, trim space from user input before passing it to `callback`.
//
// `in` is a reader to read data from. Usually it should be 'os.Stdin'.
func ask(in io.Reader, trim bool, prompt string, callback func(string) bool) (string, error) {
	// Single reader for every attempt, as it may buffer input beyond the current line.
	reader := bufio.NewReader(in)
	for {
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if trim {
			input = strings.TrimSpace(input)
		}
		if !callback(input) {
			return input, nil
		}
		if err != nil {
			return "", err
		}
	}
}
