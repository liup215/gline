package adkagent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

	"github.com/liup215/gline/internal/mcp"
	"github.com/liup215/gline/internal/tools"
	"github.com/liup215/gline/pkg/types"
)

// fakeSkillRegistry satisfies tools.SkillRegistry for tests.
type fakeSkillRegistry struct{}

func (fakeSkillRegistry) Get(name string) (*types.Skill, bool) {
	if name != "demo-skill" {
		return nil, false
	}
	return &types.Skill{
		Name:        "demo-skill",
		Description: "A demo skill",
		Contents:    "Do the demo thing.",
	}, true
}

func (fakeSkillRegistry) GetMeta() []types.SkillMeta {
	return []types.SkillMeta{{Name: "demo-skill", Description: "A demo skill"}}
}

// TestAllRegistryToolsAdaptToADK walks the full default registry and asserts
// every non-legacy tool survives the adkTool adaptation: a valid declaration
// and a ProcessRequest that packs it into the LLM request.
func TestAllRegistryToolsAdaptToADK(t *testing.T) {
	registry := tools.InitDefaultRegistry(nil, nil)
	tools.RegisterSkillTool(registry, fakeSkillRegistry{})

	a := &Agent{opts: Options{Tools: registry}}
	adkTools, err := a.buildTools()
	if err != nil {
		t.Fatalf("buildTools: %v", err)
	}
	if len(adkTools) == 0 {
		t.Fatal("no tools")
	}

	seen := map[string]bool{}
	for _, at := range adkTools {
		gt, ok := at.(*adkTool)
		if !ok {
			t.Fatalf("tool %T is not *adkTool", at)
		}
		name := gt.Name()
		if name == "" {
			t.Fatal("empty tool name")
		}
		if seen[name] {
			t.Fatalf("duplicate tool %s", name)
		}
		seen[name] = true
		if gt.Description() == "" {
			t.Errorf("tool %s: empty description", name)
		}
		if legacyToolNames[name] {
			t.Errorf("tool %s should have been excluded", name)
		}

		decl := gt.Declaration()
		if decl.Name != name {
			t.Errorf("tool %s: declaration name %q", name, decl.Name)
		}
		if decl.ParametersJsonSchema != nil {
			if _, err := json.Marshal(decl.ParametersJsonSchema); err != nil {
				t.Errorf("tool %s: schema not serializable: %v", name, err)
			}
		}

		// ProcessRequest must pack the declaration into the request.
		req := &model.LLMRequest{}
		if err := gt.ProcessRequest(nil, req); err != nil {
			t.Errorf("tool %s: ProcessRequest: %v", name, err)
		}
	}
	if !seen["read"] || !seen["write"] || !seen["run"] || !seen["use_skill"] {
		t.Fatalf("core tools missing from adaptation: %v", seen)
	}
	if seen[string(types.ToolAttemptCompletion)] {
		t.Fatal("attempt_completion must not be adapted")
	}
}

// TestProcessRequestPacksDeclaration verifies the packed request actually
// carries the tool's schema (what the provider serializes).
func TestProcessRequestPacksDeclaration(t *testing.T) {
	registry := tools.InitDefaultRegistry(nil, nil)
	a := &Agent{opts: Options{Tools: registry}}
	adkTools, err := a.buildTools()
	if err != nil {
		t.Fatal(err)
	}
	var readTool *adkTool
	for _, at := range adkTools {
		if at.Name() == "read" {
			readTool = at.(*adkTool)
			break
		}
	}
	if readTool == nil {
		t.Fatal("read tool not found")
	}

	req := &model.LLMRequest{}
	if err := readTool.ProcessRequest(nil, req); err != nil {
		t.Fatalf("ProcessRequest: %v", err)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("req.Tools has %d entries, want 1", len(req.Tools))
	}
	entry, ok := req.Tools["read"]
	if !ok {
		t.Fatalf("req.Tools missing \"read\"; keys=%v", req.Tools)
	}
	// toolutils.PackTool stores the tool itself; ADK's flow converts it to a
	// declaration via the Declaration() interface at request build time.
	type declarer interface{ Declaration() *genai.FunctionDeclaration }
	toolWithDecl, ok := entry.(declarer)
	if !ok {
		t.Fatalf("req.Tools[\"read\"] is %T, does not implement Declaration()", entry)
	}
	decl := toolWithDecl.Declaration()
	if decl == nil {
		t.Fatal("Declaration() returned nil")
	}
	if decl.ParametersJsonSchema == nil {
		t.Fatal("packed declaration lost its schema")
	}
	b, _ := json.Marshal(decl.ParametersJsonSchema)
	if !strings.Contains(string(b), "path") {
		t.Fatalf("schema missing path property: %s", b)
	}
}

