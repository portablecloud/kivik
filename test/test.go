package test

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver/couchdb/chttp"
	"github.com/flimzy/kivik/errors"
	"github.com/flimzy/kivik/test/kt"

	// Tests
	_ "github.com/flimzy/kivik/test/client"
	_ "github.com/flimzy/kivik/test/db"
)

// The available test suites
const (
	SuiteAuto        = "auto"
	SuitePouchLocal  = "pouch"
	SuitePouchRemote = "pouchRemote"
	SuiteCouch16     = "couch16"
	SuiteCouch20     = "couch20"
	SuiteCloudant    = "cloudant"
	SuiteKivikServer = "kivikServer"
	SuiteKivikMemory = "kivikMemory"
	SuiteKivikFS     = "kivikFilesystem"
)

// AllSuites is a list of all defined suites.
var AllSuites = []string{
	SuitePouchLocal,
	SuitePouchRemote,
	SuiteCouch16,
	SuiteCouch20,
	SuiteKivikMemory,
	SuiteKivikFS,
	SuiteCloudant,
	SuiteKivikServer,
}

var driverMap = map[string]string{
	SuitePouchLocal:  "pouch",
	SuitePouchRemote: "pouch",
	SuiteCouch16:     "couch",
	SuiteCouch20:     "couch",
	SuiteCloudant:    "couch",
	SuiteKivikServer: "couch",
	SuiteKivikMemory: "memory",
	SuiteKivikFS:     "fs",
}

// ListTests prints a list of available test suites to stdout.
func ListTests() {
	fmt.Printf("Available test suites:\n\tauto\n")
	for _, suite := range AllSuites {
		fmt.Printf("\t%s\n", suite)
	}
}

// Options are the options to run a test from the command line tool.
type Options struct {
	Driver  string
	DSN     string
	Verbose bool
	RW      bool
	Match   string
	Suites  []string
	Cleanup bool
}

// CleanupTests attempts to clean up any stray test databases created by a
// previous test run.
func CleanupTests(driver, dsn string, verbose bool) error {
	client, err := kivik.New(driver, dsn)
	if err != nil {
		return err
	}
	count, err := doCleanup(client, verbose)
	if verbose {
		fmt.Printf("Deleted %d test databases\n", count)
	}
	return err
}

