package tui

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"reality-scanner-rev/internal/checker"
	"reality-scanner-rev/internal/scanner"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Confirm quit
		if m.confirmQuit {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			default:
				m.confirmQuit = false
				return m, nil
			}
		}

		// Search mode
		if m.searching {
			if key.Matches(msg, m.keys.Back) {
				m.searching = false
				return m, nil
			}
			if msg.String() == "enter" {
				m.searching = false
				m.applySort()
				return m, nil
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			m.applySort()
			return m, cmd
		}

		// Manual verify
		if m.view == verifyView {
			if key.Matches(msg, m.keys.Back) {
				m.view = inputView
				return m, nil
			}
			if msg.String() == "enter" && !m.verifying {
				domain := m.verifyInput.Value()
				if domain != "" {
					m.verifying = true
					go func() {
						cfg := checker.CheckConfig{Timeout: 10}
						c := checker.NewChecker(cfg)
						r := c.Check(domain)
						item := ScanResultItem{}
						item.checkResult = r
						m.verifyResultsCh <- item
					}()
				}
				return m, nil
			}
			if m.verifying {
				return m, nil
			}
			var cmd tea.Cmd
			m.verifyInput, cmd = m.verifyInput.Update(msg)
			return m, cmd
		}

		// Global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.phase == phaseTLSScanning || m.phase == phaseVerifying || len(m.scanResults) > 0 {
				m.confirmQuit = true
				return m, nil
			}
			return m, tea.Quit
		case key.Matches(msg, m.keys.Back):
			if m.view == detailView {
				m.view = resultsView
				return m, nil
			}
			if m.view != inputView {
				if m.cancelFn != nil { m.cancelFn() }
				m.view = inputView; m.phase = phaseIdle
				return m, nil
			}
		}

		switch m.view {
		case inputView:
			return m.updateInput(msg)
		case resultsView:
			return m.updateResults(msg)
		case detailView:
			return m.updateDetail(msg)
		case scanningView, verifyingView:
			return m, nil
		}

	case scanProgressMsg:
		m.scanScanned = msg.scanned
		if msg.total > 0 { m.scanTotal = msg.total; m.progress = float64(msg.scanned) / float64(msg.total) }
		return m, nil

	case logMsg:
		if len(m.logs) >= m.maxLogs { m.logs = m.logs[len(m.logs)-m.maxLogs+1:] }
		m.logs = append(m.logs, LogEntry(msg))
		return m, nil

	case scanResultMsg:
		r := ScanResultItem(msg)
		m.scanResults = append(m.scanResults, r)
		if r.Feasible {
			m.feasibleCount++
			if !m.profile.QuickMode() { go m.verifyDomainItem(r) }
		}
		return m, nil

	case scanDoneMsg:
		if m.profile.QuickMode() {
			m.phase = phaseDone; m.view = resultsView; m.applySort()
			return m, nil
		}
		m.phase = phaseVerifying; m.view = verifyingView
		m.verifyTotal = m.feasibleCount
		return m, nil

	case verifyResultMsg:
		r := ScanResultItem(msg)
		// Manual verify result
		if m.view == verifyView {
			m.verifyItem = &r
			m.verifying = false
			return m, nil
		}
		for i, s := range m.scanResults {
			if s.Origin == r.Origin {
				m.scanResults[i].checkResult = r.checkResult
				break
			}
		}
		m.verifyChecked++
		if m.verifyChecked >= m.verifyTotal {
			m.phase = phaseDone; m.view = resultsView; m.applySort()
			return m, nil
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case errMsg:
		m.err = msg
		return m, nil
	}
	return m, nil
}

