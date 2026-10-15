# patch-geth

Patches Geth source so CoNET gets EIP-2537. The Cancun precompile table gains `0x0b`–`0x11`. Prague stays off.

Repository: <https://github.com/CoNET-project/patch-geth>

## What it does

Upstream Geth already implements BLS12-381, and registers it only on the Prague precompile table. CoNET still runs under Cancun rules. An empty call to `0x0b` is therefore an empty-account success, and that result is not a pairing proof.

This program edits only `PrecompiledContractsCancun` in `core/vm/contracts.go`:

| Address | Implementation (the type already in that Geth version) |
|---|---|
| `0x0b` | `bls12381G1Add` |
| `0x0c` | `bls12381G1MultiExp` |
| `0x0d` | `bls12381G2Add` |
| `0x0e` | `bls12381G2MultiExp` |
| `0x0f` | `bls12381Pairing` |
| `0x10` | `bls12381MapG1` |
| `0x11` | `bls12381MapG2` |

`init()` rebuilds `PrecompiledAddressesCancun` from that table. `activePrecompiledContracts` stays as upstream wrote it: Cancun still returns the Cancun table, and Prague still returns the Prague table. The program does not schedule Prague, and it does not change EIP-7702, EIP-6110, EIP-7002, or EIP-7251.

After a Geth upgrade, run `apply` again on the **new** source tree. While the table and the type names are still there, the program inserts these seven entries into the Cancun table. A hand-written unified diff is not required for each release. If the patch cannot be applied safely, the program exits with a failure and the last accepted binary stays in use.

## What it does not do

Precompiles are looked up inside the Geth process. The upstream binary has no external patch hook. None of the following replaces this program:

- a sidecar process, or an RPC proxy in front of the node
- `LD_PRELOAD` against a statically linked Geth
- editing the ELF, datadir, or genesis after an upgrade
- setting `pragueTime` in order to turn BLS on

The execution client on a node must be a Geth binary rebuilt from the patched tree. Replacing it with an upstream binary makes `0x0b` an empty success again.

## Usage

```bash
git clone --depth 1 --branch v1.17.5 https://github.com/ethereum/go-ethereum
git clone https://github.com/CoNET-project/patch-geth
cd patch-geth
go run . apply --src ../go-ethereum
go run . check --src ../go-ethereum
```

`apply` is safe to run again. When the tree is already patched it prints `already` and does not rewrite the file. `check` fails until all seven addresses are present in the Cancun table with those types.

Point at the file directly when the checkout layout differs:

```bash
go run . apply --file ../go-ethereum/core/vm/contracts.go
```

Build that Geth tree with its normal commands. For a static Linux amd64 execution client:

```bash
cd ../go-ethereum
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o geth ./cmd/geth
```

After the build, probe an empty call to `0x0b` and confirm it is no longer an empty success. An empty return is still not a successful pairing.

## Upgrading upstream Geth

1. Check out the new Geth tag in a clean source tree.
2. Run `patch-geth apply --src <that tree>`.
3. Stop if the program reports an error: the Cancun table is missing, a `bls12381*` type is missing, or an address is already bound to a different implementation. Do not hand-edit a selector to force a match, and do not switch to an unpatched upstream binary.
4. Compile only after `check` passes, then confirm `0x0b` on a test chain.
5. Install that binary on each execution client. Do not set `pragueTime`, `osakaTime`, or any other Prague timestamp.

Replacing the execution binary on a node that is producing blocks requires a Geth restart. That restart needs its own explicit authorization. This program does not connect to a node and does not restart a process.

## Exit codes

| Condition | Exit code |
|---|---|
| Missing `--src` or `--file`, or an unknown command | 2 |
| The source cannot be patched safely, or `check` finds it is not patched | 1 |
| `apply` wrote the patch or confirmed it was already present; `check` passed | 0 |