func doCleanup(client *kivik.Client, verbose bool) (int, error) {
	allDBs, err := client.AllDBs()
	if err != nil {
		return 0, err
	}
	var count int
	for _, dbName := range allDBs {
		// FIXME: This filtering should be possible in AllDBs(), but all the
		// backends need to support it first.
		if strings.HasPrefix(dbName, kt.TestDBPrefix) {
			if verbose {
				fmt.Printf("\t--- Deleting %s\n", dbName)
			}
			err := client.DestroyDB(dbName)
			if err != nil && errors.StatusCode(err) != http.StatusNotFound {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

// RunTests runs the requested test suites against the requested driver and DSN.
func RunTests(opts Options) {
	if opts.Cleanup {
		err := CleanupTests(opts.Driver, opts.DSN, opts.Verbose)
		if err != nil {
			fmt.Printf("Cleanup failed: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	flag.Set("test.run", opts.Match)
	if opts.Verbose {
		flag.Set("test.v", "true")
	}
	tests := []testing.InternalTest{
		testing.InternalTest{
			Name: "MainTest",
			F: func(t *testing.T) {
				Test(opts.Driver, opts.DSN, opts.Suites, opts.RW, t)
			},
		},
	}

	mainStart(tests)
}

// Test is the main test entry point when running tests through the command line
// tool.
func Test(driver, dsn string, testSuites []string, rw bool, t *testing.T) {
	clients, err := connectClients(driver, dsn, t)
	if err != nil {
		t.Fatalf("Failed to connect to %s (%s driver): %s\n", dsn, driver, err)
	}
	clients.RW = rw
	tests := make(map[string]struct{})
	for _, test := range testSuites {
		tests[test] = struct{}{}
	}
	if _, ok := tests[SuiteAuto]; ok {
		t.Log("Detecting target service compatibility...")
		suites, err := detectCompatibility(clients.Admin)
		if err != nil {
			t.Fatalf("Unable to determine server suite compatibility: %s\n", err)
		}
		tests = make(map[string]struct{})
		for _, suite := range suites {
			tests[suite] = struct{}{}
		}
	}
	testSuites = make([]string, 0, len(tests))
	for test := range tests {
		testSuites = append(testSuites, test)
	}
	t.Logf("Running the following test suites: %s\n", strings.Join(testSuites, ", "))
	for _, suite := range testSuites {
		runTests(clients, suite, t)
	}
}

func runTests(ctx *kt.Context, suite string, t *testing.T) {
	ctx.T = t
	conf, ok := suites[suite]
	if !ok {
		ctx.Skipf("No configuration found for suite '%s'", suite)
	}
	ctx.Config = conf
	// This is run as a sub-test so configuration will work nicely.
	ctx.Run("PreCleanup", func(ctx *kt.Context) {
		ctx.RunAdmin(func(ctx *kt.Context) {
			count, err := doCleanup(ctx.Admin, true)
			if count > 0 {
				ctx.Logf("Pre-cleanup removed %d databases from previous test runs", count)
			}
			if err != nil {
				ctx.Fatalf("Pre-cleanup failed: %s", err)
			}
		})
	})
	kt.RunSubtests(ctx)
}

func detectCompatibility(client *kivik.Client) ([]string, error) {
	info, err := client.ServerInfo()
	if err != nil {
		return nil, err
	}
	switch info.Vendor() {
	case "PouchDB":
		return []string{SuitePouchLocal}, nil
	case "IBM Cloudant":
		return []string{SuiteCloudant}, nil
	case "The Apache Software Foundation":
		if info.Version() == "2.0" {
			return []string{SuiteCouch20}, nil
		}
		return []string{SuiteCouch16}, nil
	case "Kivik Memory Adaptor":
		return []string{SuiteKivikMemory}, nil
	}
	return []string{}, errors.New("Unable to automatically determine the proper test suite")
}

func connectClients(driverName, dsn string, t *testing.T) (*kt.Context, error) {
	var noAuthDSN string
	if parsed, err := url.Parse(dsn); err == nil {
		if parsed.User == nil {
			return nil, errors.New("DSN does not contain authentication credentials")
		}
		parsed.User = nil
		noAuthDSN = parsed.String()
	}
	clients := &kt.Context{}
	t.Logf("Connecting to %s ...\n", dsn)
	if client, err := kivik.New(driverName, dsn); err == nil {
		_ = client.SetDefault("force_commit", true)
		clients.Admin = client
	} else {
		return nil, err
	}
	if chttpClient, err := chttp.New(dsn); err == nil {
		clients.CHTTPAdmin = chttpClient
	} else {
		return nil, err
	}

	t.Logf("Connecting to %s ...\n", noAuthDSN)
	if client, err := kivik.New(driverName, noAuthDSN); err == nil {
		_ = client.SetDefault("force_commit", true)
		clients.NoAuth = client
	} else {
		return nil, err
	}
	if chttpClient, err := chttp.New(noAuthDSN); err == nil {
		clients.CHTTPNoAuth = chttpClient
	} else {
		return nil, err
	}
	return clients, nil
}

func doTest(suite, envName string, t *testing.T) {
	dsn := os.Getenv(envName)
	if dsn == "" {
		t.Skipf("%s: %s DSN not set; skipping tests", envName, suite)
	}
	clients, err := connectClients(driverMap[suite], dsn, t)
	if err != nil {
		t.Errorf("Failed to connect to %s: %s\n", suite, err)
		return
	}
	clients.RW = true
	runTests(clients, suite, t)
}
