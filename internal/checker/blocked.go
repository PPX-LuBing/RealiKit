package checker

import (
	"strings"
)

type blockResult struct {
	isBlocked bool
	reason    string
}

type BlockDetector struct {
	gfwKeywords []string
}

func NewBlockDetector() *BlockDetector {
	return &BlockDetector{
		gfwKeywords: []string{
			"facebook", "youtube", "twitter", "x.com",
			"instagram", "telegram", "discord", "reddit",
			"pornhub", "xvideos", "xnxx",
			"google", "wikipedia",
		},
	}
}

func (d *BlockDetector) Check(domain string) *blockResult {
	r := &blockResult{}
	lower := strings.ToLower(domain)

	for _, kw := range d.gfwKeywords {
		if strings.Contains(lower, kw) {
			r.isBlocked = true
			r.reason = "匹配 GFW 特征关键词: " + kw
			return r
		}
	}

	return r
}
