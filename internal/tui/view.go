package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready { return "\n  加载中..." }
	if m.confirmQuit {
		return lipgloss.JoinVertical(lipgloss.Center,
			TitleStyle.Render("  Reality Scanner TUI  "),
			"",
			ErrorBoxStyle.Render("  确认退出？扫描结果将丢失！ (y/N)  "),
		)
	}
	switch m.view {
	case inputView: return m.viewInput()
	case scanningView: return m.viewScanning()
	case verifyingView: return m.viewVerifying()
	case resultsView: return m.viewResults()
	case detailView: return m.viewDetail()
	case verifyView: return m.viewVerify()
	default: return "未知视图"
	}
}

func (m Model) viewInput() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("  Reality Scanner TUI  v0.1.0  ")); b.WriteString("\n\n")

	b.WriteString(BoxStyle.Render(
		InputLabelStyle.Render("扫描方案 (m切换)") + "\n" +
			m.renderProfile(profileStandard) + m.renderProfile(profileQuick) +
			m.renderProfile(profileDeep) + m.renderProfile(profileMass),
	))
	b.WriteString("\n")

	ib := strings.Builder{}
	ib.WriteString(InputLabelStyle.Render("输入方式 (↑↓)") + "\n")
	labels := []string{"IP/CIDR/域名", "从文件 (c验证CSV)", "从 URL 抓取"}
	for i, l := range labels {
		if i == int(m.inputMode) { ib.WriteString(RadioSelectedStyle.Render("● ")) } else { ib.WriteString(RadioUnselectedStyle.Render("○ ")) }
		ib.WriteString(l); ib.WriteString("\n")
	}
	ib.WriteString("\n")
	switch m.inputMode {
	case modeAddr: ib.WriteString(m.targetInput.View())
	case modeFile:
		ib.WriteString(InputLabelStyle.Render("路径: ") + "\n" + m.targetInput.View())
		ib.WriteString("\n" + NoticeStyle.Render("按 c 验证CSV (跳过TLS扫描)"))
	case modeURL:
		ib.WriteString(m.urlInput.View())
		ib.WriteString("\n" + NoticeStyle.Render("自动提取页面域名"))
	}
	ib.WriteString("\n")
	cfg := fmt.Sprintf("端口:%-5s 线程:%-5s 超时:%-5s 上限:%-6s",
		m.portInput.View(), m.threadInput.View(), m.timeoutInput.View(), m.maxTargetInput.View())
	cfg += m.renderToggle(m.enableIPv6, "IPv6") + m.renderToggle(m.verbose, "Verbose")
	ib.WriteString(cfg)

	b.WriteString(BoxStyle.Render(ib.String())); b.WriteString("\n")
	b.WriteString(ButtonStyle.Render("  ▶ 开始 (S)  "))
	b.WriteString("\n")
	b.WriteString(StatusBarStyle.Width(m.width).Render(
		"q:退出 m:方案 ↑↓:输入 i:IPv6 v:Verbose s:开始 c:CSV t:手动验证 S:保存参数"))
	return AppStyle.Width(m.width).Height(m.height).Render(b.String())
}

func (m Model) viewScanning() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("  Reality Scanner TUI  -  TLS扫描中  ")); b.WriteString("\n\n")
	bar := m.renderProgressBar(m.progress)
	eta := m.eta()
	info := fmt.Sprintf("  %s  扫描: %d/%d  可行: %d  耗时: %s", m.spinner.View(),
		m.scanScanned, m.scanTotal, m.feasibleCount, timeSince(m.startTime))
	if eta != "" { info += "  " + eta }
	b.WriteString(BoxStyle.Render(bar + "\n" + info)); b.WriteString("\n\n")

	lb := strings.Builder{}
	start := len(m.logs) - 12
	if start < 0 { start = 0 }
	for _, e := range m.logs[start:] {
		st := LogStyle
		if e.isSuccess { st = LogSuccessStyle } else if e.isErr { st = LogErrorStyle }
		lb.WriteString(st.Render(e.text) + "\n")
	}
	b.WriteString(BoxStyle.Height(8).Render(lb.String())); b.WriteString("\n")
	b.WriteString(StatusBarStyle.Width(m.width).Render(" esc:取消  q两次退出"))
	return AppStyle.Width(m.width).Render(b.String())
}

func (m Model) viewVerifying() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("  Reality Scanner TUI  -  深度验证中  ")); b.WriteString("\n\n")
	pct := 0.0
	if m.verifyTotal > 0 { pct = float64(m.verifyChecked) / float64(m.verifyTotal) }
	bar := m.renderProgressBar(pct)
	info := fmt.Sprintf("  %s  验证: %d/%d  耗时: %s", m.spinner.View(),
		m.verifyChecked, m.verifyTotal, timeSince(m.startTime))
	b.WriteString(BoxStyle.Render(bar + "\n" + info)); b.WriteString("\n\n")

	lb := strings.Builder{}
	start := len(m.logs) - 12
	if start < 0 { start = 0 }
	for _, e := range m.logs[start:] {
		st := LogStyle
		if e.isSuccess { st = LogSuccessStyle } else if e.isErr { st = LogErrorStyle }
		lb.WriteString(st.Render(e.text) + "\n")
	}
	b.WriteString(BoxStyle.Height(8).Render(lb.String())); b.WriteString("\n")
	b.WriteString(StatusBarStyle.Width(m.width).Render(" esc:取消"))
	return AppStyle.Width(m.width).Render(b.String())
}

