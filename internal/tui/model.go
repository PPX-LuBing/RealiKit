package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"reality-scanner-rev/internal/checker"
	"reality-scanner-rev/internal/scanner"
)

type viewState int

const (
	inputView viewState = iota
	scanningView
	verifyingView
	resultsView
	detailView
	verifyView
)

type scanProfile int

const (
	profileStandard scanProfile = iota
	profileQuick
	profileDeep
	profileMass
)

var profileNames = map[scanProfile]string{
	profileStandard: "标准  (TLS + 验证, 线程10, 平衡)",
	profileQuick:    "快速  (仅TLS, 线程20, 快速筛查)",
	profileDeep:     "深度  (TLS + 验证, 线程5, 更稳)",
	profileMass:     "批量  (仅TLS, 线程50, 海量扫描)",
}

func (p scanProfile) Threads() int {
	switch p { case profileQuick: return 20; case profileDeep: return 5; case profileMass: return 50; default: return 10 }
}

func (p scanProfile) QuickMode() bool { return p == profileQuick || p == profileMass }

func (p scanProfile) Timeout() int {
	switch p { case profileDeep: return 10; case profileMass: return 3; default: return 5 }
}

type inputMode int
const (
	modeAddr inputMode = iota
	modeFile
	modeURL
)

type scanPhase int
const (
	phaseIdle scanPhase = iota
	phaseTLSScanning
	phaseVerifying
	phaseDone
)

type sortMode int
const (
	sortByScore sortMode = iota
	sortByLatency
	sortByDomain
)

type LogEntry struct {
	text      string
	isErr     bool
	isSuccess bool
}

type ScanResultItem struct {
	scanner.ScanResult
	checkResult *checker.CheckResult
}

type Model struct {
	view      viewState
	phase     scanPhase
	ready     bool
	width, height int

	profile       scanProfile
	inputMode     inputMode
	targetInput   textinput.Model
	filePath      string
	urlInput      textinput.Model
	portInput, threadInput, timeoutInput, maxTargetInput textinput.Model
	enableIPv6, verbose bool

	scanResults   []ScanResultItem
	filtered      []ScanResultItem
	selectedIdx   int
	logs          []LogEntry
	maxLogs       int
	favorites     map[string]bool
	showFavOnly   bool
	dedupe        bool

	progress      float64
	scanScanned, scanTotal, verifyTotal, verifyChecked int
	currentDomain string
	startTime     time.Time

	spinner       spinner.Model
	help          help.Model
	keys          keyMap

	scanResultsCh  chan ScanResultItem
	verifyResultsCh chan ScanResultItem
	logCh          chan LogEntry
	scanDoneCh     chan struct{}
	progressCh     chan ScanProgress
	cancelScan     chan struct{}
	cancelFn       func()

	err           error
	feasibleCount int

	sortBy            sortMode
	showUnfeasible    bool
	minStars          int
	detailItem        *ScanResultItem
	searchInput       textinput.Model
	searching         bool
	confirmQuit       bool
	scanGen           int

	// manual verify
	verifyInput textinput.Model
	verifyItem  *ScanResultItem
	verifying   bool
}

type keyMap struct {
	Quit, Start, Up, Down, Enter, Back, Profile, Export, Config, HTML,
	SortScore, SortLatency, ToggleFeasible, FilterStars, Favorite, Dedupe,
	Search, SaveCfg, FavOnly key.Binding
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "例: 1.2.3.4 或 1.2.3.0/24 或 example.com"
	ti.Width = 50; ti.Focus()

	ui := textinput.New()
	ui.Placeholder = "例: https://launchpad.net/ubuntu/+archivemirrors"
	ui.Width = 50

	pi := textinput.New(); pi.Placeholder = "443"; pi.SetValue("443"); pi.Width = 6
	thi := textinput.New(); thi.Placeholder = "10"; thi.SetValue("10"); thi.Width = 6
	toi := textinput.New(); toi.Placeholder = "5"; toi.SetValue("5"); toi.Width = 6
	mti := textinput.New(); mti.Placeholder = "0=不限"; mti.SetValue("0"); mti.Width = 8

	si := textinput.New()
	si.Placeholder = "搜索关键词..."
	si.Width = 40

	vi := textinput.New()
	vi.Placeholder = "输入域名按回车验证..."
	vi.Width = 50
	vi.Focus()

	cfg := LoadConfig()

	m := Model{
		view: inputView, phase: phaseIdle, inputMode: modeAddr,
		profile: scanProfile(cfg.Profile), maxLogs: 500,
		scanResults: make([]ScanResultItem, 0), filtered: make([]ScanResultItem, 0),
		logs: make([]LogEntry, 0), favorites: make(map[string]bool),
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		help: help.New(), sortBy: sortByScore,
		scanResultsCh: make(chan ScanResultItem, 1000),
		verifyResultsCh: make(chan ScanResultItem, 1000),
		logCh: make(chan LogEntry, 1000),
		scanDoneCh: make(chan struct{}),
		progressCh: make(chan ScanProgress, 100),
		cancelScan: make(chan struct{}),

		targetInput: ti, urlInput: ui,
		portInput: pi, threadInput: thi, timeoutInput: toi, maxTargetInput: mti,
		searchInput: si, verifyInput: vi,
	}

	if cfg.Port != "" { m.portInput.SetValue(cfg.Port) }
	if cfg.Threads != "" { m.threadInput.SetValue(cfg.Threads) }
	if cfg.Timeout != "" { m.timeoutInput.SetValue(cfg.Timeout) }
	if cfg.MaxTargets != "" { m.maxTargetInput.SetValue(cfg.MaxTargets) }
	m.enableIPv6 = cfg.EnableIPv6
	m.verbose = cfg.Verbose

	m.keys = keyMap{
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "退出")),
		Start: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "开始")),
		Up: key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "上移")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "下移")),
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "详情/确认")),
		Back: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "返回")),
		Profile: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "方案")),
		Export: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "导出CSV")),
		Config: key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "生成配置")),
		HTML: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "HTML报告")),
		SortScore: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "评分排序")),
		SortLatency: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "延迟排序")),
		ToggleFeasible: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "过滤不可用")),
		FilterStars: key.NewBinding(key.WithKeys("3", "4", "5"), key.WithHelp("3-5", "星级筛选")),
		Favorite: key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "收藏")),
		Dedupe: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "去重")),
		Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "搜索")),
		SaveCfg: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "保存参数")),
		FavOnly: key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "只看收藏")),
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, textinput.Blink)
}

type ScanProgress struct {
	scanned, total int
}

type (
	scanResultMsg   ScanResultItem
	scanDoneMsg     struct{}
	verifyResultMsg ScanResultItem
	logMsg          LogEntry
	scanProgressMsg ScanProgress
	errMsg          error
)
