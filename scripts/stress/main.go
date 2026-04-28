// OTel stress CLI — hammers the NestJS API and web to generate traces, RED metrics,
// and correlated logs. Displays a live TUI dashboard.
//
// Usage:
//
//	go run ./scripts/stress           # defaults
//	go run ./scripts/stress -d 2m -c 20
//	go run ./scripts/stress -h
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleGreen   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	styleRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	styleBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	styleStatBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("57")).Padding(0, 2).Width(20)
	styleHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).MarginBottom(1)
)

// ── endpoints ─────────────────────────────────────────────────────────────────

type endpoint struct{ method, url, label string }

var endpoints = []endpoint{
	{"GET", "http://localhost:3000/hello", "/hello  (API 200)"},
	{"GET", "http://localhost:3000/hello", "/hello  (API 200)"},
	{"GET", "http://localhost:3000/hello", "/hello  (API 200)"},
	{"GET", "http://localhost:3000/notfound", "/notfound (API 404 → error rate)"},
	{"GET", "http://localhost:5173", "/       (Web 200)"},
}

// ── metrics ───────────────────────────────────────────────────────────────────

type counters struct {
	total   atomic.Int64
	success atomic.Int64
	errors  atomic.Int64
	latency atomic.Int64 // cumulative ms
}

// per-endpoint hit ring (last 50 results per endpoint)
type resultEntry struct {
	label  string
	status int
	ms     int64
}

type shared struct {
	mu      sync.Mutex
	ring    []resultEntry
	ringCap int
}

func (s *shared) push(e resultEntry) {
	s.mu.Lock()
	if len(s.ring) >= s.ringCap {
		s.ring = s.ring[1:]
	}
	s.ring = append(s.ring, e)
	s.mu.Unlock()
}

func (s *shared) snapshot() []resultEntry {
	s.mu.Lock()
	out := make([]resultEntry, len(s.ring))
	copy(out, s.ring)
	s.mu.Unlock()
	return out
}

// ── bubbletea model ───────────────────────────────────────────────────────────

type tickMsg time.Time
type doneMsg struct{}

type model struct {
	cfg      config
	cnt      *counters
	sh       *shared
	start    time.Time
	finished bool
	quitting bool
	width    int
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tickMsg:
		elapsed := time.Since(m.start)
		if elapsed >= m.cfg.duration {
			m.finished = true
			return m, tea.Quit
		}
		return m, tick()
	case doneMsg:
		m.finished = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		m.width = 100
	}

	total := m.cnt.total.Load()
	success := m.cnt.success.Load()
	errs := m.cnt.errors.Load()
	var avgMs int64
	if total > 0 {
		avgMs = m.cnt.latency.Load() / total
	}
	var errPct float64
	if total > 0 {
		errPct = float64(errs) / float64(total) * 100
	}
	elapsed := time.Since(m.start).Round(time.Second)
	remaining := (m.cfg.duration - time.Since(m.start)).Round(time.Second)
	if remaining < 0 {
		remaining = 0
	}

	// progress bar
	barWidth := m.width - 20
	if barWidth < 10 {
		barWidth = 10
	}
	progress := float64(elapsed) / float64(m.cfg.duration)
	if progress > 1 {
		progress = 1
	}
	filled := int(float64(barWidth) * progress)
	bar := styleGreen.Render(strings.Repeat("█", filled)) +
		styleDim.Render(strings.Repeat("░", barWidth-filled))

	// stat boxes
	totalBox := styleStatBox.Render(styleBold.Render("Total") + "\n" + styleCyan.Render(fmt.Sprintf("%d", total)))
	okBox := styleStatBox.Render(styleBold.Render("Success") + "\n" + styleGreen.Render(fmt.Sprintf("%d", success)))

	errColor := styleGreen
	if errPct > 1 {
		errColor = styleYellow
	}
	if errPct > 5 {
		errColor = styleRed
	}
	errBox := styleStatBox.Render(styleBold.Render("Errors") + "\n" + errColor.Render(fmt.Sprintf("%d (%.1f%%)", errs, errPct)))

	latColor := styleGreen
	if avgMs > 200 {
		latColor = styleYellow
	}
	if avgMs > 1000 {
		latColor = styleRed
	}
	latBox := styleStatBox.Render(styleBold.Render("Avg Latency") + "\n" + latColor.Render(fmt.Sprintf("%dms", avgMs)))

	statsRow := lipgloss.JoinHorizontal(lipgloss.Top, totalBox, okBox, errBox, latBox)

	// recent requests log (last 12 lines)
	entries := m.sh.snapshot()
	var logLines []string
	start := len(entries) - 12
	if start < 0 {
		start = 0
	}
	for _, e := range entries[start:] {
		statusStr := styleGreen.Render(fmt.Sprintf("%d", e.status))
		if e.status == 0 {
			statusStr = styleRed.Render("ERR")
		} else if e.status >= 400 {
			statusStr = styleYellow.Render(fmt.Sprintf("%d", e.status))
		}
		logLines = append(logLines, fmt.Sprintf("  %s  %-32s %s",
			statusStr, e.label, styleDim.Render(fmt.Sprintf("%dms", e.ms))))
	}
	for len(logLines) < 12 {
		logLines = append(logLines, styleDim.Render("  —"))
	}
	logBlock := styleBox.Width(m.width - 4).Render(strings.Join(logLines, "\n"))

	// links
	links := styleDim.Render("Grafana → ") + styleGreen.Render("http://localhost:3001") +
		styleDim.Render("   Jaeger → ") + styleGreen.Render("http://localhost:16686") +
		styleDim.Render("   Prometheus → ") + styleGreen.Render("http://localhost:9090")

	status := styleGreen.Render("● RUNNING")
	if m.finished {
		status = styleYellow.Render("✓ DONE")
	}
	if m.quitting {
		status = styleRed.Render("✗ STOPPED")
	}

	return fmt.Sprintf(
		"\n%s  %s\n%s\n"+
			"  %s  %s  elapsed %s  remaining %s  workers %d\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"%s\n",
		styleTitle.Render("⚡ OTel Stress Generator"),
		status,
		styleDim.Render(strings.Repeat("─", m.width-2)),
		bar,
		fmt.Sprintf("%.0f%%", progress*100),
		styleCyan.Render(elapsed.String()),
		styleCyan.Render(remaining.String()),
		m.cfg.concurrency,
		statsRow,
		styleHeader.Render("  Recent Requests"),
		logBlock,
		links,
	)
}