func (m Model) viewVerify() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("  Reality Scanner TUI  -  手动验证  ")); b.WriteString("\n\n")
	b.WriteString(BoxStyle.Render(InputLabelStyle.Render("输入域名") + "\n" + m.verifyInput.View()))
	b.WriteString("\n")

	if m.verifying {
		b.WriteString(BoxStyle.Render(fmt.Sprintf("  %s  验证中...", m.spinner.View())))
	} else if m.verifyItem != nil && m.verifyItem.checkResult != nil {
		cr := m.verifyItem.checkResult
		v := strings.Builder{}
		v.WriteString(fmt.Sprintf("%s 评分 %d/10 | %s\n", cr.ScoreStars(), cr.Score, cr.Recommendation))
		v.WriteString(fmt.Sprintf("  IP: %s  TLS: %s  H2: %v\n", cr.IP, cr.TLSVersion, cr.SupportsH2))
		v.WriteString(fmt.Sprintf("  可访问: %s 状态码:%d  CDN:%s\n", boolStr(cr.Accessible), cr.StatusCode, cr.CDNConfidence))
		v.WriteString(fmt.Sprintf("  证书:%s 剩余%d天 SNI:%s\n", cr.CertDomain, cr.CertDaysLeft, boolStr(cr.SNIMatch)))
		v.WriteString(fmt.Sprintf("  被墙:%s 热门:%s\n", boolStr(cr.IsBlocked), boolStr(cr.IsHotWebsite)))
		b.WriteString(BoxStyle.Render(v.String()))
	}
	b.WriteString("\n")
	b.WriteString(StatusBarStyle.Width(m.width).Render(" esc:返回  enter:验证"))
	return AppStyle.Width(m.width).Render(b.String())
}

func (m Model) viewResults() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("  Reality Scanner TUI  -  结果  "))
	if m.dedupe { b.WriteString(TitleStyle.Render(" [去重]")) }
	if m.showFavOnly { b.WriteString(TitleStyle.Render(" [收藏]")) }
	b.WriteString("\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(BoxStyle.Render(NoticeStyle.Render("没有匹配的结果"))); b.WriteString("\n")
		b.WriteString(StatusBarStyle.Width(m.width).Render(" q:退出 esc:返回"))
		return AppStyle.Width(m.width).Render(b.String())
	}

	scoreCounts := map[string]int{}
	var totalScore, checkedCount, cdnCount int
	for _, r := range m.filtered {
		if r.checkResult != nil {
			scoreCounts[r.checkResult.ScoreStars()]++
			totalScore += r.checkResult.Score; checkedCount++
			if r.checkResult.IsCDN { cdnCount++ }
		} else if r.Feasible { scoreCounts["待验证"]++ }
	}
	stats := fmt.Sprintf("共 %d 条 | 显示 %d", len(m.scanResults), len(m.filtered))
	for stars, cnt := range scoreCounts { stats += fmt.Sprintf(" %s:%d", stars, cnt) }
	if m.scanTotal > 0 { stats += fmt.Sprintf(" | %s", timeSince(m.startTime)) }
	b.WriteString(BoxStyle.Render(stats))
	if checkedCount > 0 {
		avg := float64(totalScore) / float64(checkedCount)
		s := fmt.Sprintf("均分:%.1f CDN:%d%%", avg, cdnCount*100/checkedCount)
		if m.minStars > 0 { s += fmt.Sprintf(" ≥%d星", m.minStars) }
		if m.searchInput.Value() != "" { s += fmt.Sprintf(" 搜索:%s", m.searchInput.Value()) }
		b.WriteString("\n" + NoticeStyle.Render(s))
	}
	b.WriteString("\n")

	columns := []table.Column{
		{Title: "", Width: 2}, {Title: "星级", Width: 7}, {Title: "IP", Width: 15},
		{Title: "域名", Width: 18}, {Title: "CDN", Width: 5}, {Title: "延迟", Width: 9}, {Title: "推荐", Width: 12},
	}
	rows := make([]table.Row, 0, len(m.filtered))
	for _, item := range m.filtered {
		fav := " "
		if m.favorites[item.IP+":"+item.Origin] { fav = "★" }
		stars, cdn, rec := "", "-", ""
		if item.checkResult != nil {
			stars = item.checkResult.ScoreStars()
			cdn = item.checkResult.CDNConfidence
			rec = item.checkResult.Recommendation
		} else if item.Feasible { stars = "⭐"; rec = "待验证" }
		rows = append(rows, table.Row{
			fav, stars, item.IP, truncate(item.Origin, 18),
			cdn, fmtDuration(item.HandshakeTime), rec,
		})
	}

	t := table.New(
		table.WithColumns(columns), table.WithRows(rows),
		table.WithFocused(true), table.WithHeight(min(len(rows)+1, m.height-16)),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(ColorBgLight).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(ColorText).Background(ColorPrimary)
	t.SetStyles(s)

	b.WriteString(ResultTableStyle.Render(t.View())); b.WriteString("\n")
	h := "↑↓移动 enter详情 space收藏 e导出 E仅收藏 wHTML报告 d去重 /搜索"
	b.WriteString(StatusBarStyle.Width(m.width).Render(h))
	return AppStyle.Width(m.width).Render(b.String())
}

