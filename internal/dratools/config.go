package dratools

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	maxRecursiveNonRunXrefsEnv = "DRATOOLS_MAX_RECURSIVE_NON_RUN_XREFS"
	treeMaxDirectRunsEnv       = "DRATOOLS_TREE_MAX_DIRECT_RUNS"
	urlMaxDirectRunsEnv        = "DRATOOLS_URL_MAX_DIRECT_RUNS"
	sizeMaxDirectRunsEnv       = "DRATOOLS_SIZE_MAX_DIRECT_RUNS"
)

func maxRecursiveNonRunXrefs() (*int, error) {
	return positiveIntOrUnlimited(maxRecursiveNonRunXrefsEnv, 500)
}

func treeMaxDirectRuns() (*int, error) {
	return positiveIntOrUnlimited(treeMaxDirectRunsEnv, 200)
}

func urlMaxDirectRuns() (*int, error) {
	return positiveIntOrUnlimited(urlMaxDirectRunsEnv, 200)
}

func sizeMaxDirectRuns() (*int, error) {
	return positiveIntOrUnlimited(sizeMaxDirectRunsEnv, 200)
}

func positiveIntOrUnlimited(name string, def int) (*int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return &def, nil
	}
	if strings.EqualFold(value, "unlimited") {
		return nil, nil
	}
	i, err := strconv.Atoi(value)
	if err == nil && i > 0 {
		return &i, nil
	}
	return nil, newError("invalid_option", fmt.Sprintf("invalid %s '%s' (expected: positive integer or unlimited)", name, value))
}
