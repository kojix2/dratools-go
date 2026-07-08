package dratools

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type app struct {
	resolver   *AccessionResolver
	downloader *DownloadService
	stdin      *os.File
	stdout     io.Writer
	stderr     io.Writer
}

type reportedErrors struct{}

func (reportedErrors) Error() string { return "one or more accessions failed" }

func Main(argv []string, stdin *os.File, stdout, stderr io.Writer) int {
	downloader := NewDownloadService()
	downloader.ProgressOutput = stderr
	downloader.ProgressEnabled = interactiveWriter(stderr)
	a := &app{
		resolver:   NewAccessionResolver(NewDDBJClient("")),
		downloader: downloader,
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
	}
	return a.run(argv)
}

func (a *app) run(argv []string) int {
	if len(argv) == 0 {
		a.printHelp(a.stderr)
		return 1
	}
	name := argv[0]
	switch name {
	case "-h", "--help", "help":
		a.printHelp(a.stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(a.stdout, Version)
		return 0
	}
	aliases := map[string]string{"run": "runs", "urls": "url", "sizes": "size", "trees": "tree"}
	if canonical, ok := aliases[name]; ok {
		name = canonical
	}
	var err error
	switch name {
	case "url":
		err = a.runURL(argv[1:])
	case "runs":
		err = a.runRuns(argv[1:])
	case "tree":
		err = a.runTree(argv[1:])
	case "meta":
		err = a.runMeta(argv[1:])
	case "size":
		err = a.runSize(argv[1:])
	case "probe":
		err = a.runProbe(argv[1:])
	case "get":
		err = a.runGet(argv[1:])
	default:
		fmt.Fprintf(a.stderr, "%s unknown command '%s' (expected: url, get, probe, tree, meta, runs, size)\n", MessagePrefix, argv[0])
		return 1
	}
	if err != nil {
		if _, ok := err.(reportedErrors); ok {
			return 1
		}
		fmt.Fprintf(a.stderr, "%s %s: %v\n", MessagePrefix, name, err)
		return 1
	}
	return 0
}

func (a *app) reportAccessionError(command, accession string, err error) {
	fmt.Fprintf(a.stderr, "%s %s: %s: %v\n", MessagePrefix, command, strings.ToUpper(accession), err)
}

func (a *app) printHelp(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s <command> [options] [ACCESSION ...]\n\n", Name)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  url     show download URLs")
	fmt.Fprintln(w, "  get     download files")
	fmt.Fprintln(w, "  probe   check download URLs")
	fmt.Fprintln(w, "  tree    show traversal tree")
	fmt.Fprintln(w, "  meta    show metadata")
	fmt.Fprintln(w, "  runs    list run accessions")
	fmt.Fprintln(w, "  size    summarize download sizes")
}

