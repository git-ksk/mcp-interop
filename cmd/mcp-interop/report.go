package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/reporting"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

type reportSuiteOptions struct {
	baseline         string
	attempts         []string
	htmlPath         string
	ciPath           string
	failOnRegression bool
}

func runReport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "suite" {
		fmt.Fprintln(stderr, "report requires: suite <baseline> <attempt> [<attempt>...] --html <file> [--ci-summary <file>] [--fail-on-regression]")
		return 2
	}
	options, err := parseReportSuiteOptions(args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite report: %v\n", err)
		return 2
	}
	paths := []string{options.baseline}
	paths = append(paths, options.attempts...)
	loaded := make([]suite.LoadedResultSet, 0, len(paths))
	for i, path := range paths {
		index, err := resolveSuiteIndexPath(path)
		if err != nil {
			fmt.Fprintf(stderr, "resolve report result set %d: %v\n", i, err)
			return 2
		}
		set, err := suite.ReadResultSet(index)
		if err != nil {
			fmt.Fprintf(stderr, "read report result set %d: %v\n", i, err)
			return 2
		}
		loaded = append(loaded, set)
	}
	comparison, err := suite.CompareResultSets(loaded[0], loaded[1:])
	if err != nil {
		fmt.Fprintf(stderr, "compare suite report: %v\n", err)
		return 2
	}
	var htmlData, ciData []byte
	if options.htmlPath != "" {
		htmlData, err = reporting.SuiteHTML(comparison)
		if err != nil {
			fmt.Fprintf(stderr, "render offline HTML: %v\n", err)
			return 2
		}
	}
	if options.ciPath != "" {
		ciData, err = reporting.SuiteCI(comparison)
		if err != nil {
			fmt.Fprintf(stderr, "render CI summary: %v\n", err)
			return 2
		}
	}
	// Both views must be fully built before creating either output.
	if options.htmlPath != "" {
		if err := writeReportFile(options.htmlPath, htmlData); err != nil {
			fmt.Fprintf(stderr, "write offline HTML: %v\n", err)
			return 1
		}
	}
	if options.ciPath != "" {
		if err := writeReportFile(options.ciPath, ciData); err != nil {
			fmt.Fprintf(stderr, "write CI summary: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "DECISION\t%s\nREGRESSION\t%s\nUNSTABLE\t%s\nATTEMPTS\t%d\n", strings.ToUpper(string(comparison.Decision)), yesNo(comparison.HasRegression), yesNo(comparison.HasUnstable), comparison.AttemptCount)
	if options.failOnRegression && (comparison.HasRegression || comparison.HasUnstable) {
		return 1
	}
	return 0
}

func parseReportSuiteOptions(args []string) (reportSuiteOptions, error) {
	var o reportSuiteOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--fail-on-regression":
			o.failOnRegression = true
		case arg == "--html" || strings.HasPrefix(arg, "--html="):
			if o.htmlPath != "" {
				return o, errors.New("duplicate --html")
			}
			if arg == "--html" {
				if i+1 >= len(args) {
					return o, errors.New("--html requires file path")
				}
				i++
				o.htmlPath = args[i]
			} else {
				o.htmlPath = strings.TrimPrefix(arg, "--html=")
			}
			if o.htmlPath == "" {
				return o, errors.New("--html requires file path")
			}
		case arg == "--ci-summary" || strings.HasPrefix(arg, "--ci-summary="):
			if o.ciPath != "" {
				return o, errors.New("duplicate --ci-summary")
			}
			if arg == "--ci-summary" {
				if i+1 >= len(args) {
					return o, errors.New("--ci-summary requires file path")
				}
				i++
				o.ciPath = args[i]
			} else {
				o.ciPath = strings.TrimPrefix(arg, "--ci-summary=")
			}
			if o.ciPath == "" {
				return o, errors.New("--ci-summary requires file path")
			}
		case strings.HasPrefix(arg, "-"):
			return o, errors.New("unknown suite report flag")
		default:
			if o.baseline == "" {
				o.baseline = arg
			} else {
				o.attempts = append(o.attempts, arg)
			}
		}
	}
	if o.baseline == "" || len(o.attempts) == 0 || len(o.attempts) > reporting.MaxAttempts {
		return o, fmt.Errorf("suite report requires baseline and 1..%d current attempts", reporting.MaxAttempts)
	}
	if o.htmlPath == "" && o.ciPath == "" {
		return o, errors.New("at least one of --html or --ci-summary is required")
	}
	canonical := make(map[string]struct{}, 2)
	for _, path := range []string{o.htmlPath, o.ciPath} {
		if path == "" {
			continue
		}
		if path == "-" || strings.TrimSpace(path) != path {
			return o, errors.New("report output requires a non-empty file path")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return o, fmt.Errorf("resolve report output: %w", err)
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return o, fmt.Errorf("resolve report output parent: %w", err)
		}
		normalized := filepath.Join(parent, filepath.Base(abs))
		if _, exists := canonical[normalized]; exists {
			return o, errors.New("HTML and CI outputs must use distinct files")
		}
		canonical[normalized] = struct{}{}
		if _, err := os.Lstat(path); err == nil {
			return o, errors.New("report output already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return o, fmt.Errorf("inspect report output: %w", err)
		}
	}
	return o, nil
}

func writeReportFile(path string, data []byte) error {
	if len(data) == 0 || len(data) > reporting.MaxOutputBytes {
		return errors.New("invalid report payload size")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}
