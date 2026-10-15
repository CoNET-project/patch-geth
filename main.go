package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "apply", "check", "help", "-h", "--help":
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		return 2
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage()
		return 0
	}

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	src := fs.String("src", "", "go-ethereum checkout (reads core/vm/contracts.go)")
	file := fs.String("file", "", "contracts.go path; overrides --src")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	path, err := contractsPath(*src, *file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	next, status, err := applyContractsSource(string(raw))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	switch args[0] {
	case "check":
		if status != statusAlready {
			fmt.Fprintf(os.Stderr, "not patched: %s would change %s\n", status, path)
			return 1
		}
		fmt.Printf("check ok %s\n", path)
		return 0
	case "apply":
		if status == statusPatched {
			if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		fmt.Printf("%s %s\n", status, path)
		return 0
	default:
		usage()
		return 2
	}
}

func contractsPath(src, file string) (string, error) {
	if file != "" {
		return file, nil
	}
	if src == "" {
		return "", fmt.Errorf("pass --src <go-ethereum> or --file <contracts.go>")
	}
	return filepath.Join(src, "core", "vm", "contracts.go"), nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `patch-geth adds EIP-2537 (0x0b-0x11) to Geth's Cancun precompile table.

The BLS implementations stay the ones already in that Geth source. Prague is not scheduled.

  patch-geth apply --src <go-ethereum>
  patch-geth check --src <go-ethereum>

Apply is idempotent. A second run prints "already". Check fails until apply has succeeded.
Rebuild Geth from the patched tree. Do not schedule Prague.
`)
}
