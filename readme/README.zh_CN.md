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

默认情况下，编译脚本不会 clone [Xray-core](https://github.com/XTLS/Xray-core)，而是通过 Go modules 的 pseudo-version 将 Xray-core 固定到发布版本 `v26.9.9`。
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

构建成功或失败后都会恢复 `go.mod` 和 `go.sum`。gomobile 默认解析 `latest`，
也可通过环境变量 `LIBXRAY_GOMOBILE_VERSION` 指定 Go 模块版本；`gomobile` 与
`gobind` 使用同一个解析版本。

Linux 和 Windows 构建还会生成 `bin/xray` 或 `bin/xray.exe`。该会话 Core
会保护 Go DNS 查询不被 VPN 路由重新捕获，并且只接受以下命令：

```shell
xray run -dns <IP:port> -interface <网卡名> -config <xray.json> [-error-file <路径>]
```

`-dns`、`-interface`、`-config` 均必须提供。`-dns` 必须是 IP endpoint，`-config` 直接指向 Xray
JSON 配置。

可选的 `-error-file` 将命令失败时打印到 stderr 的原始错误同时写入 UTF-8 文件，
随后退出。运行前清空文件，成功时保持为空；父目录必须已经存在。通过提权启动 Core
的调用方应先以自身账号创建文件，以保留读取权限。该文件仅用于返回错误，不是
Xray 的 access/error 日志配置，也不增加额外的预校验。使用此参数的 App 必须同时
打包支持该参数的新版桌面 Core。

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

1. Invoke 只接受 `apiVersion: 3`，API 版本固定为 3。合同变更在该版本内同步消费方与接入文档。Xray 配置通过 `xrayJson` 传递 UTF-8 JSON 文本；libXray 不读取配置文件路径。
2. 顶层 `env` 字段会被忽略且不会生效。Xray-core 运行时环境项应写入 Xray 配置根 `env` 对象。
3. `SetTunFd` 已删除。如果 fd 只能在运行时获得，请在调用 `runXray` 前把 `xray.tun.fd` 写入 Xray 配置根 `env` 对象。
4. `countGeoData` 不依赖 Xray 配置，因此通过 method payload 的 `datDir` 传入数据目录。
5. 完整的 UTF-8 编码 Invoke 请求和响应 JSON 包体限制为 16 MiB。任一方向超过限制时，Invoke 将返回 `success: false`、`data: null` 和对应的大小限制错误。
6. `convertShareLinksToXrayJson` 会使用当前 Xray-core 配置构建器校验每个已解析的 outbound。无效 outbound 会被忽略；如果没有剩余的有效 outbound，该方法返回失败。校验不会创建或启动 Xray instance。Xray JSON 输入仅作为节点来源，只保留根级 `outbounds`，忽略其他根字段。响应仅包含 libXray 分享链接支持的字段，不支持的字段和生成的空字段会被省略；XHTTP `extra` 与 FinalMask mask `settings` 中的原始 JSON 保持不变。每次成功响应仅包含投影后的 `outbounds` 列表。可选的 `age.secretKey` 会在现有解析流程前于内存中解密官方 age ASCII armor；明文输入保持原有行为。
7. Xray-core 的系统拨号 DNS client 和 outbound manager 属于进程级状态。`pingBatch`、`testXray` 及对应导出的 Go 入口均取得受管理生命周期锁，在加载/构建配置前拒绝同进程已运行的 `runXray` instance。批量测速在全部 worker 和临时核心关闭后才释放锁，这些操作也彼此串行。由管理 API 之外创建的 instance 不在检测或恢复范围内；可能与它们重叠的调用仍须使用独立进程。

支持的 method：

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
generateAgeKeyPair
countGeoData
pingBatch
testXray
runXray
stopXray
xrayVersion
getXrayState
```

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

### generate_share

按照 [Xray-core 讨论 #716](https://github.com/XTLS/Xray-core/discussions/716)
将 Xray JSON 转换为 VMess AEAD / VLESS 分享链接，同时支持 SS、SOCKS 和 Trojan。
VMess 始终生成 AEAD URI，不生成旧版二维码格式。

没有对应分享格式的 outbound 会被跳过；无法生成任何分享链接时返回失败。

### parse_share

将 VMess AEAD / VLESS、SS、SOCKS、Trojan 分享链接以及旧版 `vmessQrCode`
链接解析为 Xray JSON。

保留 Xray JSON 节点输入以及 Base64 / Age 订阅包装。Clash/Mihomo 配置和
`hysteria2://` / `hy2://` URI 不受支持。
这一限制仅针对分享链接转换，不影响原生 Xray JSON 配置中的 Hysteria2。

#### 解析结果

`convertShareLinksToXrayJson` 只有一种响应结构。payload 包含 `text` 和可选的
`age`。每次转换成功均返回
`data: {"outbounds":[...]}`，不包含统计或额外的 `config` 包装。

无效单项会被跳过，不丢弃其余有效节点。列表按源顺序保留投影且可构建的
outbound，不做节点 hash 比较、去重或失败节点计数。

没有可用节点、无法识别格式、整份文档语法错误、容器错误或解密失败时，
返回 `success: false` 与 `data: null`。错误文案不含被拒绝的候选或解密明文。
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
通过 `streamSettings.sockopt.dialerProxy` 引用的 outbound 依赖会被自动包含。
已废弃的 `proxySettings` 字段由 Xray-core 拒绝。

`locationUrl` 为可选的绝对 HTTP(S) 地址。省略时不请求位置、不返回位置字段；
传入时，每个完成准备的配置先执行测速 HEAD，再执行位置 GET，两者使用同一个
强制经过当前所选 outbound 及其依赖的 client。每个请求各有一次配置的超时，
因此单项最多可能使用两倍超时时间。位置请求时间不计入 `delay`；两个结果独立：
`success`、`delay`、`error` 只表示延迟结果，位置失败不影响成功的延迟，延迟
失败后仍尝试位置 GET。

GET 成功后把响应正文原样放入 `locationJson` 字符串。JSON 解析和数据源专属字段
处理由 App 负责。数据源必须返回 HTTP 200，正文最大 64 KiB；传输或正文读取失败
改为返回 `locationError`。错误不回显 URL、凭据或响应正文。无效 outbound 保留
原有逐项失败结果，不发出这两个请求。

### testXray

加载传入的 Xray JSON 文本并构造临时 instance。payload 仅包含 `xrayJson`，成功时返回
`data: {}`：

```json
{
  "apiVersion": 3,
  "method": "testXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

Go 入口 `TestXray` 调用 `newXrayInstance`（`core.LoadConfig → core.New`），成功构造后
关闭临时 instance，再返回结果。不调用 `Start`，也不登记为受管理的运行 instance。
路由匹配器、缺失 balancer 等构造错误以及关闭错误通过正常错误响应返回。

校验不是沙箱：构建器和构造器可能读取本地 GeoData/证书、应用 `env`、替换进程级
日志/DNS 状态、创建日志文件，或初始化协议专属资源与后台任务。这些进程状态不恢复。
WireGuard 可能在构造时创建 TUN，VLESS reverse 可能安排后台任务。当前 Core 构造失败时
不返回部分 instance，因此 libXray 无法关闭该部分 instance。不调用 `Start` 时，内核
Geodata cron 不运行，但 assets 声明仍会检查本地文件。

调用方允许构造最小配置，或在验证副本中排除 App 管理的字段；libXray 不做 App 专属裁剪。
成功只说明传入配置可以完成实例构造和关闭，不覆盖监听/TUN 启动、系统权限与网络连通性。
被排除的配置不在验证范围内，调用方仍须处理实际启动失败。

### runXray

使用传入的 Xray JSON 文本启动由 libXray 管理的 Xray instance，并通过
`stopXray` 停止。`runXrayFromJson` 不再作为独立 method 存在。

### metrics

App 直接通过 Xray metrics HTTP API 读取当前 instance 的流量计数；libXray 不采样或保存流量。

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

libXray 这里只需要 `listen` 字段。直接用 HTTP 客户端查询 `/debug/vars`，不再通过 libXray 包装。

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