// TestUseSkillThroughAdapter runs the real use_skill tool through the
// adkTool wrapper: skill contents come back as the {"result": ...} payload.
func TestUseSkillThroughAdapter(t *testing.T) {
	registry := tools.InitDefaultRegistry(nil, nil)
	tools.RegisterSkillTool(registry, fakeSkillRegistry{})
	tool, err := registry.Get("use_skill")
	if err != nil {
		t.Fatalf("use_skill not registered: %v", err)
	}

	a := &Agent{opts: Options{Tools: registry, Yolo: true}}
	at := newADKTool(tool, nil, a)
	out, err := at.Run(nil, map[string]any{"skill_name": "demo-skill"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	result, _ := out["result"].(string)
	if !strings.Contains(result, "Do the demo thing") {
		t.Fatalf("skill contents missing from result: %q", result)
	}
}

// TestMCPToolThroughADKLoop is the MCP integration smoke: a real stdio MCP
// server subprocess registers its tools into the gline registry, and the
// ADK loop executes one of them end to end via a scripted model turn.
func TestMCPToolThroughADKLoop(t *testing.T) {
	if runtime.GOOS == "windows" && isCI() {
		t.Skip("go run subprocess flaky on CI Windows runners")
	}

	serverPath, err := filepath.Abs(filepath.Join("..", "mcp", "testdata", "echo_server", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	// Verify the server compiles before wiring it up (fails fast with a
	// clear message instead of a transport timeout).
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", os.DevNull, serverPath)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("echo server does not compile: %v\n%s", err, out)
	}

	registry := tools.NewRegistry()
	mgr := mcp.NewManager(&mcp.Config{
		Servers: []mcp.ServerConfig{{
			Name:    "echo",
			Command: "go",
			Args:    []string{"run", serverPath},
		}},
	}, registry)
	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("MCP start: %v", err)
	}
	defer mgr.Close()

	mcpToolName := "mcp_echo_echo"
	mcpTool, err := registry.Get(mcpToolName)
	if err != nil {
		names := []string{}
		for _, ti := range registry.GetAllInfo() {
			names = append(names, ti.Tool.Name())
		}
		t.Fatalf("MCP tool %s not registered; registry has %v", mcpToolName, names)
	}
	_ = mcpTool

	// Scripted loop: turn 1 calls the MCP tool, turn 2 finishes.
	fm := &fakeModel{turns: []fakeTurn{
		{final: modelWithCalls("", &genai.FunctionCall{
			ID:   "c1",
			Name: mcpToolName,
			Args: map[string]any{"message": "hello-from-adk"},
		})},
		{final: modelText("done")},
	}}
	a, err := NewWithModel(ctx, Options{Model: "fake", Tools: registry, Yolo: true}, fm)
	if err != nil {
		t.Fatalf("NewWithModel: %v", err)
	}
	cb := newRecordingCallback()
	if _, err := a.RunWithCallback(ctx, "echo something", cb); err != nil {
		t.Fatalf("RunWithCallback: %v", err)
	}

	res, ok := cb.results[mcpToolName]
	if !ok {
		t.Fatalf("MCP tool never ran; order=%v", cb.order)
	}
	if !strings.Contains(res, "ECHO: hello-from-adk") {
		t.Fatalf("MCP tool result = %q", res)
	}

	// No orphans: one call, one response.
	events := eventsSnapshot(t, a.sessionSvc, a.SessionID())
	calls, responses := 0, 0
	for _, ev := range events {
		if ev.Content == nil {
			continue
		}
		for _, part := range ev.Content.Parts {
			if part.FunctionCall != nil {
				calls++
			}
			if part.FunctionResponse != nil {
				responses++
			}
		}
	}
	if calls != 1 || responses != 1 {
		t.Fatalf("MCP loop pairing broken: calls=%d responses=%d", calls, responses)
	}
}

func isCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""
}

// compile-time guard: RunConfig referenced so imports stay honest
var _ = agent.RunConfig{}
