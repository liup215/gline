// Package viewmodel derives rendered display state from model data.
// It owns the rendering state and produces display-ready strings.
// It has no Bubbletea dependencies.
package viewmodel

import (
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/liup215/gline/internal/agent"
	"github.com/liup215/gline/internal/ui/view"
)

// StatusViewModel manages the state for rendering the compact status bar.
// It holds the spinner model and derives StatusBarData from application state.
type StatusViewModel struct {
	data    view.CompactBarData
	spinner spinner.Model
}

// NewStatusViewModel creates a new StatusViewModel ready for use.
func NewStatusViewModel() *StatusViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return &StatusViewModel{
		spinner: s,
		data:    view.CompactBarData{},
	}
}

// Refresh updates the StatusViewModel state from application state.
// This should be called whenever any of the input parameters change.
func (vm *StatusViewModel) Refresh(
	mode agent.Mode,
	provider string,
	modelName string,
	isProcessing bool,
	isStreaming bool,
	currentTool string,
	width int,
) view.CompactBarData {
	vm.data = view.CompactBarData{
		Mode:         mode,
		Provider:     provider,
		ModelName:    modelName,
		IsProcessing: isProcessing,
		IsStreaming:  isStreaming,
		CurrentTool:  currentTool,
		SpinnerView:  vm.spinner.View(),
		Width:        width,
	}
	return vm.data
}

// Data returns the current StatusBarData.
// Useful for testing or when the caller needs direct access to the data.
func (vm *StatusViewModel) Data() view.CompactBarData {
	return vm.data
}

// Tick advances the spinner state.
// Returns true if the spinner ticked (i.e., the view should be updated).
func (vm *StatusViewModel) Tick() bool {
	// The spinner view changes on each tick when spinning
	return vm.data.IsProcessing
}

// UpdateSpinner updates the spinner model.
// This should be called when receiving a spinner.TickMsg from Bubbletea.
func (vm *StatusViewModel) UpdateSpinner(msg spinner.TickMsg) {
	vm.spinner, _ = vm.spinner.Update(msg)
	vm.data.SpinnerView = vm.spinner.View()
}

// Render returns the rendered status bar string.
// This is a convenience method that calls view.RenderCompactBar with the current data.
func (vm *StatusViewModel) Render() string {
	return view.RenderCompactBar(vm.data)
}

// SetWidth updates the width of the status bar.
func (vm *StatusViewModel) SetWidth(width int) {
	vm.data.Width = width
}

// SetSpinnerStyle sets the style of the spinner.
func (vm *StatusViewModel) SetSpinnerStyle(style spinner.Spinner) {
	vm.spinner.Spinner = style
}