func (m *Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Start):
		return m.startScan()
	case key.Matches(msg, m.keys.Up):
		if m.inputMode > modeAddr { m.inputMode-- }
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.inputMode < modeURL { m.inputMode++ }
		return m, nil
	case key.Matches(msg, m.keys.Profile):
		m.profile = (m.profile + 1) % 4
		m.threadInput.SetValue(fmt.Sprint(m.profile.Threads()))
		m.timeoutInput.SetValue(fmt.Sprint(m.profile.Timeout()))
		return m, nil
	case key.Matches(msg, m.keys.SaveCfg):
		cfg := &SavedConfig{
			Profile: int(m.profile), InputMode: int(m.inputMode),
			Port: m.portInput.Value(), Threads: m.threadInput.Value(),
			Timeout: m.timeoutInput.Value(), MaxTargets: m.maxTargetInput.Value(),
			EnableIPv6: m.enableIPv6, Verbose: m.verbose,
		}
		SaveConfig(cfg)
		select { case m.logCh <- LogEntry{text: "✓ 配置已保存", isSuccess: true}: default: }
		return m, nil
	case key.Matches(msg, m.keys.FavOnly):
		m.showFavOnly = !m.showFavOnly
		return m, nil
	case msg.String() == "i":
		m.enableIPv6 = !m.enableIPv6; return m, nil
	case msg.String() == "v":
		m.verbose = !m.verbose; return m, nil
	case msg.String() == "c":
		return m.importCSV()
	case msg.String() == "t":
		m.view = verifyView; return m, nil
	}
	var cmd tea.Cmd
	switch m.inputMode {
	case modeAddr, modeFile:
		m.targetInput, cmd = m.targetInput.Update(msg)
		if m.inputMode == modeFile { m.filePath = m.targetInput.Value() }
	case modeURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) startScan() (tea.Model, tea.Cmd) {
	port, threads, timeout := 443, m.profile.Threads(), m.profile.Timeout()
	fmt.Sscanf(m.portInput.Value(), "%d", &port)
	fmt.Sscanf(m.threadInput.Value(), "%d", &threads)
	fmt.Sscanf(m.timeoutInput.Value(), "%d", &timeout)

	m.view = scanningView; m.phase = phaseTLSScanning
	m.scanResults = make([]ScanResultItem, 0)
	m.feasibleCount = 0; m.logs = make([]LogEntry, 0)
	m.startTime = time.Now()
	m.scanGen++
	gen := m.scanGen

	cancel := make(chan struct{})
	m.cancelScan = cancel
	m.cancelFn = func() {
		select { case <-cancel: default: close(cancel) }
	}

	m.scanResultsCh = make(chan ScanResultItem, 1000)
	m.verifyResultsCh = make(chan ScanResultItem, 1000)
	m.logCh = make(chan LogEntry, 1000)
	m.scanDoneCh = make(chan struct{})
	m.progressCh = make(chan ScanProgress, 100)

	go m.runScan(port, threads, timeout, gen)
	return m, tea.Batch(m.listenForResults(gen), m.listenForLogs(gen), m.listenForProgress(gen))
}

func (m *Model) runScan(port, threads, timeout int, gen int) {
	var hosts <-chan scanner.Host
	target := m.targetInput.Value()
	totalEst := 0

	if !m.checkGen(gen) { return }

	switch m.inputMode {
	case modeAddr:
		if strings.HasPrefix(target, "http") {
			domains, err := scanner.ExtractDomainsFromURL(target)
			if err != nil { m.safeLog(gen, fmt.Sprintf("URL失败: %v", err), true); return }
			hosts = scanner.ParseTargets(strings.NewReader(strings.Join(domains, "\n")), m.enableIPv6)
			totalEst = len(domains)
			m.safeLog(gen, fmt.Sprintf("URL解析到 %d 个域名", totalEst), false)
		} else if strings.Contains(target, "/") {
			totalEst = scanner.CIDRSize(target)
			hosts = scanner.ExpandAddr(target, m.enableIPv6)
			m.safeLog(gen, fmt.Sprintf("CIDR %s (共%d个IP)", target, totalEst), false)
		} else {
			hosts = scanner.ExpandAddr(target, m.enableIPv6); totalEst = 1
			m.safeLog(gen, fmt.Sprintf("扫描 %s", target), false)
		}
	case modeFile:
		f, err := os.Open(m.filePath)
		if err != nil { m.safeLog(gen, fmt.Sprintf("打开文件失败: %v", err), true); return }
		hosts = scanner.ParseTargets(f, m.enableIPv6)
		f.Close()
		totalEst = 0
		m.safeLog(gen, fmt.Sprintf("文件: %s", m.filePath), false)
	case modeURL:
		domains, err := scanner.ExtractDomainsFromURL(m.urlInput.Value())
		if err != nil { m.safeLog(gen, fmt.Sprintf("URL失败: %v", err), true); return }
		hosts = scanner.ParseTargets(strings.NewReader(strings.Join(domains, "\n")), m.enableIPv6)
		totalEst = len(domains)
		m.safeLog(gen, fmt.Sprintf("URL解析到 %d 个域名", totalEst), false)
	}

	maxTargets := 0
	fmt.Sscanf(m.maxTargetInput.Value(), "%d", &maxTargets)
	if maxTargets > 0 && (totalEst == 0 || totalEst > maxTargets) { totalEst = maxTargets }
	cfg := scanner.ScannerConfig{
		Port: port, Threads: threads, Timeout: timeout,
		MaxTargets: maxTargets, EnableIPv6: m.enableIPv6, Verbose: m.verbose,
	}
	s := scanner.NewScanner(cfg); defer s.Close()

	count := 0
	resultCh := make(chan scanner.ScanResult, 1000)
	done := make(chan struct{})
	go s.Scan(hosts, resultCh, func() { close(done) })

loop:
	for {
		select {
		case <-m.cancelScan:
			return
		default:
		}

		select {
		case result, ok := <-resultCh:
			if !ok { break loop }
			count++
			item := ScanResultItem{ScanResult: result}
			m.safeResult(gen, item)
			if result.Feasible {
				m.safeLog(gen, fmt.Sprintf("✓ %s tls=%s h2=%s", result.IP, result.TLSVersion, result.CertDomain), false)
			} else if m.verbose {
				m.safeLog(gen, fmt.Sprintf("✗ %s 不可用", result.IP), true)
			}
			total := totalEst
			if total <= 0 { total = count + 1 }
			m.safeProgress(gen, count, total)
		case <-m.cancelScan:
			return
		}
	}
	m.safeLog(gen, fmt.Sprintf("TLS扫描完成: %d目标 %d可行", count, m.feasibleCount), false)
	m.safeDone(gen)
}

