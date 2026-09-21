package adkagent

// stuckDetector detects degenerate agent-loop shapes — repeated identical
// tool calls, alternating call cycles, consecutive tool errors and repetitive
// model output — ported from pi-go's internal/tui/agent_loop.go (Phase 5b).
//
// The detector is a passive observer: the event bridge feeds it every tool
// call, tool result and text chunk, and when it reports stuck the run is
// aborted with a descriptive error. pi-go additionally hands the reason back
// to the model for a bounded number of recovery attempts; that resumption
// loop is deferred to Phase 6, where it needs an outer-run wrapper.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// maxRepeatToolCalls is the number of identical consecutive tool calls
	// before the loop is considered stuck. A repeated call whose result
	// changes resets the count (see observeResult).
	maxRepeatToolCalls = 10

	// maxRunningPollRepeats is the identical-call threshold that replaces
	// maxRepeatToolCalls while a bash poll's command is still running.
	// gline has no bash_wait/bash_output tools yet, so this branch is
	// currently unreachable; it is kept so the port stays drop-in when a
	// background-run poll tool arrives.
	maxRunningPollRepeats = 500

	// maxToolErrorStreak is the number of consecutive failures of the same
	// tool before the loop is aborted. Catches the "flailing" pattern where
	// the model tries a different argument each turn but the call still
	// fails.
	maxToolErrorStreak = 10

	// recentWindowSize is the sliding window of tool-call fingerprints kept
	// for repetition detection.
	recentWindowSize = 12

	// maxOutputRepeats is the number of back-to-back copies of one phrase in
	// the model's own output before the turn is called degenerate.
	maxOutputRepeats = 6

	// outputWindowBytes is the rolling tail of streamed output kept for
	// repetition detection. It has to hold maxOutputRepeats copies of the
	// longest phrase worth catching (~680 bytes).
	outputWindowBytes = 8192

	// outputProbeBytes is the suffix matched against earlier output to find
	// the length of the phrase being repeated.
	outputProbeBytes = 48

	// minOutputPeriod is the shortest phrase treated as a repetition unit.
	// Below this, ordinary output (indentation, ASCII art) is periodic often
	// enough to matter.
	minOutputPeriod = 16

	// minPeriodVariety is how many distinct bytes the repeating phrase must
	// contain. A rule of dashes or a run of spaces is perfectly periodic and
	// perfectly harmless; a sentence the model cannot stop restating is not.
	minPeriodVariety = 8

	// outputCheckEvery is how much new output accumulates between scans.
	// Scanning per token would put a linear search in the streaming path.
	outputCheckEvery = 512
)

// stuckError signals that the loop was called dead by the detectors.
type stuckError struct{ detail string }

func (e *stuckError) Error() string { return "agent loop aborted: " + e.detail }

// Detail returns the human-readable reason the loop was called dead.
func (e *stuckError) Detail() string { return e.detail }

// stuckErr adapts a stuckDetector verdict into an error, so both detector call
// sites read as a single guard instead of a repeated five-line block.
func stuckErr(stuck bool, detail string) error {
	if !stuck {
		return nil
	}
	return &stuckError{detail: detail}
}

// stuckDetector tracks recent tool calls and detects repetition loops.
type stuckDetector struct {
	recent     []recentCall // ring of call+result pairs (len <= recentWindowSize)
	recentSeq  int          // monotonic id handed to each entry in recent
	lastPrint  string       // fingerprint of last tool call
	lastName   string       // tool name behind lastPrint
	lastResult string       // fingerprint of the streak's previous result ("" = none yet)
	streak     int          // consecutive identical tool calls with identical results
	// livePoll is set while the streak's tool is a bash poll whose last
	// response reported the command still running. It swaps the identical-call
	// threshold for maxRunningPollRepeats — see repeatLimit.
	livePoll bool
	// Two error detectors run in parallel. The args-aware streak keys on the
	// call fingerprint, so only the same tool call failing repeatedly counts;
	// the name-only streak keys on the tool name and compounds only across
	// distinct model messages, so a batch of different calls in one message is
	// a single attempt rather than a streak. See observeError.
	callInfo       map[string]callRecord // call ID -> fingerprint + message, for response correlation
	lastCallByName map[string]callRecord // name -> most recent call, fallback when a response carries no/unknown ID
	msgSeq         int                   // per-event counter; calls in the same event share it
	lastErrFP      string                // fingerprint of the previous error (args-aware streak)
	errFPStreak    int                   // consecutive errors of the same call fingerprint
	lastErrTool    string                // name of last tool that errored
	errStreak      int                   // consecutive errors of that tool across distinct messages
	lastErrBatch   int                   // message sequence of the call the last error answered
	outBuf         string                // rolling tail of the model's own output
	outSince       int                   // bytes of output since the last repetition scan
}

