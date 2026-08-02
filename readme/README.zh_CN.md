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

### Android

使用 [gomobile](https://github.com/golang/mobile) 。

### iOS && macOS

#### 1. 使用 gomobile

需要 “iOS Simulator Runtime”。

这是常规场景下的最佳选择，不会与其他 frameworks 冲突。

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
  "apiVersion": 2,
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

1. Invoke 当前只接受 `apiVersion: 2`。Xray 配置通过 `xrayJson` 传递 UTF-8 JSON 文本；libXray 不读取配置文件路径。
2. 顶层 `env` 字段会被忽略且不会生效。Xray-core 运行时环境项应写入 Xray 配置根 `env` 对象。
3. `SetTunFd` 已删除。如果 fd 只能在运行时获得，请在调用 `runXray` 前把 `xray.tun.fd` 写入 Xray 配置根 `env` 对象。
4. `countGeoData` 不依赖 Xray 配置，因此通过 method payload 的 `datDir` 传入数据目录。
5. 完整的 UTF-8 编码 Invoke 请求和响应 JSON 包体限制为 16 MiB。任一方向超过限制时，Invoke 将返回 `success: false`、`data: null` 和对应的大小限制错误。
6. `convertShareLinksToXrayJson` 会使用当前 Xray-core 配置构建器校验每个已解析的 outbound。无效 outbound 会被忽略；如果没有剩余的有效 outbound，该方法返回失败。校验不会创建或启动 Xray instance。可选的 `age.secretKey` 会在现有解析流程前于内存中解密官方 age ASCII armor；明文输入保持原有行为。
7. Xray-core 的系统拨号 DNS client 和 outbound manager 属于进程级状态。当 `runXray` 正在运行时，通过 `pingBatch`、`testXray` 或导出的 Go API 创建另一个 Xray instance，可能覆盖这些状态并影响正在运行的 instance。关闭临时 instance 不会恢复之前的状态。libXray 不对并发 instance 进行串行化、隔离或状态恢复；调用方如需同时运行多个 instance，必须将它们放在不同进程中。

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

libXray 使用 `sendThrough` 来存储节点名称。

### clash_meta

解析 Clash.Meta 配置。

### generate_share

转换 Xray Json 为 VMessAEAD/VLESS 分享协议。

### parse_share

转换 VMessAEAD/VLESS 分享协议为 Xray Json。

转换 VMessQRCode 为 Xray Json。

### age 加密订阅

`convertShareLinksToXrayJson` 接受可选的 age 原生私钥。仅支持 X25519
（`AGE-SECRET-KEY-1...`）和 ML-KEM-768 + X25519 hybrid
（`AGE-SECRET-KEY-PQ-1...`）identity。识别到 age armor 后会在内存中完成
解密，解密后明文上限为 16 MiB。

```json
{
  "apiVersion": 2,
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
  "apiVersion": 2,
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
  "apiVersion": 2,
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
    "url": "https://cp.cloudflare.com/"
  }
}
```

每次请求最多接受 5 份配置，并发测试该请求中所有已接受的配置。超过 5 份配置的
请求会在开始测试前直接失败。

批次请求本身被接受时，顶层 response 为成功；每个配置通过自己的结果表示成功或
失败。`delay` 为 `10000` 表示错误，`11000` 表示超时。结果数组与输入配置数组
长度相同且顺序一致。
通过 `streamSettings.sockopt.dialerProxy` 或 `proxySettings.tag` 引用的
outbound 依赖会被自动包含。

### testXray

直接校验传入的 Xray JSON 文本，不读取配置文件：

```json
{
  "apiVersion": 2,
  "method": "testXray",
  "payload": {
    "xrayJson": "{\"outbounds\":[...]}"
  }
}
```

### runXray

使用传入的 Xray JSON 文本启动由 libXray 管理的 Xray instance，并通过
`stopXray` 停止。`runXrayFromJson` 不再作为独立 method 存在。

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
