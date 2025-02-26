package cmd

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/askgitdev/askgit/pkg/display"
	. "github.com/askgitdev/askgit/pkg/query"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var format string                                     // output format flag
var presetQuery string                                // named / preset query flag
var repo string                                       // path to repo on disk
var githubToken = os.Getenv("GITHUB_TOKEN")           // GitHub auth token for GitHub tables
var sourcegraphToken = os.Getenv("SOURCEGRAPH_TOKEN") // Sourcegraph auth token for Sourcegraph queries
var verbose bool                                      // whether or not to print logs to stderr
var logger = zerolog.Nop()                            // By default use a NOOP logger

func init() {
	// local (root command only) flags
	rootCmd.Flags().StringVarP(&format, "format", "f", "table", "specify the output format. Options are 'csv' 'tsv' 'table' 'single' and 'json'")
	rootCmd.Flags().StringVarP(&presetQuery, "preset", "p", "", "used to pick a preset query")
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", ".", "specify a path to a default repo on disk. This will be used if no repo is supplied as an argument to a git table")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "whether or not to print query execution logs to stderr")

	// register the sqlite extension ahead of any command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		setupLogger()
		registerExt()
	}

	// add the export sub command
	rootCmd.AddCommand(exportCmd)

	// conditionally add the pgsync sub command
	// TODO(patrickdevivo) "conditional" for now until the behavior stabilizes
	if os.Getenv("PGSYNC") != "" {
		rootCmd.AddCommand(pgsyncCmd)
	}
}

// setupLogger sets the global logger variable according to whether the verbose flag is used
// or if the DEBUG environment variable is set to anything
func setupLogger() {
	l := zerolog.New(os.Stderr).
		Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Stamp}).
		Level(zerolog.ErrorLevel).
		With().
		Timestamp().Logger()
	if debug := os.Getenv("DEBUG") != ""; verbose || debug {
		l = l.Level(zerolog.InfoLevel)
		if debug {
			l = l.Level(zerolog.DebugLevel)
		}
	}
	logger = l
}

// handleExitError should be used for any errors that should stop execution of the CLI (exit)
// it will report an error with the logger and exit with code 1
func handleExitError(err error) {
	if err != nil {
		logger.Error().Msgf(err.Error())
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:  `display "SELECT * FROM commits"`,
	Args: cobra.MaximumNArgs(2),
	Long: `
  askgit is a CLI for querying git repositories with SQL, using SQLite virtual tables.
  Example queries can be found in the GitHub repo: https://github.com/askgitdev/askgit`,
	Short: `query your github repos with SQL`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		var info os.FileInfo
		if info, err = os.Stdin.Stat(); err != nil {
			handleExitError(fmt.Errorf("failed to read stdin stat: %v", err))
		}

		var query string
		if len(args) > 0 {
			query = args[0]
		} else if isPiped(info) {
			var stdin []byte
			if stdin, err = ioutil.ReadAll(os.Stdin); err != nil {
				handleExitError(fmt.Errorf("failed to read from stdin: %v", err))
			}
			query = string(stdin)
		} else if presetQuery != "" {
			var found bool
			if query, found = Find(presetQuery); !found {
				handleExitError(fmt.Errorf("unknown preset query: %s", presetQuery))
			}
		} else {
			if err = cmd.Help(); err != nil {
				handleExitError(err)
			}
			os.Exit(0)
		}

		var db *sql.DB
		if db, err = sql.Open("sqlite3", ":memory:"); err != nil {
			handleExitError(fmt.Errorf("failed to initialize database connection: %v", err))
		}

		var rows *sql.Rows
		if rows, err = db.Query(query); err != nil {
			handleExitError(fmt.Errorf("query execution failed: %v", err))
		}
		defer rows.Close()

		if err = display.WriteTo(rows, os.Stdout, format, false); err != nil {
			handleExitError(fmt.Errorf("failed to output resultset: %v", err))
		}
	},
}

func isPiped(info os.FileInfo) bool { return info.Mode()&os.ModeCharDevice == 0 }

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		handleExitError(fmt.Errorf("execution failed: %v", err))
	}
}
