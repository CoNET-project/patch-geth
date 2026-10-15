# patch-geth

给 CoNET 使用的 Geth 源码打上 EIP-2537。Cancun 规则下的预编译表会多出 `0x0b`–`0x11`，Prague 不打开。

仓库：<https://github.com/CoNET-project/patch-geth>

## 它做什么

上游 Geth 已经实现了 BLS12-381，但只挂在 Prague 的预编译表上。CoNET 仍使用 Cancun。对 `0x0b` 的空调用因此是空账户的空成功，不能当作配对证明。

本程序只改 `core/vm/contracts.go` 里的 `PrecompiledContractsCancun`：

| 地址 | 实现（沿用该版本 Geth 里已有的类型） |
|---|---|
| `0x0b` | `bls12381G1Add` |
| `0x0c` | `bls12381G1MultiExp` |
| `0x0d` | `bls12381G2Add` |
| `0x0e` | `bls12381G2MultiExp` |
| `0x0f` | `bls12381Pairing` |
| `0x10` | `bls12381MapG1` |
| `0x11` | `bls12381MapG2` |

`init()` 会按这张表重建 `PrecompiledAddressesCancun`。`activePrecompiledContracts` 保持原样：Cancun 仍返回 Cancun 表，Prague 仍返回 Prague 表。程序不排 Prague，也不改 7702、6110、7002、7251。

Geth 升级后，对**新的源码树**再执行一次 `apply`。表结构还在、类型名还在时，程序自己把这七项补进 Cancun 表。不必为每个版本手改 unified diff。打不上就失败退出，继续用上一份已经验收的二进制。

## 它不做什么

预编译在 Geth 进程内部查表。官方二进制没有外部补丁入口。下面这些不能代替本程序：

- 在节点旁边再跑一个进程，或在 RPC 前面做代理
- 对静态链接的 Geth 使用 `LD_PRELOAD`
- 升级后再改 ELF、datadir 或 genesis
- 设置 `pragueTime` 来“顺便”打开 BLS

节点上安装的必须是用打过补丁的源码重新编译的 Geth。换成上游二进制后，`0x0b` 会回到空成功。

## 用法

```bash
git clone --depth 1 --branch v1.17.5 https://github.com/ethereum/go-ethereum
git clone https://github.com/CoNET-project/patch-geth
cd patch-geth
go run . apply --src ../go-ethereum
go run . check --src ../go-ethereum
```

`apply` 可重复执行。已经打过时打印 `already`，不会再改文件。`check` 在七个地址都以原类型出现在 Cancun 表之前失败。

也可以直接指向文件：

```bash
go run . apply --file ../go-ethereum/core/vm/contracts.go
```

然后在该 Geth 树里按原有方式编译。CoNET 执行层使用 Linux amd64 静态二进制时：

```bash
cd ../go-ethereum
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o geth ./cmd/geth
```

编译完成后，用一份会调用 `0x0b` 的探针确认空调用不再返回空成功。空返回仍然不能当成配对成功。

## 升级上游时

1. 取出新的 Geth tag，使用干净的源码树。
2. 运行 `patch-geth apply --src <那棵树>`。
3. 程序报错（找不到 Cancun 表、找不到某个 `bls12381*` 类型、或某个地址已经绑到别的实现）时停止。不要手改去凑选择器，也不要改用上游未打补丁的二进制。
4. `check` 通过后再编译，并在测试链上确认 `0x0b` 的行为。
5. 各执行层安装这一份二进制。不要写 `pragueTime`、`osakaTime` 或其它 Prague 时间戳。

替换正在出块的执行层二进制需要重启 Geth。重启须另行得到明确授权；本程序不连接节点，也不重启进程。

## 失败时的退出码

| 情况 | 退出码 |
|---|---|
| 缺少 `--src` 或 `--file`，或命令名不对 | 2 |
| 源码无法安全打补丁，或 `check` 发现尚未打上 | 1 |
| `apply` 写出补丁，或确认已经打过；`check` 通过 | 0 |
