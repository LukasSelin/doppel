package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// funcPatterns maps language to a regex that matches the opening line of a function declaration.
var funcPatterns = map[string]*regexp.Regexp{
	"python":     regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(`),
	"javascript": regexp.MustCompile(`^(\s*)(?:async\s+)?(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>|\w+\s*=>))`),
	"typescript": regexp.MustCompile(`^(\s*)(?:async\s+)?(?:function\s+(\w+)|(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:function|\([^)]*\)\s*=>|\w+\s*=>))`),
	"java":       regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|abstract|synchronized|native)\s+)*[\w<>\[\]]+\s+(\w+)\s*\([^)]*\)\s*(?:throws\s+\w+(?:\s*,\s*\w+)*)?\s*\{`),
	"rust":       regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)\s*[<(]`),
	"csharp":     regexp.MustCompile(`^\s*(?:(?:public|private|protected|internal|static|virtual|override|abstract|async)\s+)*[\w<>\[\]?]+\s+(\w+)\s*\(`),
	"cpp":        regexp.MustCompile(`^\s*(?:(?:static|inline|virtual|explicit|constexpr|friend)\s+)*[\w:*&<>]+\s+(\w+)\s*\(`),
	"c":          regexp.MustCompile(`^\s*(?:static\s+)?[\w*]+\s+(\w+)\s*\(`),
	"ruby":       regexp.MustCompile(`^\s*def\s+(\w+)`),
}

// indentDepth returns the number of leading spaces/tabs on a line.
func indentDepth(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func parseGeneric(path, lang string) ([]CodeUnit, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pattern, ok := funcPatterns[lang]
	if !ok {
		return nil, nil
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line length
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		// File has lines too long even for 1MB (e.g. minified bundles) — skip it
		return nil, nil
	}

	var units []CodeUnit
	for i, line := range lines {
		m := pattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		name := extractName(m)
		if name == "" {
			continue
		}

		body := extractBlock(lines, i, lang)
		units = append(units, CodeUnit{
			Name:      name,
			File:      path,
			StartLine: i + 1,
			Body:      body,
			Language:  lang,
		})
	}
	return units, nil
}

// extractName pulls the function name from regex submatches.
func extractName(m []string) string {
	for _, s := range m[1:] {
		if s != "" && !strings.Contains(s, " ") {
			return s
		}
	}
	return ""
}

// extractBlock collects lines belonging to the function starting at startIdx.
// Uses brace counting for C-like languages, indent-based for Python/Ruby.
func extractBlock(lines []string, startIdx int, lang string) string {
	if lang == "python" || lang == "ruby" {
		return extractIndentBlock(lines, startIdx)
	}
	return extractBraceBlock(lines, startIdx)
}

func extractBraceBlock(lines []string, startIdx int) string {
	var sb strings.Builder
	depth := 0
	started := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		sb.WriteString(line)
		sb.WriteByte('\n')

		for _, ch := range line {
			if ch == '{' {
				depth++
				started = true
			} else if ch == '}' {
				depth--
			}
		}
		if started && depth == 0 {
			break
		}
		// Safety: stop after 200 lines to avoid runaway collection
		if i-startIdx > 200 {
			break
		}
	}
	return sb.String()
}

func extractIndentBlock(lines []string, startIdx int) string {
	var sb strings.Builder
	baseIndent := indentDepth(lines[startIdx])
	sb.WriteString(lines[startIdx])
	sb.WriteByte('\n')

	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			sb.WriteString(line)
			sb.WriteByte('\n')
			continue
		}
		if indentDepth(line) <= baseIndent {
			break
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
		if i-startIdx > 200 {
			break
		}
	}
	return sb.String()
}
