package adkagent

import (
	"strings"
	"testing"
)

// mkCalls feeds n identical tool calls with the given args to the detector.
func mkCalls(t *testing.T, d *stuckDetector, idPrefix, name string, args map[string]any, n int) (stuck bool, detail string) {
	t.Helper()
	for i := 0; i < n; i++ {
		d.beginEvent()
		s, detail := d.observe(idPrefix+string(rune('a'+i)), name, args)
		if s {
			return true, detail
		}
		// All calls succeed with the identical result, which is what makes
		// the repetition meaningless.
		resp := map[string]any{"result": "same output"}
		if s, detail := d.observeResult(idPrefix+string(rune('a'+i)), name, resp); s {
			return true, detail
		}
	}
	return false, detail
}

func TestStuckIdenticalCallStreakTriggers(t *testing.T) {
	var d stuckDetector
	args := map[string]any{"path": "main.go"}
	stuck, detail := mkCalls(t, &d, "c1-", "read", args, maxRepeatToolCalls)
	if !stuck {
		t.Fatalf("expected stuck after %d identical calls", maxRepeatToolCalls)
	}
	if !strings.Contains(detail, "read") {
		t.Fatalf("detail should name the tool, got %q", detail)
	}
}

func TestStuckVolatileArgsCollapseToStreak(t *testing.T) {
	var d stuckDetector
	// Paging through one file with varying offsets is the same call.
	for i := 0; i < maxRepeatToolCalls; i++ {
		d.beginEvent()
		args := map[string]any{"path": "log.txt", "offset": i * 50}
		if s, detail := d.observe("v"+string(rune('a'+i)), "read", args); s {
			_ = detail
			return // detector caught the loop
		}
		resp := map[string]any{"result": "same output"}
		if s, _ := d.observeResult("v"+string(rune('a'+i)), "read", resp); s {
			return
		}
	}
	t.Fatal("expected identical-call detection despite varying offset args")
}

func TestStuckChangingResultResetsStreak(t *testing.T) {
	var d stuckDetector
	// Poll-style: identical args but fresh output every time.
	for i := 0; i < maxRepeatToolCalls+3; i++ {
		d.beginEvent()
		args := map[string]any{"command": "tail -f app.log"}
		if s, detail := d.observe("p"+string(rune('a'+i)), "run", args); s {
			t.Fatalf("fresh results should not trip the detector: %s", detail)
		}
		resp := map[string]any{"result": "output line " + string(rune('0'+i))}
		if s, detail := d.observeResult("p"+string(rune('a'+i)), "run", resp); s {
			t.Fatalf("fresh results should not trip the detector: %s", detail)
		}
	}
}

func TestStuckErrorStreakArgsAware(t *testing.T) {
	var d stuckDetector
	fp := toolFingerprint("write", map[string]any{"path": "x.go", "content": "bad"})
	for i := 0; i < maxToolErrorStreak; i++ {
		d.beginEvent()
		if s, _ := d.observe("e"+string(rune('a'+i)), "write", map[string]any{"path": "x.go", "content": "bad"}); s && i < maxToolErrorStreak-1 {
			// observe may legitimately fire on the final attempt too: ten
			// identical calls without results is itself an identical-call
			// streak. Only an earlier fire is a bug.
			t.Fatalf("observe should not fire on errors before the streak ends: attempt %d", i)
		}
		s, detail := d.observeError("e"+string(rune('a'+i)), "write", true)
		if i < maxToolErrorStreak-1 && s {
			t.Fatalf("stuck too early at %d: %s", i+1, detail)
		}
		if i == maxToolErrorStreak-1 && !s {
			t.Fatalf("expected stuck after %d identical failures", maxToolErrorStreak)
		}
		_ = fp
	}
}

func TestStuckErrorSuccessResetsStreak(t *testing.T) {
	var d stuckDetector
	// 9 failures, one success, 9 more failures — must not trigger.
	for round := 0; round < 2; round++ {
		for i := 0; i < maxToolErrorStreak-1; i++ {
			d.beginEvent()
			_, _ = d.observe("r"+string(rune(round))+string(rune('a'+i)), "run", map[string]any{"command": "make"})
			if s, _ := d.observeError("r"+string(rune(round))+string(rune('a'+i)), "run", true); s {
				t.Fatal("stuck triggered before reaching streak")
			}
		}
		d.beginEvent()
		id := "ok" + string(rune('0'+round))
		_, _ = d.observe(id, "run", map[string]any{"command": "make"})
		_, _ = d.observeResult(id, "run", map[string]any{"result": "ok"})
		if s, detail := d.observeError(id, "run", false); s {
			t.Fatalf("success must reset the error streak: %s", detail)
		}
	}
}

func TestStuckCycleDetection(t *testing.T) {
	var d stuckDetector
	// Alternate two tools with identical results: a length-2 cycle.
	total := 6 // 3 repetitions of [a, b]
	for i := 0; i < total; i++ {
		d.beginEvent()
		name := "read"
		if i%2 == 1 {
			name = "search"
		}
		id := "z" + string(rune('a'+i))
		if s, detail := d.observe(id, name, map[string]any{"q": name}); s {
			t.Fatalf("identical-call detector should not fire on alternation: %s", detail)
		}
		if s, detail := d.observeResult(id, name, map[string]any{"result": "static"}); s {
			if i < total-1 {
				t.Fatalf("stuck early at %d: %s", i, detail)
			}
			if !strings.Contains(detail, "cycle") {
				t.Fatalf("expected cycle detail, got %q", detail)
			}
			return
		}
	}
	t.Fatal("expected cycle detection after 3 alternating repetitions")
}

func TestStuckOutputRepetition(t *testing.T) {
	var d stuckDetector
	phrase := "I will now try harder. " // 22 bytes, > minPeriodVariety distinct bytes
	var sb strings.Builder
	for sb.Len() < outputCheckEvery+outputProbeBytes*4 {
		sb.WriteString(phrase)
	}
	// Feed in small chunks like a real stream.
	text := sb.String()
	for len(text) > 0 {
		chunk := text
		if len(chunk) > 200 {
			chunk = chunk[:200]
		}
		text = text[len(chunk):]
		if s, detail := d.observeOutput(chunk); s {
			if !strings.Contains(detail, "repeated") {
				t.Fatalf("unexpected detail %q", detail)
			}
			return
		}
	}
	t.Fatal("expected output repetition detection")
}

func TestStuckOutputBenignPeriodicDoesNotTrigger(t *testing.T) {
	var d stuckDetector
	// Dashes are periodic but have no byte variety — must not trigger.
	filler := strings.Repeat("-", 2000)
	for i := 0; i < len(filler); i += 200 {
		if s, _ := d.observeOutput(filler[i : i+200]); s {
			t.Fatal("low-variety filler must not be flagged as repetition")
		}
	}
}

func TestStuckFingerprintStableUnderArgOrder(t *testing.T) {
	a := toolFingerprint("read", map[string]any{"path": "x", "limit": 10})
	b := toolFingerprint("read", map[string]any{"limit": 10, "path": "x"})
	if a != b {
		t.Fatal("fingerprint must not depend on argument order")
	}
	// limit is volatile — dropping it must not change the fingerprint.
	c := toolFingerprint("read", map[string]any{"path": "x"})
	if a != c {
		t.Fatal("volatile args must be excluded from the fingerprint")
	}
}