// ── worker ────────────────────────────────────────────────────────────────────

func runWorker(id int, cfg config, client *http.Client, cnt *counters, sh *shared, stop <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	deadline := time.Now().Add(cfg.duration)
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*997))

	for time.Now().Before(deadline) {
		select {
		case <-stop:
			return
		default:
		}

		ep := endpoints[rng.Intn(len(endpoints))]
		req, _ := http.NewRequest(ep.method, ep.url, nil)
		req.Header.Set("traceparent",
			fmt.Sprintf("00-%032x-%016x-01", rng.Int63(), rng.Int63()))

		t0 := time.Now()
		resp, err := client.Do(req)
		ms := time.Since(t0).Milliseconds()

		cnt.total.Add(1)
		cnt.latency.Add(ms)

		entry := resultEntry{label: ep.label, ms: ms}
		if err != nil {
			cnt.errors.Add(1)
		} else {
			resp.Body.Close()
			entry.status = resp.StatusCode
			if resp.StatusCode >= 500 {
				cnt.errors.Add(1)
			} else {
				cnt.success.Add(1)
			}
		}
		sh.push(entry)

		time.Sleep(time.Duration(50+rng.Intn(350)) * time.Millisecond)
	}
}

// ── config + main ─────────────────────────────────────────────────────────────

type config struct {
	duration    time.Duration
	concurrency int
}

func main() {
	dur := flag.Duration("d", 60*time.Second, "test duration (e.g. 60s, 2m, 5m)")
	con := flag.Int("c", 10, "concurrent workers")
	flag.Parse()

	cfg := config{duration: *dur, concurrency: *con}
	cnt := &counters{}
	sh := &shared{ringCap: 200}
	client := &http.Client{Timeout: 5 * time.Second}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 1; i <= cfg.concurrency; i++ {
		wg.Add(1)
		go runWorker(i, cfg, client, cnt, sh, stop, &wg)
	}

	m := model{cfg: cfg, cnt: cnt, sh: sh, start: time.Now()}
	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		wg.Wait()
		p.Send(doneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		close(stop)
		os.Exit(1)
	}

	close(stop)
	wg.Wait()

	// Final summary printed after TUI exits
	total := cnt.total.Load()
	var avgMs int64
	if total > 0 {
		avgMs = cnt.latency.Load() / total
	}
	fmt.Printf("\nFinal: %d requests | %d success | %d errors | %dms avg latency\n",
		total, cnt.success.Load(), cnt.errors.Load(), avgMs)
	fmt.Println("Grafana  → http://localhost:3001  (NestJS API — RED Metrics)")
	fmt.Println("Jaeger   → http://localhost:16686")
}