func (m *Model) verifyDomainItem(item ScanResultItem) {
	timeout := 10; fmt.Sscanf(m.timeoutInput.Value(), "%d", &timeout)
	c := checker.NewChecker(checker.CheckConfig{Timeout: timeout})
	r := c.Check(item.CertDomain)
	item.checkResult = r
	select {
	case m.verifyResultsCh <- item:
	default:
	}
	select {
	case m.logCh <- LogEntry{text: fmt.Sprintf("✓ %s %s", item.CertDomain, r.Recommendation), isSuccess: true}:
	default:
	}
}

func (m *Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selectedIdx > 0 { m.selectedIdx-- }
	case key.Matches(msg, m.keys.Down):
		if m.selectedIdx < len(m.filtered)-1 { m.selectedIdx++ }
	case key.Matches(msg, m.keys.Enter):
		if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
			m.detailItem = &m.filtered[m.selectedIdx]; m.view = detailView
		}
	case key.Matches(msg, m.keys.Export):
		m.exportCSV(false)
	case msg.String() == "E":
		m.exportCSV(true)
	case key.Matches(msg, m.keys.HTML):
		m.exportHTML()
	case key.Matches(msg, m.keys.SortScore):
		m.sortBy = sortByScore; m.applySort()
	case key.Matches(msg, m.keys.SortLatency):
		m.sortBy = sortByLatency; m.applySort()
	case key.Matches(msg, m.keys.ToggleFeasible):
		m.showUnfeasible = !m.showUnfeasible; m.applySort()
	case key.Matches(msg, m.keys.FilterStars):
		s := 0
		switch msg.String() { case "3": s = 3; case "4": s = 4; case "5": s = 5 }
		if m.minStars == s { m.minStars = 0 } else { m.minStars = s }
		m.applySort()
	case msg.String() == "0":
		m.minStars = 0; m.applySort()
	case key.Matches(msg, m.keys.Favorite):
		if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
			item := m.filtered[m.selectedIdx]
			key := item.IP + ":" + item.Origin
			if m.favorites[key] { delete(m.favorites, key) } else { m.favorites[key] = true }
			m.applySort()
		}
	case key.Matches(msg, m.keys.Dedupe):
		m.dedupe = !m.dedupe; m.applySort()
	case key.Matches(msg, m.keys.Search):
		m.searching = true; return m, nil
	}
	if m.selectedIdx >= len(m.filtered) { m.selectedIdx = len(m.filtered) - 1 }
	return m, nil
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.view = resultsView
	case key.Matches(msg, m.keys.Config):
		if m.detailItem != nil && m.detailItem.checkResult != nil {
			cfg := GenerateRealityConfig(m.detailItem.checkResult.CertDomain)
			m.logCh <- LogEntry{text: "--- Reality 配置 ---", isSuccess: true}
			for _, line := range strings.Split(cfg, "\n") {
				m.logCh <- LogEntry{text: line, isSuccess: true}
			}
			m.logCh <- LogEntry{text: "--- 已复制到日志 ---", isSuccess: true}
		}
	case key.Matches(msg, m.keys.Favorite):
		if m.detailItem != nil {
			item := m.detailItem
			key := item.IP + ":" + item.Origin
			if m.favorites[key] { delete(m.favorites, key) } else { m.favorites[key] = true }
			m.logCh <- LogEntry{text: fmt.Sprintf("✓ 收藏 %s", item.Origin), isSuccess: true}
		}
	}
	return m, nil
}

