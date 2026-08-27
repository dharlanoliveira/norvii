package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dharlanoliveira/norvii/apps/api/internal/evaluation/application"
	"github.com/dharlanoliveira/norvii/apps/api/internal/platform/config"
)

func TestClientContractWithPythonEvaluationTransport(t *testing.T) {
	t.Setenv("NORVII_EVALUATION_RETRIEVAL_STRATEGY", "hybrid")
	root := repositoryRoot(t)
	fixture := filepath.Join(root, "apps", "agent", "tests", "contract", "evaluation_transport_fixture.py")
	agentProject := filepath.Join(root, "apps", "agent")
	command := exec.Command("uv", "run", "--project", agentProject, "python", fixture)
	command.Env = append(os.Environ(), fixtureEnvironment("agent-build-test", "embedding-model-test")...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("open Python transport stdout: %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start Python evaluation transport: %v", err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	})

	portLine := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			portLine <- scanner.Text()
		}
	}()
	select {
	case value := <-portLine:
		port, parseErr := strconv.Atoi(value)
		if parseErr != nil || port < 1 {
			t.Fatalf("Python evaluation transport port = %q, parse error = %v", value, parseErr)
		}
		request := evaluationRequest()
		result, executeErr := NewClient(config.AgentConfig{
			BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port), Timeout: 2 * time.Second,
		}).Execute(context.Background(), request)
		if executeErr != nil {
			t.Fatalf("Execute() against Python transport error = %v", executeErr)
		}
		if result.Materialized.Answer != "The fixed snapshot applies [1]." ||
			len(result.Materialized.Retrieved) != 1 || len(result.Materialized.Cited) != 1 ||
			result.Materialized.Retrieved[0].Provenance.UnitID.String() != "60000000-0000-4000-8000-000000000001" ||
			string(result.Materialized.Retrieved[0].Provenance.ContentSHA256) != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" ||
			len(result.CitationMarkers) != 1 || result.CitationMarkers[0] != (CitationMarkerInput{MarkerPosition: 1, EvidenceRank: 1}) {
			t.Fatalf("Python evaluation transport result = %#v, want ordered complete fixed-snapshot provenance", result)
		}
		if result.AgentBuildIdentity != request.ExecutionIdentity.AgentBuild ||
			result.EmbeddingModelIdentity != request.ExecutionIdentity.EmbeddingModelIdentity {
			t.Fatalf("Python evaluation transport identity = %#v, want %#v", result, request.ExecutionIdentity)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Python evaluation transport did not report a listening port")
	}
}

func TestClientContractNormalizesDeployedExecutionIdentity(t *testing.T) {
	root := repositoryRoot(t)
	fixture := filepath.Join(root, "apps", "agent", "tests", "contract", "evaluation_transport_fixture.py")
	agentProject := filepath.Join(root, "apps", "agent")
	configuration, err := config.Load(func(key string) (string, bool) {
		switch key {
		case "NORVII_EVALUATION_AGENT_BUILD":
			return "  agent-build-test  ", true
		case "NORVII_EMBEDDING_MODEL":
			return "  embedding-model-test  ", true
		case "NORVII_CHAT_MODEL":
			return "test-model", true
		default:
			return "", false
		}
	})
	if err != nil {
		t.Fatalf("load normalized deployed configuration: %v", err)
	}
	command := exec.Command("uv", "run", "--project", agentProject, "python", fixture)
	command.Env = append(os.Environ(), fixtureEnvironment("  agent-build-test  ", "  embedding-model-test  ")...)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("open Python transport stdout: %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start Python evaluation transport: %v", err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	})

	request := evaluationRequest()
	request.ExecutionIdentity = application.ExecutionIdentity{
		AgentBuild: configuration.Evaluation.AgentBuild, ChatModelIdentity: configuration.Evaluation.ChatModelIdentity,
		EmbeddingModelIdentity: configuration.Evaluation.EmbeddingModelIdentity,
	}
	result, err := NewClient(config.AgentConfig{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", fixturePort(t, stdout)), Timeout: 2 * time.Second,
	}).Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute against normalized Python transport: %v", err)
	}
	if result.AgentBuildIdentity != "agent-build-test" || result.EmbeddingModelIdentity != "embedding-model-test" {
		t.Fatalf("normalized execution identity = %#v", result)
	}
}

func fixtureEnvironment(agentBuild, embeddingModel string) []string {
	return []string{
		"NORVII_EVALUATION_AGENT_BUILD=" + agentBuild,
		"NORVII_EMBEDDING_MODEL=" + embeddingModel,
		"NORVII_CHAT_MODEL=test-model",
		"NORVII_EVALUATION_RETRIEVAL_STRATEGY=vector",
		"NORVII_EVALUATION_RETRIEVAL_FINGERPRINT=" + strings.Repeat("f", 64),
	}
}

func fixturePort(t *testing.T, stdout io.Reader) int {
	t.Helper()
	portLine := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			portLine <- scanner.Text()
		}
	}()
	select {
	case value := <-portLine:
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 {
			t.Fatalf("Python evaluation transport port = %q, parse error = %v", value, err)
		}
		return port
	case <-time.After(10 * time.Second):
		t.Fatal("Python evaluation transport did not report a listening port")
		return 0
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}