// callRecord is what the detector remembers about an observed tool call so a
// later response can be correlated back to it: the call's fingerprint (for the
// args-aware error streak), the message it appeared in (for the name-only
// cross-batch streak), and its slot in the repetition window (so the response
// can be filed against the call it answers rather than against whichever call
// happened to be observed last).
type callRecord struct {
	fp  string
	msg int
	seq int
}

// recentCall is one entry in the repetition window: a call, and what came back
// from it. Both halves matter. Comparing calls alone cannot tell a model
// spinning on a dead pattern from one driving several long-running jobs — the
// fan-out of a parallel poll is a repeating sequence of fingerprints by
// construction, and the only thing separating it from a genuine loop is that
// its results keep changing.
type recentCall struct {
	call   string // call fingerprint
	result string // result fingerprint; "" until the response arrives
	seq    int    // matches the callRecord that produced it
}

// repeats reports whether c re-ran prev and got nothing new back.
func (c recentCall) repeats(prev recentCall) bool {
	return c.call == prev.call && c.result == prev.result
}

// volatileToolArgs address a slice of a target rather than the target itself.
// A model paging through one file — read(x, offset 1), read(x, offset 230),
// read(x, offset 240) — is repeating itself, but hashing the raw args makes
// every one of those calls unique and hides the loop from the detector.
var volatileToolArgs = map[string]bool{
	"offset":     true,
	"limit":      true,
	"head_limit": true,
	"start_line": true,
	"end_line":   true,
}

