package tui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func ExportHTMLReport(results []ScanResultItem, fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	var stars5, stars4, stars3, stars2, stars1 int
	var totalScore, checked int
	var rows strings.Builder

	for _, r := range results {
		cr := r.checkResult
		if cr == nil {
			continue
		}
		checked++
		totalScore += cr.Score

		stars := "⭐"
		rec := "不推荐"
		switch {
		case cr.Score >= 9: stars = "⭐⭐⭐⭐⭐"; stars5++; rec = "强烈推荐"
		case cr.Score >= 7: stars = "⭐⭐⭐⭐"; stars4++; rec = "推荐"
		case cr.Score >= 5: stars = "⭐⭐⭐"; stars3++; rec = "可用"
		case cr.Score >= 3: stars = "⭐⭐"; stars2++; rec = "勉强"
		default: stars1++
		}

		cdn := cr.CDNConfidence
		if cdn == "" {
			cdn = "-"
		}
		blocked := "否"
		if cr.IsBlocked {
			blocked = "是"
		}

		rows.WriteString(fmt.Sprintf(`<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%d</td>
			<td>%s</td>
		</tr>`, stars, r.IP, cr.Domain, cr.CertDomain, cdn, blocked, cr.Score, rec))
	}

	avgScore := 0.0
	if checked > 0 {
		avgScore = float64(totalScore) / float64(checked)
	}

	css := `body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;max-width:1000px;margin:0 auto;padding:20px;background:#1a1a2e;color:#e0e0e0}
h1{color:#7c3aed;border-bottom:2px solid #7c3aed;padding-bottom:10px}
.summary{display:flex;gap:20px;margin:20px 0}
.card{background:#16213e;padding:15px 20px;border-radius:10px;flex:1;text-align:center}
.card h3{margin:0;color:#7c3aed}
.card .num{font-size:28px;font-weight:bold;margin:5px 0}
table{width:100%;border-collapse:collapse;margin:20px 0}
th,td{padding:8px 12px;text-align:left;border-bottom:1px solid #333}
th{background:#7c3aed;color:#fff}
tr:hover{background:#16213e}
.time{color:#6b7280;font-size:14px;margin-top:20px}
.fav{color:#f59e0b}`

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh">
<head><meta charset="utf-8"><title>Reality Scanner 报告</title>
<style>`+css+`</style></head><body>
<h1>Reality Scanner 扫描报告</h1>
<div class="summary">
<div class="card"><h3>总结果</h3><div class="num">%d</div></div>
<div class="card"><h3>平均评分</h3><div class="num">%.1f</div></div>
<div class="card"><h3>⭐5</h3><div class="num">%d</div></div>
<div class="card"><h3>⭐4</h3><div class="num">%d</div></div>
</div>
<div style="display:flex;justify-content:space-around;font-size:18px;padding:10px 0">
<span>⭐⭐⭐⭐⭐ %d</span><span>⭐⭐⭐⭐ %d</span><span>⭐⭐⭐ %d</span><span>⭐⭐ %d</span><span>⭐ %d</span>
</div>
<h2>推荐列表</h2>
<table><thead><tr>
<th>星级</th><th>IP</th><th>域名</th><th>证书</th><th>CDN</th><th>被墙</th><th>评分</th><th>推荐</th>
</tr></thead><tbody>%s</tbody></table>
<div class="time">生成时间: %s</div>
</body></html>`, checked, avgScore, stars5, stars4,
		stars5, stars4, stars3, stars2, stars1,
		rows.String(), time.Now().Format("2006-01-02 15:04:05"))

	_, err = f.WriteString(html)
	return err
}


