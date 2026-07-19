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

默认情况下，编译脚本不会 clone [Xray-core](https://github.com/XTLS/Xray-core)，而是通过 Go modules 的 pseudo-version 将 Xray-core 固定到发布版本 `v26.7.11`。
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
  "apiVersion": 1,
  "method": "runXray",
  "payload": {
    "configPath": "/path/to/config.json"
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

1. 顶层 `env` 字段会被忽略且不会生效。Xray-core 运行时环境项应写入 Xray 配置根 `env` 对象。
2. `SetTunFd` 已删除。如果 fd 只能在运行时获得，请在调用 `runXray` 前把 `xray.tun.fd` 写入 Xray 配置根 `env` 对象。
3. `countGeoData` 不依赖 Xray 配置，因此通过 method payload 的 `datDir` 传入数据目录。
4. 完整的 UTF-8 编码 Invoke 请求和响应 JSON 包体限制为 16 MiB。任一方向超过限制时，Invoke 将返回 `success: false`、`data: null` 和对应的大小限制错误。

支持的 method：

```text
getFreePorts
convertShareLinksToXrayJson
convertXrayJsonToShareLinks
countGeoData
ping
testXray
runXray
runXrayFromJson
stopXray
xrayVersion
getXrayState
```

## controller

用于解决 Android 上 socket protect 问题。

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

### vmess

转换 VMessQRCode 为 Xray Json。

### xray_json

解析分享链接时用到的一些工具。

## xray

### ping

延迟测试。

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

# License

本仓库基于 MIT License 。