// toolFingerprint produces a short hash of a tool call for comparison.
// Pagination arguments are dropped first, so re-reading one file region by
// region collapses to a single fingerprint. json.Marshal sorts map keys, so
// the hash does not depend on argument order.
func toolFingerprint(name string, args map[string]any) string {
	h := sha256.New()
	h.Write([]byte(name))
	b, _ := json.Marshal(stableToolArgs(args))
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// stableToolArgs returns args without the volatile keys. The input map belongs
// to the event being streamed, so it is copied rather than filtered in place.
func stableToolArgs(args map[string]any) map[string]any {
	stable := make(map[string]any, len(args))
	for k, v := range args {
		if volatileToolArgs[k] {
			continue
		}
		stable[k] = v
	}
	return stable
}

// beginEvent advances the message counter so calls in one model event share a
// batch identity. Responses later use it to tell a single batch of calls from
// repeated attempts spread across turns.
func (s *stuckDetector) beginEvent() {
	s.msgSeq++
}

// observe records a tool call and returns true if the loop appears stuck.
// id correlates the call to its later response (FunctionCall.ID). Calls with
// an empty id cannot be correlated by ID, so they are remembered only by name
// (see lastCallByName); observeError uses that fallback so a batch of ID-less
// calls still reads as one message.
func (s *stuckDetector) observe(id, name string, args map[string]any) (stuck bool, detail string) {
	fp := toolFingerprint(name, args)
	rec := callRecord{fp: fp, msg: s.msgSeq, seq: s.recentSeq}
	if id != "" {
		if s.callInfo == nil {
			s.callInfo = make(map[string]callRecord)
		}
		s.callInfo[id] = rec
	}
	if s.lastCallByName == nil {
		s.lastCallByName = make(map[string]callRecord)
	}
	s.lastCallByName[name] = rec

	// Consecutive identical call detection.
	if fp == s.lastPrint {
		s.streak++
	} else {
		s.streak = 1
		s.lastPrint = fp
		s.lastName = name
		s.lastResult = ""
		s.livePoll = false
	}

	// Sliding window. The result half is filled in later, by observeResult.
	s.recent = append(s.recent, recentCall{call: fp, seq: s.recentSeq})
	s.recentSeq++
	if len(s.recent) > recentWindowSize {
		s.recent = s.recent[1:]
	}

	if limit := s.repeatLimit(); s.streak >= limit {
		return true, fmt.Sprintf("identical tool call %q repeated %d times", name, s.streak)
	}

	// Cycle detection runs from observeResult instead: a cycle is only a cycle
	// once the results are in, and at this point the call just observed has no
	// result yet.
	return false, ""
}

// repeatLimit is how many identical consecutive calls the current streak is
// allowed before it counts as stuck. Waiting on a live command gets a much
// larger budget than repeating a call against something static.
func (s *stuckDetector) repeatLimit() int {
	if s.livePoll {
		return maxRunningPollRepeats
	}
	return maxRepeatToolCalls
}

// observeResult records a tool call's response. Polling tools repeat identical
// calls by design — a poll on a running command sends the same handle every
// time and gets fresh output back — so a response that differs from the
// streak's previous response is progress, and resets the identical-call
// streak. A response identical to the last one keeps the streak counting:
// re-polling a finished command or re-reading an unchanged file is genuine
// repetition.
//
// It also closes the repetition window entry for the call it answers and runs
// cycle detection, which is why it reports a verdict: a cycle is a claim about
// calls *and* their results, so it cannot be decided when the call is observed.
func (s *stuckDetector) observeResult(id, name string, response map[string]any) (stuck bool, detail string) {
	fp := resultFingerprint(name, response)
	rec, known := s.callFor(id, name)
	if known {
		s.fillResult(rec.seq, fp)
	}

	// The streak only tracks one call at a time, so a result belonging to some
	// other call — another handle in the same parallel batch, or another tool
	// entirely — must not be compared against it.
	if name == s.lastName && (!known || rec.fp == s.lastPrint) {
		if s.lastResult != "" && fp != s.lastResult {
			s.streak = 0
		}
		s.lastResult = fp
		s.livePoll = isLiveBashPoll(name, response)
	}

	if cycle := s.detectCycle(); cycle != "" {
		return true, fmt.Sprintf("repeating tool cycle detected: %s", cycle)
	}
	return false, ""
}

// resultFingerprint hashes a tool response for comparison against the previous
// response to the same call.
//
// bash_wait-like tools include elapsed/idle fields that change on every poll
// even when the command produced no new output. Those fields are progress for
// the UI, not progress from the command, so exclude them from loop detection.
func resultFingerprint(name string, response map[string]any) string {
	stable := response
	if isBashPoll(name) {
		stable = make(map[string]any, len(response))
		for key, value := range response {
			if key != "elapsed" && key != "idle" {
				stable[key] = value
			}
		}
	}
	b, _ := json.Marshal(stable)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16]
}

// isBashPoll reports whether name is one of the polling tools that re-check a
// backgrounded command. gline has no such tools today (run is synchronous);
// the hook stays so the detector keeps its pi-go semantics when one lands.
func isBashPoll(name string) bool {
	return name == "bash_wait" || name == "bash_output"
}

// isLiveBashPoll reports whether response is a poll of a command that has not
// finished yet.
func isLiveBashPoll(name string, response map[string]any) bool {
	if !isBashPoll(name) {
		return false
	}
	running, _ := response["running"].(bool)
	return running
}

// callFor resolves the call a response belongs to, by ID where the provider
// supplies one and by tool name otherwise — the same fallback observeError
// uses for ID-less calls.
func (s *stuckDetector) callFor(id, name string) (callRecord, bool) {
	if id != "" {
		if rec, ok := s.callInfo[id]; ok {
			return rec, true
		}
	}
	rec, ok := s.lastCallByName[name]
	return rec, ok
}