type commonOptions struct {
	fileType string
	input    string
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func addCommonFlags(fs *flag.FlagSet, c *commonOptions) {
	fs.StringVar(&c.fileType, "type", FileTypeSRA, "target file type: sra, fastq, all")
	fs.StringVar(&c.input, "input", "", "read accession list from file or '-'")
	fs.StringVar(&c.input, "i", "", "read accession list from file or '-'")
}

func validateCommon(c commonOptions) error {
	switch c.fileType {
	case FileTypeSRA, FileTypeFASTQ, FileTypeAll:
		return nil
	default:
		return newError("invalid_option", "invalid --type '"+c.fileType+"' (expected: sra, fastq, all)")
	}
}

func validateProtocol(protocol string) error {
	if protocol == "https" || protocol == "ftp" {
		return nil
	}
	return newError("invalid_option", "invalid --protocol '"+protocol+"' (expected: https, ftp)")
}

func (a *app) accessions(fs *flag.FlagSet, opts commonOptions) ([]string, error) {
	return collectAccessions(fs.Args(), opts.input, a.stdin)
}

func (a *app) runURL(argv []string) error {
	var opts commonOptions
	var protocol string
	var asJSON, asTSV bool
	fs := newFlagSet("url", a.stderr)
	addCommonFlags(fs, &opts)
	fs.StringVar(&protocol, "protocol", "https", "preferred URL protocol: https, ftp")
	fs.BoolVar(&asJSON, "json", false, "print JSON")
	fs.BoolVar(&asTSV, "tsv", false, "print TSV")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	var buffer []DownloadCandidate
	headerPrinted := false
	failed := 0
accessionLoop:
	for _, accession := range accessions {
		record, err := a.fetchRecordWithDirectRunLimit(accession, "url")
		if err != nil {
			failed++
			a.reportAccessionError("url", accession, err)
			continue
		}
		downloads, err := a.resolver.ResolveDownloadsFromRecord(accession, record, opts.fileType)
		if err != nil {
			failed++
			a.reportAccessionError("url", accession, err)
			continue
		}
		if asJSON {
			buffer = append(buffer, downloads...)
			continue
		}
		if asTSV && !headerPrinted {
			fmt.Fprintln(a.stdout, "#run_accession\ttype\turl\tsize\tmd5")
			headerPrinted = true
		}
		for _, download := range downloads {
			raw, err := download.URLForProtocol(protocol)
			if err != nil {
				failed++
				a.reportAccessionError("url", accession, err)
				continue accessionLoop
			}
			if asTSV {
				fmt.Fprintf(a.stdout, "%s\t%s\t%s\t%s\t%s\n", download.RunAccession, download.Type, raw, missingAny(download.Size), missingString(download.MD5))
			} else {
				fmt.Fprintln(a.stdout, raw)
			}
		}
	}
	if asJSON {
		if err := writeJSON(a.stdout, buffer); err != nil {
			return err
		}
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) runRuns(argv []string) error {
	var opts commonOptions
	fs := newFlagSet("runs", a.stderr)
	addCommonFlags(fs, &opts)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	var runs []string
	failed := 0
	for _, accession := range accessions {
		direct, err := a.resolver.DirectRunAccessionsFor(accession)
		if err != nil {
			failed++
			a.reportAccessionError("runs", accession, err)
			continue
		}
		if len(direct) > 0 {
			runs = append(runs, direct...)
			continue
		}
		tree, err := a.resolver.ResolveTree(accession, opts.fileType, true, nil)
		if err != nil {
			failed++
			a.reportAccessionError("runs", accession, err)
			continue
		}
		runs = append(runs, tree.RunAccessions()...)
	}
	for _, run := range uniqueStrings(runs) {
		fmt.Fprintln(a.stdout, run)
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) runTree(argv []string) error {
	var opts commonOptions
	fs := newFlagSet("tree", a.stderr)
	addCommonFlags(fs, &opts)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	limit, err := treeMaxDirectRuns()
	if err != nil {
		return err
	}
	threshold := 5
	if limit != nil && *limit < threshold {
		threshold = *limit
	}
	failed := 0
	for _, accession := range accessions {
		tree, err := a.resolver.ResolveTree(accession, opts.fileType, true, limit)
		if err != nil {
			failed++
			a.reportAccessionError("tree", accession, err)
			continue
		}
		fmt.Fprintln(a.stdout, NewTreeRenderer(opts.fileType, threshold).Render(tree))
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) runMeta(argv []string) error {
	var opts commonOptions
	var asJSON bool
	fs := newFlagSet("meta", a.stderr)
	addCommonFlags(fs, &opts)
	fs.BoolVar(&asJSON, "json", false, "print raw JSON")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	var records []Record
	failed := 0
	for _, accession := range accessions {
		record, err := a.resolver.FetchRecordFor(accession)
		if err != nil {
			failed++
			a.reportAccessionError("meta", accession, err)
			continue
		}
		if asJSON {
			records = append(records, record)
			continue
		}
		a.printMetaSummary(accession, record, opts.fileType)
	}
	if asJSON {
		if len(records) == 1 {
			if err := writeJSON(a.stdout, records[0]); err != nil {
				return err
			}
		} else if err := writeJSON(a.stdout, records); err != nil {
			return err
		}
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) printMetaSummary(accession string, record Record, fileType string) {
	for _, key := range infoFieldKeys {
		value := normalizedValue(recordValue(record, key))
		if value == "" {
			continue
		}
		label := key
		if key == IdentifierKey {
			label = "accession"
		}
		fmt.Fprintf(a.stdout, "%-18s %s\n", label+":", value)
	}
	switch stringValue(record, TypeKey) {
	case SRARunResourceType:
		fmt.Fprintf(a.stdout, "%-18s %d\n", "runs:", 1)
	case SRAExperimentResourceType:
		tree, err := a.resolver.ResolveTree(accession, fileType, true, nil)
		if err == nil {
			fmt.Fprintf(a.stdout, "%-18s %d\n", "runs:", len(uniqueStrings(tree.RunAccessions())))
		}
	}
}

func recordValue(record Record, key string) any {
	if key == IdentifierKey && stringValue(record, IdentifierKey) == "" {
		return record[AccessionKey]
	}
	return record[key]
}

func normalizedValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.Join(strings.Fields(v), " ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s := normalizedValue(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if name := normalizedValue(v["name"]); name != "" {
			return name
		}
		parts := make([]string, 0, len(v))
		for _, key := range sortedKeys(v) {
			if s := strings.TrimSpace(fmt.Sprint(v[key])); s != "" && s != "<nil>" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (a *app) runSize(argv []string) error {
	var opts commonOptions
	var protocol string
	var timeoutSeconds int
	var bytesOutput, asJSON, perRun, total bool
	perRun = true
	fs := newFlagSet("size", a.stderr)
	addCommonFlags(fs, &opts)
	fs.StringVar(&protocol, "protocol", "https", "preferred URL protocol: https, ftp")
	fs.IntVar(&timeoutSeconds, "timeout", 10, "HEAD/directory timeout in seconds")
	fs.BoolVar(&bytesOutput, "bytes", false, "print bytes")
	fs.BoolVar(&asJSON, "json", false, "print JSON")
	fs.BoolVar(&perRun, "per-run", true, "summarize per run")
	fs.BoolVar(&perRun, "r", true, "summarize per run")
	fs.BoolVar(&total, "total", false, "summarize per accession")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if total {
		perRun = false
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	if timeoutSeconds <= 0 {
		return newError("invalid_option", "invalid --timeout '"+strconv.Itoa(timeoutSeconds)+"' (expected: positive integer)")
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	var results []sizeResult
	headerPrinted := false
	failed := 0
	for _, accession := range accessions {
		record, err := a.fetchRecordWithDirectRunLimit(accession, "size")
		if err != nil {
			failed++
			a.reportAccessionError("size", accession, err)
			continue
		}
		downloads, err := a.resolver.ResolveDownloadsFromRecord(accession, record, opts.fileType)
		if err != nil {
			failed++
			a.reportAccessionError("size", accession, err)
			continue
		}
		if perRun {
			order, groups := groupDownloadsByRun(downloads)
			for _, run := range order {
				group := groups[run]
				result := a.sizeResultFor(run, group, protocol, time.Duration(timeoutSeconds)*time.Second)
				results = append(results, result)
				if !asJSON {
					printSizeResult(a.stdout, result, bytesOutput, &headerPrinted)
				}
			}
		} else {
			result := a.sizeResultFor(accession, downloads, protocol, time.Duration(timeoutSeconds)*time.Second)
			results = append(results, result)
			if !asJSON {
				printSizeResult(a.stdout, result, bytesOutput, &headerPrinted)
			}
		}
	}
	if asJSON {
		if err := writeJSON(a.stdout, results); err != nil {
			return err
		}
	} else if len(results) > 1 {
		totalResult := totalSizeResult(results)
		target := a.stdout
		if perRun {
			target = a.stderr
		}
		printSizeResult(target, totalResult, bytesOutput, &headerPrinted)
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

type sizeResult struct {
	Accession       string `json:"accession"`
	FileCount       int    `json:"file_count"`
	TotalSize       int64  `json:"total_size"`
	UnresolvedCount int    `json:"unresolved_count"`
}

func (a *app) sizeResultFor(label string, downloads []DownloadCandidate, protocol string, timeout time.Duration) sizeResult {
	result := sizeResult{Accession: label}
	for _, download := range downloads {
		for _, length := range a.downloader.ContentLengths(download, protocol, timeout) {
			result.FileCount++
			if length < 0 {
				result.UnresolvedCount++
			} else {
				result.TotalSize += length
			}
		}
	}
	return result
}

func groupDownloadsByRun(downloads []DownloadCandidate) ([]string, map[string][]DownloadCandidate) {
	out := map[string][]DownloadCandidate{}
	var order []string
	for _, download := range downloads {
		if _, ok := out[download.RunAccession]; !ok {
			order = append(order, download.RunAccession)
		}
		out[download.RunAccession] = append(out[download.RunAccession], download)
	}
	return order, out
}

func printSizeResult(w io.Writer, result sizeResult, bytesOutput bool, headerPrinted *bool) {
	if !*headerPrinted {
		fmt.Fprintln(w, "#accession\tfiles\tsize\tunresolved")
		*headerPrinted = true
	}
	size := "NA"
	if result.TotalSize != 0 || result.UnresolvedCount == 0 {
		if bytesOutput {
			size = strconv.FormatInt(result.TotalSize, 10)
		} else {
			size = formatBytes(result.TotalSize)
		}
	}
	fmt.Fprintf(w, "%s\t%d\t%s\t%d\n", result.Accession, result.FileCount, size, result.UnresolvedCount)
}

func totalSizeResult(results []sizeResult) sizeResult {
	out := sizeResult{Accession: "total"}
	for _, result := range results {
		out.FileCount += result.FileCount
		out.TotalSize += result.TotalSize
		out.UnresolvedCount += result.UnresolvedCount
	}
	return out
}

func (a *app) runProbe(argv []string) error {
	var opts commonOptions
	var protocol string
	var timeoutSeconds int
	fs := newFlagSet("probe", a.stderr)
	addCommonFlags(fs, &opts)
	fs.StringVar(&protocol, "protocol", "https", "preferred URL protocol: https, ftp")
	fs.IntVar(&timeoutSeconds, "timeout", 5, "probe timeout in seconds")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	if timeoutSeconds <= 0 {
		return newError("invalid_option", "invalid --timeout '"+strconv.Itoa(timeoutSeconds)+"' (expected: positive integer)")
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	failed := 0
accessionLoop:
	for _, accession := range accessions {
		downloads, err := a.resolver.ResolveDownloads(accession, opts.fileType)
		if err != nil {
			failed++
			a.reportAccessionError("probe", accession, err)
			continue
		}
		for _, download := range downloads {
			if err := a.downloader.ProbeDownload(download, protocol, time.Duration(timeoutSeconds)*time.Second); err != nil {
				failed++
				a.reportAccessionError("probe", accession, err)
				continue accessionLoop
			}
			raw, _ := download.URLForProtocol(protocol)
			fmt.Fprintf(a.stdout, "OK\t%s\n", raw)
		}
	}
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) runGet(argv []string) error {
	var opts commonOptions
	var protocol, outdir string
	var noVerify, force, skipExisting bool
	fs := newFlagSet("get", a.stderr)
	addCommonFlags(fs, &opts)
	fs.StringVar(&outdir, "outdir", ".", "output directory")
	fs.StringVar(&outdir, "O", ".", "output directory")
	fs.StringVar(&protocol, "protocol", "https", "preferred URL protocol: https, ftp")
	fs.BoolVar(&noVerify, "no-verify", false, "skip md5 verification")
	fs.BoolVar(&force, "force", false, "re-download existing files")
	fs.BoolVar(&skipExisting, "skip-existing", false, "skip existing files")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := validateCommon(opts); err != nil {
		return err
	}
	if err := validateProtocol(protocol); err != nil {
		return err
	}
	accessions, err := a.accessions(fs, opts)
	if err != nil {
		return err
	}
	downloaded, skipped, failed := 0, 0, 0
	for _, accession := range accessions {
		downloads, err := a.resolver.ResolveDownloads(accession, opts.fileType)
		if err != nil {
			failed++
			a.reportAccessionError("get", accession, err)
			continue
		}
		for _, download := range downloads {
			result, err := a.downloader.SaveDownload(download, outdir, protocol, !noVerify, force, skipExisting)
			if err != nil {
				failed++
				a.reportAccessionError("get", accession, err)
				continue
			}
			if result.Skipped {
				skipped++
				fmt.Fprintf(a.stderr, "%s skipped\t%s\n", MessagePrefix, result.Path)
			} else {
				downloaded++
				fmt.Fprintf(a.stderr, "%s downloaded\t%s\n", MessagePrefix, result.Path)
			}
		}
	}
	fmt.Fprintf(a.stderr, "%s get: %d downloaded, %d skipped", MessagePrefix, downloaded, skipped)
	if failed > 0 {
		fmt.Fprintf(a.stderr, ", %d failed", failed)
	}
	fmt.Fprintln(a.stderr)
	if failed > 0 {
		return reportedErrors{}
	}
	return nil
}

func (a *app) fetchRecordWithDirectRunLimit(accession, command string) (Record, error) {
	var limit *int
	var err error
	var envName string
	switch command {
	case "url":
		limit, err = urlMaxDirectRuns()
		envName = urlMaxDirectRunsEnv
	case "size":
		limit, err = sizeMaxDirectRuns()
		envName = sizeMaxDirectRunsEnv
	}
	if err != nil {
		return nil, err
	}
	if limit != nil {
		count, err := a.resolver.DirectRunCountFor(accession)
		if err != nil {
			return nil, err
		}
		if count > *limit {
			return nil, newError("invalid_record", fmt.Sprintf("%s has %d direct runs; %s expands at most %d direct runs from one parent accession. Use `%s runs %s` and pass narrower accessions, or set %s=unlimited.", strings.ToUpper(accession), count, command, *limit, Name, accession, envName))
		}
	}
	return a.resolver.FetchRecordFor(strings.ToUpper(accession))
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func missingString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "NA"
	}
	return value
}

func missingAny(value any) string {
	if value == nil {
		return "NA"
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return "NA"
	}
	return text
}
