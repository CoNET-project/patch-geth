package main

import (
	"strings"
	"testing"
)

const fixture = `var PrecompiledContractsBerlin = PrecompiledContracts{
	common.BytesToAddress([]byte{0x5}): &bigModExp{eip2565: true, eip7823: false, eip7883: false},
	common.BytesToAddress([]byte{0x9}): &blake2F{},
}

// PrecompiledContractsCancun contains the default set of pre-compiled Ethereum
// contracts used in the Cancun release.
var PrecompiledContractsCancun = PrecompiledContracts{
	common.BytesToAddress([]byte{0x5}): &bigModExp{eip2565: true, eip7823: false, eip7883: false},
	common.BytesToAddress([]byte{0xa}): &kzgPointEvaluation{},
}

// PrecompiledContractsPrague contains the set of pre-compiled Ethereum
// contracts used in the Prague release.
var PrecompiledContractsPrague = PrecompiledContracts{
	common.BytesToAddress([]byte{0x0a}): &kzgPointEvaluation{},
	common.BytesToAddress([]byte{0x0b}): &bls12381G1Add{},
	common.BytesToAddress([]byte{0x0c}): &bls12381G1MultiExp{},
	common.BytesToAddress([]byte{0x0d}): &bls12381G2Add{},
	common.BytesToAddress([]byte{0x0e}): &bls12381G2MultiExp{},
	common.BytesToAddress([]byte{0x0f}): &bls12381Pairing{},
	common.BytesToAddress([]byte{0x10}): &bls12381MapG1{},
	common.BytesToAddress([]byte{0x11}): &bls12381MapG2{},
}

func activePrecompiledContracts(rules params.Rules) PrecompiledContracts {
	switch {
	case rules.IsPrague:
		return PrecompiledContractsPrague
	case rules.IsCancun:
		return PrecompiledContractsCancun
	default:
		return PrecompiledContractsHomestead
	}
}
`

func TestApplyInsertsOnlyCancun(t *testing.T) {
	out, status, err := applyContractsSource(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if status != statusPatched {
		t.Fatalf("status %s", status)
	}
	if strings.Contains(section(out, "PrecompiledContractsBerlin"), "bls12381") {
		t.Fatal("Berlin map was modified")
	}
	cancun := section(out, "PrecompiledContractsCancun")
	prague := section(out, "PrecompiledContractsPrague")
	for _, item := range cancunBLS {
		needle := "&" + item.typ + "{}"
		if !strings.Contains(cancun, needle) {
			t.Fatalf("Cancun missing %s\n%s", needle, cancun)
		}
	}
	if strings.Count(prague, "bls12381G1Add") != 1 {
		t.Fatalf("Prague bindings changed:\n%s", prague)
	}
	if strings.Contains(out, "pragueTime") {
		t.Fatal("patch introduced pragueTime")
	}
	again, status, err := applyContractsSource(out)
	if err != nil {
		t.Fatal(err)
	}
	if status != statusAlready || again != out {
		t.Fatalf("second apply status=%s changed=%v", status, again != out)
	}
}

func TestApplyPointerForkSwitch(t *testing.T) {
	src := strings.ReplaceAll(fixture, "return PrecompiledContracts", "return &PrecompiledContracts")
	out, status, err := applyContractsSource(src)
	if err != nil {
		t.Fatal(err)
	}
	if status != statusPatched {
		t.Fatalf("status %s", status)
	}
	if !strings.Contains(out, "return &PrecompiledContractsCancun") {
		t.Fatal("pointer return was rewritten")
	}
}

func TestApplyRefusesForeignBinding(t *testing.T) {
	src := strings.Replace(fixture,
		"common.BytesToAddress([]byte{0xa}): &kzgPointEvaluation{},\n}",
		"common.BytesToAddress([]byte{0xa}): &kzgPointEvaluation{},\n\tcommon.BytesToAddress([]byte{0x0b}): &ecrecover{},\n}",
		1)
	_, _, err := applyContractsSource(src)
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("got %v", err)
	}
}

func TestApplyRefusesMissingType(t *testing.T) {
	src := strings.ReplaceAll(fixture, "&bls12381MapG2{}", "&somethingElse{}")
	_, _, err := applyContractsSource(src)
	if err == nil || !strings.Contains(err.Error(), "bls12381MapG2") {
		t.Fatalf("got %v", err)
	}
}

func section(src, name string) string {
	start := strings.Index(src, "var "+name)
	if start < 0 {
		return ""
	}
	rest := src[start+len("var "+name):]
	next := strings.Index(rest, "\nvar ")
	if next < 0 {
		next = strings.Index(rest, "\nfunc ")
	}
	if next < 0 {
		return src[start:]
	}
	return src[start : start+len("var "+name)+next]
}