func (m *Model) importCSV() (tea.Model, tea.Cmd) {
	fname := m.filePath
	if fname == "" { m.logCh <- LogEntry{text: "请先输入文件路径", isErr: true}; return m, nil }
	f, err := os.Open(fname)
	if err != nil { m.logCh <- LogEntry{text: fmt.Sprintf("打开失败: %v", err), isErr: true}; return m, nil }
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil { m.logCh <- LogEntry{text: fmt.Sprintf("读取CSV失败: %v", err), isErr: true}; return m, nil }
	if len(records) < 2 { m.logCh <- LogEntry{text: "CSV为空", isErr: true}; return m, nil }

	m.scanResults = make([]ScanResultItem, 0); m.feasibleCount = 0
	for i, row := range records {
		if i == 0 || len(row) < 3 { continue }
		item := ScanResultItem{}
		item.IP = row[0]; item.Origin = row[1]; item.CertDomain = row[2]
		item.Feasible = true
		if len(row) > 3 { item.TLSVersion = row[3] }
		m.scanResults = append(m.scanResults, item); m.feasibleCount++
	}
	m.logCh <- LogEntry{text: fmt.Sprintf("CSV导入 %d 个域名，开始验证", m.feasibleCount), isSuccess: true}
	m.view = verifyingView; m.phase = phaseVerifying
	m.verifyTotal = m.feasibleCount; m.verifyChecked = 0
	for _, item := range m.scanResults {
		if item.CertDomain != "" { go m.verifyDomainItem(item) }
	}
	return m, nil
}

func (m *Model) applySort() {
	m.filtered = make([]ScanResultItem, 0, len(m.scanResults))
	seen := make(map[string]bool)

	for _, r := range m.scanResults {
		if !m.showUnfeasible && !r.Feasible { continue }
		if m.showFavOnly && !m.favorites[r.IP+":"+r.Origin] { continue }
		if m.minStars > 0 && r.checkResult != nil {
			minScore := map[int]int{3: 5, 4: 7, 5: 9}[m.minStars]
			if r.checkResult.Score < minScore { continue }
		}
		if m.searching || m.searchInput.Value() != "" {
			q := strings.ToLower(m.searchInput.Value())
			if q != "" && !strings.Contains(strings.ToLower(r.Origin), q) &&
				!strings.Contains(strings.ToLower(r.CertDomain), q) &&
				!strings.Contains(r.IP, q) {
				continue
			}
		}
		if m.dedupe && r.checkResult != nil {
			k := r.checkResult.CertDomain
			if k != "" {
				if seen[k] { continue }
				seen[k] = true
			}
		}
		m.filtered = append(m.filtered, r)
	}

	switch m.sortBy {
	case sortByScore:
		sort.Slice(m.filtered, func(i, j int) bool {
			si, sj := 0, 0
			if m.filtered[i].checkResult != nil { si = m.filtered[i].checkResult.Score }
			if m.filtered[j].checkResult != nil { sj = m.filtered[j].checkResult.Score }
			if si != sj { return si > sj }
			return m.filtered[i].HandshakeTime < m.filtered[j].HandshakeTime
		})
	case sortByLatency:
		sort.Slice(m.filtered, func(i, j int) bool { return m.filtered[i].HandshakeTime < m.filtered[j].HandshakeTime })
	case sortByDomain:
		sort.Slice(m.filtered, func(i, j int) bool { return m.filtered[i].Origin < m.filtered[j].Origin })
	}
}

