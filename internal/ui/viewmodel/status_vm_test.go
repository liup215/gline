package viewmodel

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"

	"github.com/liup215/gline/internal/agent"
)

func TestNewStatusViewModel(t *testing.T) {
	vm := NewStatusViewModel()
	if vm == nil {
		t.Fatal("NewStatusViewModel returned nil")
	}

	// Verify spinner is initialized - just check it's not nil
	if reflect.ValueOf(vm.spinner).IsZero() {
		t.Error("Expected spinner to be initialized")
	}
}

func TestStatusViewModelRefresh(t *testing.T) {
	vm := NewStatusViewModel()

	data := vm.Refresh(
		agent.ModeAct,
		"openai",
		"gpt-4",
		true,   // isProcessing
		false,  // isStreaming
		"test_tool",
		80,     // width
	)

	if data.Mode != agent.ModeAct {
		t.Errorf("Expected Mode=Act, got %v", data.Mode)
	}
	if data.Provider != "openai" {
		t.Errorf("Expected Provider=openai, got %s", data.Provider)
	}
	if data.ModelName != "gpt-4" {
		t.Errorf("Expected ModelName=gpt-4, got %s", data.ModelName)
	}
	if !data.IsProcessing {
		t.Error("Expected IsProcessing=true")
	}
	if data.IsStreaming {
		t.Error("Expected IsStreaming=false")
	}
	if data.CurrentTool != "test_tool" {
		t.Errorf("Expected CurrentTool=test_tool, got %s", data.CurrentTool)
	}
	if data.Width != 80 {
		t.Errorf("Expected Width=80, got %d", data.Width)
	}

	// Verify data is stored in vm
	storedData := vm.Data()
	if storedData.Provider != "openai" {
		t.Error("Data() should return the stored data")
	}
}

func TestStatusViewModelData(t *testing.T) {
	vm := NewStatusViewModel()

	// Initially empty data
	data := vm.Data()
	if data.Provider != "" {
		t.Error("Expected empty Provider initially")
	}

	// After refresh
	vm.Refresh(agent.ModePlan, "anthropic", "claude-3", false, false, "", 100)
	data = vm.Data()
	if data.Mode != agent.ModePlan {
		t.Errorf("Expected Mode=Plan, got %v", data.Mode)
	}
	if data.Provider != "anthropic" {
		t.Errorf("Expected Provider=anthropic, got %s", data.Provider)
	}
}

func TestStatusViewModelSetWidth(t *testing.T) {
	vm := NewStatusViewModel()
	vm.Refresh(agent.ModeAct, "openai", "gpt-4", false, false, "", 80)

	vm.SetWidth(120)
	data := vm.Data()
	if data.Width != 120 {
		t.Errorf("Expected Width=120 after SetWidth, got %d", data.Width)
	}
}

func TestStatusViewModelSetSpinnerStyle(t *testing.T) {
	vm := NewStatusViewModel()

	// Get the initial spinner state
	initialSpinner := vm.spinner.Spinner

	// Change to a different spinner style
	vm.SetSpinnerStyle(spinner.Line)

	// Verify spinner style was changed
	if reflect.DeepEqual(vm.spinner.Spinner, initialSpinner) {
		t.Error("Expected spinner style to be changed")
	}

	// Change back
	vm.SetSpinnerStyle(spinner.Dot)

	// Verify spinner style was changed again
	if reflect.DeepEqual(vm.spinner.Spinner, spinner.Line) {
		t.Error("Expected spinner style to be changed to Dot")
	}
}

func TestStatusViewModelTick(t *testing.T) {
	vm := NewStatusViewModel()

	// Without processing, Tick should return false
	vm.Refresh(agent.ModeAct, "openai", "gpt-4", false, false, "", 80)
	if vm.Tick() {
		t.Error("Tick should return false when not processing")
	}

	// With processing, Tick should return true
	vm.Refresh(agent.ModeAct, "openai", "gpt-4", true, false, "", 80)
	if !vm.Tick() {
		t.Error("Tick should return true when processing")
	}
}

func TestStatusViewModelRender(t *testing.T) {
	vm := NewStatusViewModel()
	vm.Refresh(agent.ModeAct, "openai", "gpt-4", true, true, "", 80)

	rendered := vm.Render()
	if rendered == "" {
		t.Error("Render should return non-empty string")
	}

	// The rendered output should contain "gline"
	if !contains(rendered, "gline") {
		t.Error("Rendered output should contain 'gline'")
	}

	// CompactBar only shows spinner + gline, not "AI" text
	// The AI responding status is shown elsewhere (in conversation as system message)
}

func TestStatusViewModelRenderNotProcessing(t *testing.T) {
	vm := NewStatusViewModel()
	vm.Refresh(agent.ModePlan, "anthropic", "claude-3", false, false, "", 80)

	rendered := vm.Render()
	if rendered == "" {
		t.Error("Render should return non-empty string")
	}

	// Should contain gline and provider info
	if !contains(rendered, "gline") {
		t.Error("Rendered output should contain 'gline'")
	}
}

func TestStatusViewModelRenderWithTool(t *testing.T) {
	vm := NewStatusViewModel()
	vm.Refresh(agent.ModeAct, "openai", "gpt-4", true, false, "read_file", 80)

	rendered := vm.Render()
	if rendered == "" {
		t.Error("Render should return non-empty string")
	}

	// CompactBar only shows spinner + gline, not tool name
	// Tool status is shown in conversation as system message (🔧 icon)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
