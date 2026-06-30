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

默认情况下，编译脚本不会 clone [Xray-core](https://github.com/XTLS/Xray-core)，而是通过 Go modules 将 Xray-core 固定到 tag `v26.6.27`（Go 会记录为对应的 pseudo-version）。
传入可选参数 `local` 时，会通过 Go module `replace` 改用已有的本地仓库 `../Xray-core`。
Linux 和 Windows 构建只输出 libXray 动态库。需要独立 Xray 可执行文件的应用应使用 Xray-core 官方发布的二进制。

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

依赖 LLVM MinGW 。

你可使用 winget 安装 [LLVM MinGW](https://github.com/mstorsjo/llvm-mingw)。

```shell
winget install MartinStorsjo.LLVM-MinGW.UCRT
```

## API

libXray 只暴露一个结构化入口：

```go
func Invoke(requestJSON string) string
```

C 导出为：

```c
char* CGoInvoke(char* requestJSON);
```

请求是 JSON 对象：

```json
{
  "apiVersion": 1,
  "method": "runXray",
  "env": {
    "xray.location.config": "/path/to/config.json",
    "xray.location.asset": "/path/to/dat",
    "xray.location.cert": "/path/to/dat",
    "xray.tun.fd": "123"
  },
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

`env` 是可选字段，只支持 libXray 显式建模的 Xray-core 环境变量：

| JSON key | 含义 |
| --- | --- |
| `xray.location.config` | Xray 配置文件路径 |
| `xray.location.confdir` | Xray 配置目录路径 |
| `xray.location.asset` | 保存 `geosite.dat`、`geoip.dat` 和自定义 GeoData 的目录 |
| `xray.location.cert` | Xray-core 使用的证书目录 |
| `xray.buf.readv` | Xray-core readv buffer 开关 |
| `xray.buf.splice` | Xray-core splice buffer 开关 |
| `xray.vmess.padding` | VMess padding 开关 |
| `xray.cone.disabled` | Cone 行为开关 |
| `xray.json.strict` | 严格 JSON 解析开关 |
| `xray.ray.buffer.size` | Ray buffer size |
| `xray.browser.dialer` | Browser dialer 地址 |
| `xray.xudp.show` | XUDP 日志显示开关 |
| `xray.xudp.basekey` | XUDP base key |
| `xray.tun.fd` | Android、iOS、macOS Packet Tunnel 使用的 TUN 文件描述符 |

设计决定：

1. `env` 在 Go 和 Dart 侧都是固定字段 model，不是自由 map。
2. 未知 `env` key 会被忽略，不会写入进程环境变量。
3. `env` 只设置已建模的非空字段，缺失字段不会 unset。
4. libXray 不会在 method 结束后 restore 旧环境变量。调用方必须在每次依赖环境变量的请求中显式传入对应字段。这样可以避免并发调用时，一个请求恢复旧值覆盖另一个请求的新值。
5. `SetTunFd` 已删除。请在 `runXray` 请求的 `env` 对象中传入 `xray.tun.fd`。

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