func (m *Model) exportCSV(favOnly bool) {
	fname := fmt.Sprintf("reality-results-%s.csv", time.Now().Format("20060102-150405"))
	f, err := os.Create(fname)
	if err != nil { m.logCh <- LogEntry{text: fmt.Sprintf("导出失败: %v", err), isErr: true}; return }
	defer f.Close()
	w := csv.NewWriter(f); defer w.Flush()
	w.Write([]string{"星级", "评分", "IP", "域名", "TLS", "ALPN", "证书域名", "证书颁发者", "CDN等级", "CDN提供商", "被墙", "地区", "延迟", "推荐度", "收藏"})

	for _, item := range m.filtered {
		if favOnly && !m.favorites[item.IP+":"+item.Origin] { continue }
		stars, score, rec, cdn, cdnProv, blocked := "", "", "", "-", "", ""
		if item.checkResult != nil {
			stars = item.checkResult.ScoreStars(); score = fmt.Sprint(item.checkResult.Score)
			rec = item.checkResult.Recommendation; cdn = item.checkResult.CDNConfidence
			cdnProv = item.checkResult.CDNProvider
			if item.checkResult.IsBlocked { blocked = "是" } else { blocked = "否" }
		}
		fav := ""
		if m.favorites[item.IP+":"+item.Origin] { fav = "★" }
		w.Write([]string{stars, score, item.IP, item.Origin, item.TLSVersion, item.ALPN,
			item.CertDomain, item.CertIssuer, cdn, cdnProv, blocked, item.GeoCode,
			fmt.Sprintf("%v", item.HandshakeTime), rec, fav})
	}
	m.logCh <- LogEntry{text: fmt.Sprintf("已导出 %s", fname), isSuccess: true}
}

func (m *Model) exportHTML() {
	fname := fmt.Sprintf("reality-report-%s.html", time.Now().Format("20060102-150405"))
	if err := ExportHTMLReport(m.filtered, fname); err != nil {
		m.logCh <- LogEntry{text: fmt.Sprintf("HTML导出失败: %v", err), isErr: true}
		return
	}
	m.logCh <- LogEntry{text: fmt.Sprintf("HTML报告已导出: %s", fname), isSuccess: true}
}

func (m *Model) checkGen(gen int) bool {
	return gen == m.scanGen
}

func (m *Model) safeLog(gen int, text string, isErr bool) {
	if !m.checkGen(gen) { return }
	select {
	case m.logCh <- LogEntry{text: text, isErr: isErr, isSuccess: !isErr}:
	default:
	}
}

func (m *Model) safeResult(gen int, item ScanResultItem) {
	if !m.checkGen(gen) { return }
	select {
	case m.scanResultsCh <- item:
	default:
	}
}

func (m *Model) safeProgress(gen int, scanned, total int) {
	if !m.checkGen(gen) { return }
	select {
	case m.progressCh <- ScanProgress{scanned, total}:
	default:
	}
}

func (m *Model) safeDone(gen int) {
	if !m.checkGen(gen) { return }
	select {
	case m.scanDoneCh <- struct{}{}:
	default:
	}
}

func (m *Model) listenForResults(gen int) tea.Cmd {
	return func() tea.Msg {
		for {
			if !m.checkGen(gen) { return nil }
			select {
			case r := <-m.scanResultsCh: return scanResultMsg(r)
			case r := <-m.verifyResultsCh: return verifyResultMsg(r)
			case <-m.scanDoneCh: return scanDoneMsg{}
			case <-m.cancelScan: return nil
			}
		}
	}
}

func (m *Model) listenForLogs(gen int) tea.Cmd {
	return func() tea.Msg {
		if !m.checkGen(gen) { return nil }
		select {
		case entry := <-m.logCh: return logMsg(entry)
		case <-m.cancelScan: return nil
		}
	}
}

func (m *Model) listenForProgress(gen int) tea.Cmd {
	return func() tea.Msg {
		if !m.checkGen(gen) { return nil }
		select {
		case p := <-m.progressCh: return scanProgressMsg(p)
		case <-m.cancelScan: return nil
		}
	}
}

func (m *Model) eta() string {
	if m.scanScanned == 0 || m.scanTotal == 0 { return "" }
	elapsed := time.Since(m.startTime)
	rate := float64(m.scanScanned) / elapsed.Seconds()
	if rate <= 0 { return "" }
	remaining := float64(m.scanTotal-m.scanScanned) / rate
	d := time.Duration(remaining) * time.Second
	d = d.Round(time.Second)
	if d < 0 { return "" }
	h := d / time.Hour; d -= h * time.Hour
	mi := d / time.Minute; d -= mi * time.Minute
	s := d / time.Second
	if h > 0 { return fmt.Sprintf("ETA %dh%dm%ds", h, mi, s) }
	if mi > 0 { return fmt.Sprintf("ETA %dm%ds", mi, s) }
	return fmt.Sprintf("ETA %ds", s)
}