func (m Model) viewDetail() string {
	if m.detailItem == nil { m.view = resultsView; return "" }
	item := m.detailItem
	var b strings.Builder
	fav := " "
	if m.favorites[item.IP+":"+item.Origin] { fav = "★" }
	b.WriteString(TitleStyle.Render(fmt.Sprintf("  %s %s  ", fav, item.Origin))); b.WriteString("\n\n")

	info := strings.Builder{}
	info.WriteString(InputLabelStyle.Render("TLS 信息") + "\n")
	info.WriteString(fmt.Sprintf("  IP:     %s\n", item.IP))
	info.WriteString(fmt.Sprintf("  TLS:    %s\n", item.TLSVersion))
	info.WriteString(fmt.Sprintf("  ALPN:   %s\n", item.ALPN))
	info.WriteString(fmt.Sprintf("  延迟:   %v\n", item.HandshakeTime))
	b.WriteString(BoxStyle.Render(info.String())); b.WriteString("\n")

	cert := strings.Builder{}
	cert.WriteString(InputLabelStyle.Render("证书信息") + "\n")
	cert.WriteString(fmt.Sprintf("  域名:   %s\n", item.CertDomain))
	cert.WriteString(fmt.Sprintf("  颁发者: %s\n", truncate(item.CertIssuer, 50)))
	cert.WriteString(fmt.Sprintf("  签名:   %s  密钥: %s\n", item.CertSignature, item.CertPubKey))
	b.WriteString(BoxStyle.Render(cert.String())); b.WriteString("\n")

	if item.checkResult != nil {
		cr := item.checkResult
		vr := strings.Builder{}
		vr.WriteString(InputLabelStyle.Render(fmt.Sprintf("验证: %s  %d/10  %s", cr.ScoreStars(), cr.Score, cr.Recommendation)))
		vr.WriteString(fmt.Sprintf("\n  可访问: %s  状态码: %d  响应: %v", boolStr(cr.Accessible), cr.StatusCode, cr.ResponseTime))
		vr.WriteString(fmt.Sprintf("\n  CDN: %s %s  热门: %s  被墙: %s", cr.CDNConfidence, cr.CDNProvider, boolStr(cr.IsHotWebsite), boolStr(cr.IsBlocked)))
		vr.WriteString(fmt.Sprintf("\n  证书: %s 剩余%d天 SNI:%s", cr.CertDomain, cr.CertDaysLeft, boolStr(cr.SNIMatch)))
		b.WriteString(BoxStyle.Render(vr.String()))
	} else {
		b.WriteString(BoxStyle.Render(NoticeStyle.Render("尚未深度验证")))
	}
	b.WriteString("\n")
	help := "esc返回 g生成配置 space收藏"
	b.WriteString(StatusBarStyle.Width(m.width).Render(help))
	return AppStyle.Width(m.width).Render(b.String())
}

func (m Model) renderProgressBar(pct float64) string {
	if pct > 1 { pct = 1 }
	w := 50; f := int(float64(w) * pct)
	return fmt.Sprintf("  %s  %3.0f%%", strings.Repeat("█", f)+strings.Repeat("░", w-f), pct*100)
}

func (m Model) renderToggle(on bool, label string) string {
	if on { return fmt.Sprintf(" [%s]%s", RadioSelectedStyle.Render("✓"), label) }
	return fmt.Sprintf(" [%s]%s", RadioUnselectedStyle.Render(" "), label)
}

func (m Model) renderProfile(p scanProfile) string {
	sel := m.profile == p
	n := profileNames[p]
	if sel { return RadioSelectedStyle.Render("● "+n) + "\n" }
	return RadioUnselectedStyle.Render("○ "+n) + "\n"
}

func timeSince(t time.Time) string {
	d := time.Since(t).Round(time.Second)
	h := d / time.Hour; d -= h * time.Hour
	mi := d / time.Minute; d -= mi * time.Minute
	s := d / time.Second
	if h > 0 { return fmt.Sprintf("%dh%dm%ds", h, mi, s) }
	if mi > 0 { return fmt.Sprintf("%dm%ds", mi, s) }
	return fmt.Sprintf("%ds", s)
}

func fmtDuration(d time.Duration) string {
	if d == 0 { return "-" }
	return d.Round(time.Millisecond).String()
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n-1] + "…"
}

func boolStr(b bool) string {
	if b { return "是" }
	return "否"
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
