package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	statusPatched = "patched"
	statusAlready = "already"
)

var entryRE = regexp.MustCompile(`common\.BytesToAddress\(\[\]byte\{(0x[0-9a-fA-F]+)\}\)\s*:\s*&([A-Za-z0-9_]+)\{`)

// applyContractsSource inserts EIP-2537 entries into PrecompiledContractsCancun only.
// It does not change fork selection, Prague, or Osaka.
func applyContractsSource(src string) (string, string, error) {
	blockStart, openBrace, closeBrace, err := cancunLiteral(src)
	if err != nil {
		return "", "", err
	}
	body := src[openBrace+1 : closeBrace]
	found, err := parseEntries(body)
	if err != nil {
		return "", "", err
	}
	if err := requireUpstreamTypes(src); err != nil {
		return "", "", err
	}

	missing := make([]blsBinding, 0, len(cancunBLS))
	for _, want := range cancunBLS {
		got, ok := found[want.addr]
		if !ok {
			missing = append(missing, want)
			continue
		}
		if got != want.typ {
			return "", "", fmt.Errorf("Cancun address 0x%x is bound to %s, want %s; refusing to overwrite", want.addr, got, want.typ)
		}
	}

	out := src
	changed := false
	if !strings.Contains(out[:blockStart], "CoNET build:") {
		out = out[:blockStart] + "// " + conetBuildNote + "\n" + out[blockStart:]
		changed = true
		blockStart, openBrace, closeBrace, err = cancunLiteral(out)
		if err != nil {
			return "", "", err
		}
		body = out[openBrace+1 : closeBrace]
	}

	if len(missing) > 0 {
		body = ensureTrailingComma(body)
		out = out[:openBrace+1] + body + out[closeBrace:]
		closeBrace = openBrace + 1 + len(body)
		var b strings.Builder
		if !strings.Contains(body, conetEntryNote) {
			b.WriteString("\n\t// ")
			b.WriteString(conetEntryNote)
		}
		for _, item := range missing {
			fmt.Fprintf(&b, "\n\tcommon.BytesToAddress([]byte{0x%02x}): &%s{},", item.addr, item.typ)
		}
		b.WriteString("\n")
		out = out[:closeBrace] + b.String() + out[closeBrace:]
		changed = true
	}

	if !changed {
		if err := verifyPatched(out); err != nil {
			return "", "", err
		}
		return out, statusAlready, nil
	}
	if err := verifyPatched(out); err != nil {
		return "", "", err
	}
	if err := forkSwitchUntouched(src, out); err != nil {
		return "", "", err
	}
	return out, statusPatched, nil
}

func verifyPatched(src string) error {
	_, openBrace, closeBrace, err := cancunLiteral(src)
	if err != nil {
		return err
	}
	found, err := parseEntries(src[openBrace+1 : closeBrace])
	if err != nil {
		return err
	}
	for _, want := range cancunBLS {
		got, ok := found[want.addr]
		if !ok || got != want.typ {
			return fmt.Errorf("Cancun map is missing 0x%x %s after patch", want.addr, want.typ)
		}
	}
	if !forkCaseReturns(src, "IsCancun", "PrecompiledContractsCancun") {
		return fmt.Errorf("Cancun fork switch no longer returns PrecompiledContractsCancun")
	}
	if !forkCaseReturns(src, "IsPrague", "PrecompiledContractsPrague") {
		return fmt.Errorf("Prague fork switch no longer returns PrecompiledContractsPrague")
	}
	return nil
}

var forkReturnRE = regexp.MustCompile(`(?m)^[ \t]*case rules\.(Is[A-Za-z0-9]+):[ \t]*\r?\n[ \t]*return &?([A-Za-z0-9_]+)\s*$`)

func forkCaseReturns(src, rule, name string) bool {
	for _, m := range forkReturnRE.FindAllStringSubmatch(src, -1) {
		if m[1] == rule && m[2] == name {
			return true
		}
	}
	return false
}

func forkSwitchUntouched(before, after string) error {
	fn := "func activePrecompiledContracts"
	b := strings.Index(before, fn)
	a := strings.Index(after, fn)
	if b < 0 || a < 0 {
		return fmt.Errorf("activePrecompiledContracts was not found")
	}
	bEnd := strings.Index(before[b:], "\n}\n")
	aEnd := strings.Index(after[a:], "\n}\n")
	if bEnd < 0 || aEnd < 0 {
		return fmt.Errorf("activePrecompiledContracts body was not found")
	}
	if before[b:b+bEnd] != after[a:a+aEnd] {
		return fmt.Errorf("refusing to change activePrecompiledContracts")
	}
	return nil
}

func requireUpstreamTypes(src string) error {
	for _, item := range cancunBLS {
		needle := "&" + item.typ + "{}"
		if !strings.Contains(src, needle) {
			return fmt.Errorf("upstream source has no %s; refusing to invent a BLS binding", needle)
		}
	}
	return nil
}

func cancunLiteral(src string) (blockStart, openBrace, closeBrace int, err error) {
	blockStart = strings.Index(src, cancunMarker)
	if blockStart < 0 {
		return 0, 0, 0, fmt.Errorf("PrecompiledContractsCancun was not found")
	}
	rel := strings.Index(src[blockStart:], "PrecompiledContracts{")
	if rel < 0 {
		return 0, 0, 0, fmt.Errorf("Cancun composite literal was not found")
	}
	openBrace = blockStart + rel + len("PrecompiledContracts{") - 1
	depth := 0
	for i := openBrace; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return blockStart, openBrace, i, nil
			}
		}
	}
	return 0, 0, 0, fmt.Errorf("Cancun composite literal is not closed")
}

func parseEntries(body string) (map[uint64]string, error) {
	found := map[uint64]string{}
	for _, m := range entryRE.FindAllStringSubmatch(body, -1) {
		addr, err := strconv.ParseUint(m[1], 0, 64)
		if err != nil {
			return nil, fmt.Errorf("bad precompile address %s", m[1])
		}
		if prev, ok := found[addr]; ok && prev != m[2] {
			return nil, fmt.Errorf("Cancun address 0x%x is bound twice (%s and %s)", addr, prev, m[2])
		}
		found[addr] = m[2]
	}
	return found, nil
}

func ensureTrailingComma(body string) string {
	trimmed := strings.TrimRight(body, " \t\r\n")
	if trimmed == "" || strings.HasSuffix(trimmed, ",") {
		return body
	}
	tail := body[len(trimmed):]
	return trimmed + "," + tail
}
