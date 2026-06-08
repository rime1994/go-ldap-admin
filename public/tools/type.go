package tools

import "github.com/mozillazg/go-pinyin"

// H is a shortcut for map[string]any
type H map[string]any

func ConvertToPinYin(src string) (dst string) {
	args := pinyin.NewArgs()
	args.Fallback = func(r rune, args pinyin.Args) []string {
		return []string{string(r)}
	}

	for _, singleResult := range pinyin.Pinyin(src, args) {
		for _, result := range singleResult {
			dst = dst + result
		}
	}
	return
}

// compoundSurnames is a whitelist of common two-character Chinese surnames.
var compoundSurnames = []string{
	"欧阳", "司马", "诸葛", "上官", "夏侯", "东方", "独孤", "南宫",
	"皇甫", "尉迟", "令狐", "慕容", "宇文", "长孙", "宗政", "司徒",
	"司空", "公孙", "钟离", "濮阳", "公羊", "澹台", "轩辕", "百里",
	"端木", "赫连", "万俟", "闻人", "拓跋", "东郭", "呼延", "羊舌",
}

// ConvertToUIDShort generates a short UID: given-name initials + full surname pinyin.
// Examples: 王建国 → jgwang, 张三 → szhang, 欧阳娜娜 → nnouyang.
// Single-character names return the full pinyin. Non-Chinese chars pass through as-is.
func ConvertToUIDShort(src string) string {
	runes := []rune(src)
	if len(runes) == 0 {
		return ""
	}

	args := pinyin.NewArgs()
	args.Fallback = func(r rune, args pinyin.Args) []string {
		return []string{string(r)}
	}

	// detect compound surname
	surnameLen := 1
	if len(runes) >= 2 {
		prefix := string(runes[:2])
		for _, cs := range compoundSurnames {
			if prefix == cs {
				surnameLen = 2
				break
			}
		}
	}

	syllables := pinyin.Pinyin(src, args)

	surname := ""
	for _, syl := range syllables[:surnameLen] {
		if len(syl) > 0 {
			surname += syl[0]
		}
	}

	givenInitials := ""
	for _, syl := range syllables[surnameLen:] {
		if len(syl) > 0 && len(syl[0]) > 0 {
			givenInitials += string(syl[0][0])
		}
	}

	return givenInitials + surname
}
