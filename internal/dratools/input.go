package dratools

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func collectAccessions(args []string, inputPath string, stdin *os.File) ([]string, error) {
	values := make([]string, 0, len(args))
	for _, arg := range args {
		if acc := strings.ToUpper(strings.TrimSpace(arg)); acc != "" {
			values = append(values, acc)
		}
	}

	var r io.Reader
	switch {
	case inputPath == "-":
		r = stdin
	case inputPath != "":
		file, err := os.Open(inputPath)
		if err != nil {
			return nil, newError("input_file", "--input: "+err.Error())
		}
		defer file.Close()
		r = file
	case stdin != nil && !stdinIsTTY(stdin):
		r = stdin
	}
	if r != nil {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if acc := strings.ToUpper(strings.TrimSpace(scanner.Text())); acc != "" {
				values = append(values, acc)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, wrapError("input_file", "--input", err)
		}
	}
	values = uniqueStrings(values)
	if len(values) == 0 {
		return nil, newError("missing_accession", "ACCESSION is required")
	}
	return values, nil
}

func stdinIsTTY(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return true
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
