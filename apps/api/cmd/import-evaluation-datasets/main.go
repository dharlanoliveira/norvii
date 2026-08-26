// Command import-evaluation-datasets validates and imports repository-owned evaluation assets.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	evaluationpostgres "github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/postgres"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/persistence"
)

const commandHelp = `Usage: import-evaluation-datasets

Validate and import the three repository-owned evaluation datasets as immutable drafts.
The command reads only data/corpora/*/evaluation assets and does not fetch sources,
run models, bind sources, publish datasets, or change active snapshots.`

func main() {
	os.Exit(run(os.Args[1:], os.Getwd, os.LookupEnv, os.Stdout, os.Stderr))
}

func run(
	arguments []string,
	getwd func() (string, error),
	lookup persistence.EnvironmentLookup,
	stdout, stderr io.Writer,
) int {
	if len(arguments) == 1 && (arguments[0] == "-h" || arguments[0] == "--help") {
		fmt.Fprintln(stdout, commandHelp)
		return 0
	}
	if len(arguments) != 0 {
		fmt.Fprintln(stderr, commandHelp)
		return 2
	}

	configuration, err := persistence.LoadConfig(lookup)
	if err != nil {
		fmt.Fprintf(stderr, "Evaluation dataset import configuration failed: %v\n", err)
		return 1
	}
	workingDirectory, err := getwd()
	if err != nil {
		fmt.Fprintln(stderr, "Evaluation dataset import could not determine the working directory.")
		return 1
	}
	repositoryRoot, err := findRepositoryRoot(workingDirectory)
	if err != nil {
		fmt.Fprintln(stderr, "Evaluation dataset import must run inside the Norvii repository.")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), configuration.Timeout)
	defer cancel()
	pool, err := persistence.OpenPostgresPool(ctx, configuration.Postgres)
	if err != nil {
		fmt.Fprintf(stderr, "Evaluation dataset import storage failed: %v\n", err)
		return 1
	}
	defer pool.Close()

	reader := projectAssetReader{repositoryRoot: repositoryRoot}
	service := application.NewDatasetImportService(
		application.NewImporter(reader),
		evaluationpostgres.NewDatasetImporter(pool),
	)
	for _, request := range application.OwnedAssetRequests() {
		result, importErr := service.Import(ctx, request)
		if importErr != nil {
			fmt.Fprintf(stderr, "Evaluation dataset import failed for %s: %v\n", request.CorpusKey, importErr)
			return 1
		}
		outcome := "already present"
		if result.Created {
			outcome = "imported"
		}
		fmt.Fprintf(stdout, "Dataset %s revision %s %s (sources: %d, cases: %d, evidence: %d, starters: %d).\n",
			result.DatasetKey, result.RevisionID, outcome, result.Sources, result.Cases, result.Evidence, result.Starters,
		)
	}
	return 0
}

type projectAssetReader struct {
	repositoryRoot string
}

func (reader projectAssetReader) OpenAsset(projectRelativePath string) (io.ReadCloser, error) {
	if reader.repositoryRoot == "" || filepath.IsAbs(projectRelativePath) {
		return nil, os.ErrPermission
	}
	candidate := filepath.Join(reader.repositoryRoot, filepath.FromSlash(projectRelativePath))
	relative, err := filepath.Rel(reader.repositoryRoot, candidate)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, os.ErrPermission
	}
	return os.Open(candidate)
}

func findRepositoryRoot(workingDirectory string) (string, error) {
	directory, err := filepath.Abs(workingDirectory)
	if err != nil {
		return "", err
	}
	for {
		if isRepositoryRoot(directory) {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("repository root not found")
		}
		directory = parent
	}
}

func isRepositoryRoot(directory string) bool {
	corpora, corporaErr := os.Stat(filepath.Join(directory, "data", "corpora"))
	apiModule, apiErr := os.Stat(filepath.Join(directory, "apps", "api", "go.mod"))
	return corporaErr == nil && corpora.IsDir() && apiErr == nil && !apiModule.IsDir()
}