// fillResult attaches a result fingerprint to its call in the repetition
// window. The entry is gone if the window has already slid past it, which is
// not an error: a window that old cannot be part of a cycle being detected now.
func (s *stuckDetector) fillResult(seq int, fp string) {
	for i := range s.recent {
		if s.recent[i].seq == seq {
			s.recent[i].result = fp
			return
		}
	}
}

// observeError records the outcome of a tool call and reports whether the loop
// is stuck. Two streaks run in parallel, catching different loop shapes:
//
//  1. Args-aware (errFPStreak): the same call — same tool and same argument
//     fingerprint — failing maxToolErrorStreak times.
//
//  2. Name-only, cross-batch (errStreak): consecutive failures of the same
//     tool name that come from calls in *different* model messages. A batch of
//     distinct calls sent in one message is one attempt, not ten, so it does
//     not compound; the same tool failing once per message across ten messages
//     is the flailing pattern and does.
//
// A success (isError == false) resets both streaks. id is the FunctionResponse
// ID, used to look up which call — and therefore which message — this result
// answers.
func (s *stuckDetector) observeError(id, name string, isError bool) (stuck bool, detail string) {
	// Correlate this response to the call it answers. By ID first: a matched
	// record gives the call's fingerprint and message, and is consumed so the
	// map stays bounded by outstanding (unanswered) calls. When the response
	// carries no usable ID, fall back to the most recent call of that name —
	// responses trail their calls in order, so this is the call being answered.
	// Only the batch is trusted from the name fallback: without an ID we cannot
	// pair a response to a specific call's arguments, so the args-aware streak
	// must not see a fingerprint — a batch of distinct ID-less calls would all
	// collapse onto the last call's fingerprint and false-abort.
	fp := ""
	batch := 0
	if rec, ok := s.callInfo[id]; ok {
		fp, batch = rec.fp, rec.msg
		delete(s.callInfo, id)
	} else if rec, ok := s.lastCallByName[name]; ok {
		batch = rec.msg
	}

	if isError {
		if fp != "" && fp == s.lastErrFP {
			s.errFPStreak++
		} else {
			s.errFPStreak = 1
			s.lastErrFP = fp
		}
		if s.errFPStreak >= maxToolErrorStreak {
			return true, fmt.Sprintf("tool %q failed %d times in a row", name, s.errFPStreak)
		}

		// Name-only streak. Compound only when this failure answers a call
		// from a different message than the previous one (or from an
		// unknown message — treat unobserved calls as new attempts).
		if name == s.lastErrTool {
			if s.lastErrBatch == 0 || batch != s.lastErrBatch {
				s.errStreak++
				s.lastErrBatch = batch
			}
		} else {
			s.errStreak = 1
			s.lastErrTool = name
			s.lastErrBatch = batch
		}
		if s.errStreak >= maxToolErrorStreak {
			return true, fmt.Sprintf("tool %q failed %d times in a row", name, s.errStreak)
		}
		return false, ""
	}

	s.errFPStreak = 0
	s.lastErrFP = ""
	s.errStreak = 0
	s.lastErrTool = ""
	s.lastErrBatch = 0
	return false, ""
}

// observeOutput records a chunk of the model's own output — reply text or
// thinking — and reports whether the turn has collapsed into repetition.
//
// The detectors above watch tool calls, so a turn that makes no calls at all is
// invisible to them. That is exactly the shape of a degenerate turn: one
// sentence restated until the output cap is hit, with nothing else emitted.
// This scans the tail of the stream for a repeating period instead.
func (s *stuckDetector) observeOutput(text string) (stuck bool, detail string) {
	if text == "" {
		return false, ""
	}
	s.outBuf += text
	if len(s.outBuf) > outputWindowBytes {
		s.outBuf = s.outBuf[len(s.outBuf)-outputWindowBytes:]
	}

	s.outSince += len(text)
	if s.outSince < outputCheckEvery {
		return false, ""
	}
	s.outSince = 0

	period := repeatPeriod(s.outBuf)
	if period < minOutputPeriod || !isPeriodic(s.outBuf, period, maxOutputRepeats) {
		return false, ""
	}
	if !hasVariety(s.outBuf[len(s.outBuf)-period:]) {
		return false, ""
	}
	return true, fmt.Sprintf("model repeated a %d-character phrase %d times", period, maxOutputRepeats)
}

