package main

// cancunBLS is EIP-2537 on the Cancun precompile map.
// These types already exist in upstream Geth; Prague is what normally exposes them.
var cancunBLS = []blsBinding{
	{addr: 0x0b, typ: "bls12381G1Add"},
	{addr: 0x0c, typ: "bls12381G1MultiExp"},
	{addr: 0x0d, typ: "bls12381G2Add"},
	{addr: 0x0e, typ: "bls12381G2MultiExp"},
	{addr: 0x0f, typ: "bls12381Pairing"},
	{addr: 0x10, typ: "bls12381MapG1"},
	{addr: 0x11, typ: "bls12381MapG2"},
}

type blsBinding struct {
	addr uint64
	typ  string
}

const (
	cancunMarker   = "var PrecompiledContractsCancun"
	conetBuildNote = "CoNET build: addresses 0x0b-0x11 are included here so BLS12-381 is active\n" +
		"// without scheduling Prague (7702 / 6110 / 7002 / 7251 stay off)."
	conetEntryNote = "CoNET: EIP-2537 on the Cancun ruleset. Prague stays unscheduled."
)
