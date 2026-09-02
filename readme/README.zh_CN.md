# libXray

这是一个 [Xray-core](https://github.com/XTLS/Xray-core) 的包装器，用于改善客户端的开发体验。

# 注意

1. 本仓库维护人员很少。如果你不是报告 bug 或发起 PR，你的问题将被忽略。
2. 本仓库不保证 API 稳定，你需要自行适配。
3. 本仓库仅与 Xray-core 最新的发布版本保持兼容。

# 功能

## build

编译脚本。建议始终使用该脚本编译 libXray。我们不解答使用其他编译方式引起的问题。

依赖 git 和 go。

默认情况下，编译脚本不会 clone [Xray-core](https://github.com/XTLS/Xray-core)，而是通过 Go modules 的 pseudo-version 将 Xray-core 固定到发布版本 `v26.7.28`。
传入可选参数 `local` 时，会通过 Go module `replace` 改用已有的本地仓库 `../Xray-core`。

### 使用方式

```shell
python3 build/main.py android
python3 build/main.py android local
python3 build/main.py apple gomobile
python3 build/main.py apple go
python3 build/main.py apple gomobile local
python3 build/main.py apple go local
python3 build/main.py linux
python3 build/main.py linux local
python3 build/main.py windows
python3 build/main.py windows local
```

每次构建尝试都会在恢复 `go.mod` 和 `go.sum` 前写入已忽略的
`build/build-metadata-<builder>.json`；builder 为 `android`、`apple-go`、
`apple-gomobile`、`linux` 或 `windows`。记录包含 libXray commit、受跟踪文件的
dirty 状态（包括临时模块修改）、Go 版本、实际生效的
`go list -mod=readonly -m all` 输出，以及生效模块文件的 SHA-256。
gomobile 仍默认解析 `latest`，同时记录解析版本、实际 PATH 中二进制的模块版本和
`go version -m` 输出；不使用 gomobile 的构建记录 `gomobile: null`。
设置环境变量 `LIBXRAY_GOMOBILE_VERSION` 可指定 Go 模块版本，`resolvedVersion` 仍记录实际解析结果。

这些记录是**构建输入证据，不是构建成功或产物匹配的证明**。失败构建也会执行记录；
采集失败写入 `errors` 或输出警告，不会覆盖原始构建错误。使用方必须独立确认构建
命令成功、记录属于本次构建，并另行验证产物；记录缺失或不完整不能视为输入已验证。

Linux 和 Windows 构建还会生成 `bin/xray` 或 `bin/xray.exe`。该会话 Core
会保护 Go DNS 查询不被 VPN 路由重新捕获，并且只接受以下命令：

```shell
xray run -dns <IP:port> -interface <网卡名> -config <xray.json> [-runtime <runtime.json>]
```

前三个参数都必须提供。`-dns` 必须是 IP endpoint，`-config` 直接指向 Xray
JSON 配置。可选 `-runtime` 的 JSON 对象见“托管运行统计”，不含外层 `runtime`。

> [!WARNING]
> **每个进程只能使用一个 Go runtime。** Go 不支持在同一进程中加载多个独立构建的
> Go runtime。libXray 的所有原生产物都会嵌入 Go runtime，无论它们通过 cgo 还是
> gomobile 生成。不要在同一个可执行文件或进程中同时加载 libXray 与另一个独立构建的
> Go、cgo 或 gomobile 库，否则可能在构建、链接或加载阶段失败，也可能在应用代码执行前
> 的 runtime 初始化阶段崩溃。
> 如果同一进程需要多个库中的 Go package，应将这些 package 放入同一次 Go build 或
> `gomobile bind` 并生成一个原生产物，使其共享一个 runtime。仅重新打包或合并已经独立
> 构建的 framework、archive、AAR、shared library 或 DLL 并不能解决问题。不同的操作系统
> 进程可以各自加载一个 Go runtime，因此需要分别对每个进程遵守这一限制。参见
> [Go #18976](https://github.com/golang/go/issues/18976#issuecomment-308505600)、
> [golang/go#15956](https://github.com/golang/go/issues/15956#issuecomment-373709423)
> 和 [libXray #116](https://github.com/XTLS/libXray/issues/116)。

### Android

使用 [gomobile](https://github.com/golang/mobile) 。

### iOS && macOS

#### 1. 使用 gomobile

需要 “iOS Simulator Runtime”。

这是常规场景下的最佳选择；与其他基于 Go 的库同时集成时，仍须遵守上方跨平台的
单 runtime 限制。

支持 iOS，iOSSimulator，macOS，macCatalyst。

但无法设置最低 macOS 版本，编译时会引起一些警告。而且不支持 tvOS。

#### 2. 使用 cgo

需要 “iOS Simulator Runtime” 和 “tvOS Simulator Runtime”。

支持更多编译选项，输出 c 头文件。

当你使用 ffi 进行集成时，这种方式将十分有效。如与 swift，kotlin，dart 进行集成。

支持 iOS，iOSSimulator，macOS，tvOS。

产物 `LibXray.xcframework` 包含 **module.modulemap**。当使用 Swift 时，
可通过 `LibXray` 模块导入。

### Linux

依赖 gcc 和 g++ 。

### Windows

依赖 `PATH` 中的 gcc 和 g++。

支持原生 amd64 和 arm64 构建。Release workflow 会在对应架构的 GitHub
Windows runner 上分别构建产物。

## API

libXray 只暴露一个结构化入口：

```go
func Invoke(requestJSON string) string
```

C 导出为：

```c
char* CGoInvoke(char* requestJSON);
void CGoFree(char* value);
```

`CGoInvoke` 会分配返回值。调用方必须使用 `CGoFree` 释放每个非空返回值，
不要直接使用平台分配器释放。

请求是 JSON 对象：

```json
{
  "apiVersion": 3,
  "method": "runXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

响应是 JSON 对象：

```json
{
  "success": true,
  "data": {},
  "error": ""
}
```

设计决定：

1. Invoke 当前只接受 `apiVersion: 3`。Xray 配置通过 `xrayJson` 传递 UTF-8 JSON 文本；libXray 不读取配置文件路径。
2. 顶层 `env` 字段会被忽略且不会生效。Xray-core 运行时环境项应写入 Xray 配置根 `env` 对象。
3. `SetTunFd` 已删除。如果 fd 只能在运行时获得，请在调用 `runXray` 前把 `xray.tun.fd` 写入 Xray 配置根 `env` 对象。
4. `countGeoData` 不依赖 Xray 配置，因此通过 method payload 的 `datDir` 传入数据目录。
5. 完整的 UTF-8 编码 Invoke 请求和响应 JSON 包体限制为 16 MiB。任一方向超过限制时，Invoke 将返回 `success: false`、`data: null` 和对应的大小限制错误。
6. `convertShareLinksToXrayJson` 会使用当前 Xray-core 配置构建器校验每个已解析的 outbound。无效 outbound 会被忽略；如果没有剩余的有效 outbound，该方法返回失败。校验不会创建或启动 Xray instance。Xray JSON 输入仅作为节点来源，只保留根级 `outbounds`，忽略其他根字段。响应仅包含 libXray 分享链接支持的字段，不支持的字段和生成的空字段会被省略；XHTTP `extra` 与 FinalMask mask `settings` 中的原始 JSON 保持不变。可选的 `age.secretKey` 会在现有解析流程前于内存中解密官方 age ASCII armor；明文输入保持原有行为。
7. Xray-core 的系统拨号 DNS client 和 outbound manager 属于进程级状态。`pingBatch`、`testXray`（含 `buildOnly`）、`checkRoute` 及对应导出的 Go 入口均取得受管理生命周期锁，在加载/构建配置前拒绝同进程已运行的 `runXray` instance。批量测速在全部 worker 和临时核心关闭后才释放锁，这些临时操作也彼此串行。由管理 API 之外创建的 instance 不在检测或恢复范围内；可能与它们重叠的调用仍须使用独立进程。

支持的 method：

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
generateAgeKeyPair
countGeoData
pingBatch
testXray
checkRoute
runXray
stopXray
xrayVersion
getXrayState
```

### 配置校验

`testXray` 支持 `{"xrayJson":"...","buildOnly":true}`，只加载并构建完整配置，
不创建 Xray instance。它校验包括 TUN/WireGuard 定义在内的配置结构，不创建设备、
监听、日志文件或后台连接。构建器仍可能读取本地 GeoData/证书，并将根 `env` 应用
到当前进程。Geodata assets 声明只校验 HTTPS URL 和已存在的本地文件，下载器及
cron 不会在只构建校验期间运行。

`buildOnly` 可选，默认 `false`，保留原有 `testXray` 创建并关闭 instance 的行为。
原行为虽然不调用 `Start`，但构造函数可能创建 TUN、打开日志或启动后台任务。
尚未启动的草稿应使用只构建校验；构建成功不代表运行资源可用，也不代表 instance
可以启动。调用方仍须处理真实构造和启动阶段的失败。

### 配置 URL 测试

`testXray` 还可随 `xrayJson` 提供 `url`、`timeout`（1–60 秒）和可选
`inboundTag`，返回整数毫秒 `data: {"delay": 12}`。`url` 与
`buildOnly: true` 互斥；省略 `url` 时保留原校验响应。

测试使用草稿完整的 DNS、routing 和 outbounds 发送 HTTP HEAD，不强制单个出站。
沿用路由检查的安全构造：禁用入站、日志输出和 webhook，拒绝 WireGuard/VLESS
reverse，不调用临时 instance 的 Start，也不发布为活动核心。结果不证明额外监听、
仅在启动时工作的集成或所有目标均可用；生命周期锁和受管理核心重叠拒绝仍然生效。

### 草稿路由检查

`checkRoute` 是 API version 3 的增量方法。通过 `xrayJson` 接收完整草稿，
调用当前锁定版本 Xray-core 的 Router；不启动临时 instance，也不向输入的目标
派发访问流量：

```json
{
  "apiVersion": 3,
  "method": "checkRoute",
  "payload": {
    "xrayJson": "{\"outbounds\":[{\"tag\":\"direct\",\"protocol\":\"freedom\"}]}",
    "domain": "example.com",
    "port": 443,
    "network": "tcp",
    "inboundTag": "tunIn",
    "timeout": 5000
  }
}
```

`domain`（主机名，不是 URL）和 `ip`（不带 zone 的 IPv4/IPv6）必须且只能提供
一个。`port` 为 1–65535，`network` 为 `tcp` 或 `udp`，必填的 `timeout` 为
1–60000 毫秒。`inboundTag` 可选，省略时为空，不默认假设 VPN 入站。沿用
16 MiB 的完整请求/响应包体限制。

成功响应的 `data` 始终包含全部五个字段：

```json
{"matched":false,"ruleTag":"","outboundTag":"direct","balancerTag":"","defaulted":true}
```

`matched` 和 `ruleTag` 表示首次 Router 匹配，保留原始的空名称或重名。
`defaulted` 表示首次 Router 没有匹配规则，随后使用 outbound manager 的真实
默认出站；如果是 loopback，则按其原生入站 tag / 跳过 DNS 解析的转换再次调用
Router。`outboundTag` 是最终选中的出站，`balancerTag` 是路径中最后经过的
balancer，没有则为空。因此带默认 loopback 的草稿可以得到其实际配置的
balancer，但不会假设任意 Raw JSON 的默认动作都是 `proxy`。仅在默认 loopback
之后命中的规则不会被报告为首次用户规则命中。出站不存在、loopback 循环、依赖访问流量的 loopback
sniffing、路由或节点选择失败均返回错误，不生成虚假结果。节点选择使用新建
instance，而非运行实例的健康度/历史；它不验证连通性或出口 IP。

只在内存中的检查配置移除 inbounds、禁用日志输出并移除规则 webhook，不回写
调用方草稿。WireGuard 出站会在构造时创建 TUN，因此即使不调用 `Start`，也必须
拒绝这类检查。不会启动入站监听或后台探测。DNS 解析仍可能通过草稿配置发出网络
查询。VLESS reverse 出站也会在构造时启动后台连接，因此同样拒绝。timeout 的
context 会传入核心，超时后不会返回成功；但部分核心解析器
（尤其 `localhost`）不能立即响应取消，因此不承诺严格的墙钟耗时上限。调用会等
匹配实际结束后才关闭 instance，不会留下继续使用已关闭核心的后台匹配。

`checkRoute` 拒绝与同进程受管理的 `runXray` instance 重叠，并在构建、匹配和
关闭期间持有受管理生命周期锁。`testXray`（含 `buildOnly`）和 `pingBatch` 具有相同
保护；未托管 instance 不在检测范围内，可能与其重叠时调用方仍须使用独立进程。

## controller

用于解决 Android 上 socket protect 问题。

### DNS 解析器

Android VPN 运行时可能会向 Go 解析器提供回环 DNS 地址。请在调用
`runXray` 前调用 `SetDNS`，让 Go 使用 VPN 配置指定的 DNS，并通过
`protectFd` 将 DNS socket 排除在 VPN 隧道外。DNS 必须是包含端口的 IP
地址，例如 `8.8.8.8:53` 或 `[2001:4860:4860::8888]:53`。

Xray 停止后调用 `ResetDNS`。这两个 API 仅存在于 Android 产物中，并会
修改 Go 进程级默认解析器。

```java
LibXray.setDNS(controller, "8.8.8.8:53");
LibXray.invoke(runXrayRequest);

// 稍后停止 Core 时：
LibXray.invoke(stopXrayRequest);
LibXray.resetDNS();
```

## geo

### count

读取 geo 文件，并对分类和规则进行计数。

## main

下载 geosite.dat 和 geoip.dat，并进行计数。

## memory

仅在 iOS 下执行，每秒发起一次 gc。可缓解 iOS 上内存压力。

## nodep

### file

写入数据到文件。

### measure

对 Xray 配置进行测速。

### port

获取空闲端口。

## share

libXray 使用 `tag` 存储节点名称。`sendThrough` 保留 Xray 原生语义，用于指定本地绑定地址。

### clash_meta

解析 Clash.Meta 配置。

### generate_share

转换 Xray Json 为 VMessAEAD/VLESS 分享协议。

### parse_share

转换 VMessAEAD/VLESS 分享协议为 Xray Json。

转换 VMessQRCode 为 Xray Json。

#### 可选解析数量

给 `convertShareLinksToXrayJson` 传入 `payload.includeStats: true` 时，返回
`data: {"config":{"outbounds":[...]},"usableCount":2,"failedCount":1}`。
省略或设为 `false` 时，保留原来的 `data.outbounds` 响应和转换行为。

数量只描述本次输入，不区分新增和更新。JSON 根 `outbounds` 的每个元素、YAML
`proxies` 的每个元素各算一个候选。已识别的分享链接列表中，每条 URI 形式的行
算一个候选，空行、注释和文本标题忽略。Base64 / age 包装使用内部格式的候选
数量。统计模式逐项跳过类型错误，不丢弃其余有效元素。`usableCount` 与最终投影且
可构建的 outbound 数量相同；解析失败、构建失败和投影不支持的候选均计入
`failedCount`。不做节点 hash 比较或去重。

已识别容器中没有可用节点时，返回 `success: false`，保留结构化数量和
`config: {"outbounds":[]}`。无法识别格式、整份文档语法错误、容器错误或解密
失败时返回 `data: null`，不猜测数量。错误文案不含被拒绝的候选或解密明文。
调用方不得在可用节点为零时导入或覆盖订阅。

### age 加密订阅

`convertShareLinksToXrayJson` 接受可选的 age 原生私钥。仅支持 X25519
（`AGE-SECRET-KEY-1...`）和 ML-KEM-768 + X25519 hybrid
（`AGE-SECRET-KEY-PQ-1...`）identity。识别到 age armor 后会在内存中完成
解密，解密后明文上限为 16 MiB。

```json
{
  "apiVersion": 3,
  "method": "convertShareLinksToXrayJson",
  "payload": {
    "text": "-----BEGIN AGE ENCRYPTED FILE-----\n...",
    "age": {
      "secretKey": "AGE-SECRET-KEY-1..."
    }
  }
}
```

`generateAgeKeyPair` 可生成新密钥对，`keyType` 支持 `x25519` 或
`hybrid`；省略时默认为 `x25519`。`hybrid` 对应 Mihomo 的
`age keygen-pq`，生成 `AGE-SECRET-KEY-PQ-1...` identity 和
`age1pq1...` recipient：

```json
{
  "apiVersion": 3,
  "method": "generateAgeKeyPair",
  "payload": {
    "keyType": "x25519"
  }
}
```

响应同时包含 `secretKey` 和 `publicKey`。接入 App 必须持久化该密钥对，并且
只将 `publicKey` 作为 `X-Age-Public-Key` 发送。libXray 不负责订阅 HTTP 请求、
密钥持久化或请求 Header；严禁通过 HTTP 发送私钥，也不能把解密后的订阅文本
写入磁盘。

### vmess

转换 VMessQRCode 为 Xray Json。

### xray_json

解析分享链接时用到的一些工具。

## xray

### pingBatch

在一个临时 Xray instance 内并发测试多份 outbound 配置。每个 `xrayJson` 文本只
解析 `outbounds`，其他根字段全部忽略。目标 outbound 依次按 `outboundTag`、
`proxy` tag、首个 outbound 选择。

```json
{
  "apiVersion": 3,
  "method": "pingBatch",
  "payload": {
    "configs": [
      {
        "xrayJson": "{\"outbounds\":[...]}"
      },
      {
        "xrayJson": "{\"outbounds\":[...]}",
        "outboundTag": "media"
      }
    ],
    "timeout": 5,
    "url": "https://cp.cloudflare.com/",
    "locationUrl": "https://ip-check-perf.radar.cloudflare.com/"
  }
}
```

每次请求最多接受 5 份配置，并发测试该请求中所有已接受的配置。超过 5 份配置的
请求会在开始测试前直接失败。

批次请求本身被接受时，顶层 response 为成功；每个配置通过自己的结果表示成功或
失败。`delay` 为 `10000` 表示错误，`11000` 表示超时。结果数组与输入配置数组
长度相同且顺序一致。
`delay` 始终输出，包含成功的 0 毫秒结果。
通过 `streamSettings.sockopt.dialerProxy` 或 `proxySettings.tag` 引用的
outbound 依赖会被自动包含。

`locationUrl` 为可选的绝对 HTTP(S) 地址。省略时不请求位置、不返回位置字段；
传入时，每个完成准备的配置先执行测速 HEAD，再执行位置 GET，两者使用同一个
强制经过当前所选 outbound 及其依赖的 client。每个请求各有一次配置的超时，
因此单项最多可能使用两倍超时时间。位置请求时间不计入 `delay`；两个结果独立：
`success`、`delay`、`error` 只表示延迟结果，位置失败不影响成功的延迟，延迟
失败后仍尝试位置 GET。

GET 成功后增加 `location: {"ip":"203.0.113.1","countryCode":"JP"}`。
数据源必须返回 HTTP 200、最大 64 KiB 的 JSON，字段为 Cloudflare 的
`ip_address` / `country`，或规范化的 `ip` / `countryCode`。IP 必须有效，
地区代码为两个 ASCII 字母并统一返回大写。缺失或无效的位置改为返回
`locationError`，错误不回显 URL、凭据或响应正文。无效 outbound 保留原有
逐项失败结果，不发出这两个请求。

### testXray

直接校验传入的 Xray JSON 文本，不读取配置文件：

```json
{
  "apiVersion": 3,
  "method": "testXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

### runXray

使用传入的 Xray JSON 文本启动由 libXray 管理的 Xray instance，并通过
`stopXray` 停止。`runXrayFromJson` 不再作为独立 method 存在。

### 托管运行统计

API v3 的 `runXray.payload.runtime` 为可选宿主元数据。省略时保留原生命周期，
不写运行快照。宿主传入以下对象；Desktop 的 `-runtime` 文件也直接使用此对象，
不含外层 `runtime`，原始 Xray 配置仍通过独立的 `-config` 传入。

```json
{
  "statePath": "/private/app/run/runtime.json",
  "planId": "opaque-plan-id",
  "inboundTag": "tunIn"
}
```

宿主提供已存在的私有目录和绝对 `statePath`。`planId` / `inboundTag` 非空且
各不超过 256 字节；`planId` 是不包含凭据的不透明标识。元数据独立于 Xray JSON，
用户配置不能覆盖。指定入站必须存在，并启用上下行系统统计和 stats manager。
元数据无效、已有快照损坏、归档失败或首次保存失败均拒绝启动，并关闭已构建的核心。

落盘文件仅包含本次会话的原始入站计数：

```json
{
  "version": 1,
  "session": {
    "id": "2a7e2e49b947a802d8b39af4fbc48f52",
    "planId": "opaque-plan-id",
    "startedAtMs": 1788300000000,
    "endedAtMs": 0,
    "uplink": 120,
    "downlink": 800
  },
  "available": true,
  "sampledAtMs": 1788300030000,
  "savedAtMs": 1788300030000,
  "error": ""
}
```

时间为 Unix 毫秒。每次新启动生成 32 位小写十六进制随机 session ID，即使重放相同
元数据也不复用。`endedAtMs: 0` 只表示没有保存最终停止快照，不能用来判断 VPN
仍在运行。宿主启动时先保存新快照，此后每 30 秒采样保存，`stopXray` 在关闭核心前
尽力完成最终采样保存。

采样直接读取指定入站的 `Value()`，不重置计数，不叠加节点或 outbound 计数。
重复采样不累加字节；非负计数回退时保存实际较小值，不合成差额。计数缺失或为负时，
`available: false`、`error: "counters_unavailable"`，保留上次合法的非负值。
有效入站尚无流量时为可用的 0。不维护 App 总量、重置代次、runtime HTTP 接口、
控制端口或 token。`resetRuntime` 不是 Invoke method。App 可通过已有 Xray metrics
读取实时速率；App 累计与重置策略由 App 自行管理，不属于 libXray。

新会话覆盖 `runtime.json` 前，先将已有合法快照原子归档到同级目录
`runtime-sessions/<session-id>.json`。归档保留原始计数、时间及可能未设置的结束
时间，不推测丢失流量或崩溃时间。重复启动失败使用同一个归档文件名；准备失败时，
当前文件和归档可能同时存在相同会话，消费者必须按 session ID 识别，不能按文件数
重复计入。libXray 不清理归档，也不把旧计数继承到新会话。归档目录以 0700 创建，
拒绝符号链接和非目录对象。

快照文件使用同目录 0600 临时文件，sync 后原子替换；Windows 使用
`MoveFileEx` 的替换和 write-through 标志。私有父目录/Windows ACL 由宿主管理。
保存失败保留上次完整磁盘快照供后续重试；最终保存失败向调用方报告，但仍关闭核心。
rename 后发生 I/O 错误时结果可能不确定，消费者应重新读取文件。这是参考数据，
不是计费账本：崩溃/强杀允许丢失最后成功保存后的尾部，不承诺严格 30 秒丢失上限。
下次启动只归档已有快照，不伪造最终计数。

`statePath + ".lock"` 的非阻塞操作系统文件锁保持至核心关闭，防止跨进程同时
改写当前快照和归档。宿主须使用一致的规范路径并保留锁文件。UI 只读快照，不能在
宿主持有路径期间直接写入。此能力不解决 macOS System Extension 的 root 文件访问
权限，也不能让 Windows Job 强制终止获得正常最终结算；这些平台边界仍由接入方处理。

### metrics

统计。

参考如下配置：

```json
{
  "metrics" : {
    "listen": "127.0.0.1:49227"
  },
  "policy" : {
    "system" : {
      "statsInboundDownlink" : true,
      "statsInboundUplink" : true,
      "statsOutboundDownlink" : true,
      "statsOutboundUplink" : true
    }
  },
  "stats" : {}
}
```

metrics 服务通过 HTTP 暴露 Xray 运行时计数。例如 `listen` 为
`127.0.0.1:49227` 时，读取：

```text
http://localhost:49227/debug/vars
```

注意：

1. 当进行测试延迟或验证配置时，确保 `metrics` 为 `null`。
2. libXray 这里只需要 `listen` 字段。直接用 HTTP 客户端查询 `/debug/vars`，不再通过 libXray 包装。

### validation

验证 Xray 配置。

### xray

启动和停止 Xray 实例。

# 致谢

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)

[MetaCubeX age](https://github.com/MetaCubeX/age)（BSD 3-Clause）

# License

本仓库基于 MIT License 。