// repeatPeriod returns the distance between the tail of buf and the previous
// occurrence of that same tail — the length of the phrase the model may be
// cycling on — or 0 when the tail does not recur.
func repeatPeriod(buf string) int {
	if len(buf) < outputProbeBytes*2 {
		return 0
	}
	probe := buf[len(buf)-outputProbeBytes:]
	prev := strings.LastIndex(buf[:len(buf)-outputProbeBytes], probe)
	if prev < 0 {
		return 0
	}
	return len(buf) - outputProbeBytes - prev
}

// hasVariety reports whether unit contains enough distinct bytes to be a
// phrase rather than filler.
func hasVariety(unit string) bool {
	var seen [256]bool
	distinct := 0
	for i := range len(unit) {
		if seen[unit[i]] {
			continue
		}
		seen[unit[i]] = true
		distinct++
		if distinct >= minPeriodVariety {
			return true
		}
	}
	return false
}

// isPeriodic reports whether the last period*repeats bytes of buf are one
// period-long phrase repeated back to back. Comparing bytes is safe for UTF-8
// here: a byte-exact repeat is a rune-exact repeat.
func isPeriodic(buf string, period, repeats int) bool {
	span := period * repeats
	if period <= 0 || span > len(buf) {
		return false
	}
	tail := buf[len(buf)-span:]
	for i := period; i < len(tail); i++ {
		if tail[i] != tail[i-period] {
			return false
		}
	}
	return true
}

// detectCycle checks the recent window for repeating subsequences.
// Returns a description if found, empty string otherwise.
//
// A "cycle" requires that consecutive elements differ — a uniform window
// like [a,a,a,a,a,a] is a streak, not a cycle, and the identical-call
// detector above already handles that case at maxRepeatToolCalls.
//
// It also requires every repetition to have produced the same result as the
// one before it. Distinct calls arranged in a repeating order are not on their
// own evidence of anything: a model driving four background jobs polls them in
// the same order every message, and a rotation of that fan-out is a perfect
// length-3 cycle across message boundaries while every poll returns fresh
// output. Without the result half, that shape aborts a working run.
func (s *stuckDetector) detectCycle() string {
	window := s.completedWindow()
	n := len(window)
	if n < 6 {
		return ""
	}
	// Check cycle lengths 2 and 3.
	for cycleLen := 2; cycleLen <= 3; cycleLen++ {
		need := cycleLen * 3 // require 3 full repetitions
		if n < need {
			continue
		}
		tail := window[n-need:]
		cycle := tail[:cycleLen]
		// Require adjacent elements in the candidate cycle to differ —
		// otherwise it's a uniform streak, not an alternating cycle.
		cycleValid := true
		for i := 1; i < cycleLen; i++ {
			if cycle[i].call == cycle[i-1].call {
				cycleValid = false
				break
			}
		}
		if !cycleValid {
			continue
		}
		match := true
		for i := cycleLen; i < need; i++ {
			if !tail[i].repeats(cycle[i%cycleLen]) {
				match = false
				break
			}
		}
		if match {
			return fmt.Sprintf("length-%d cycle repeated %d times", cycleLen, need/cycleLen)
		}
	}
	return ""
}

// completedWindow returns the prefix of the repetition window whose results
// have all arrived. Entries still missing a result are excluded: a cycle needs
// call+result pairs to be evidence.
func (s *stuckDetector) completedWindow() []recentCall {
	for i := range s.recent {
		if s.recent[i].result == "" {
			return s.recent[:i]
		}
	}
	return s.recent
}
