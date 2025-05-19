# libXray

这是一个 [Xray-core](https://github.com/XTLS/Xray-core) 的包装器，用于改善客户端的开发体验。

# 注意

1. 本仓库维护人员很少。如果你不是报告 bug 或发起 PR，你的问题将被忽略。
2. 本仓库不保证 API 稳定，你需要自行适配。
3. 本仓库仅与 Xray-core 最新的发布版本保持兼容。

# 功能

## build

编译脚本。建议始终使用该脚本编译 libXray。我们不解答使用其他编译方式引起的问题。

### 使用方式

```shell
python3 build/main.py android
python3 build/main.py apple gomobile
python3 build/main.py apple go
python3 build/main.py linux
python3 build/main.py windows
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

注意：产物 `LibXray.xcframework` 不包含 **module.modulemap**。当使用 swift 时，你需要创建一个桥接文件。

### Linux

依赖 clang 和 clang++ 。

### Windows

依赖 [LLVM MinGW](https://github.com/mstorsjo/llvm-mingw)，你可使用 winget 安装。

```shell
winget install MartinStorsjo.LLVM-MinGW.UCRT
```

## controller

用于解决 Android 上 socket protect 问题。

## dns

用于解决 Android，Linux，Windows 的服务器地址解析问题。若不处理，该 DNS 流量将被重新发送至 tun 设备，导致无法发起连接。

## geo

### count

读取 geo 文件，并对分类和规则进行计数。

### read

读取 Xray Json 配置，提取使用到的 geo 文件名。

### thin

读取 Xray Json 配置，剪切使用到的 geo 文件。

## main

下载 geosite.dat 和 geoip.dat，并进行计数。

## memory

仅在 iOS 下执行，每秒发起一次 gc。可缓解 iOS 上内存压力。

## nodep

### file

写入数据到文件。

### measure

对 Xray 配置进行测速。

### model

包装接口的响应体。

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

### stats

统计。

参考如下配置：

```json
{
  "metrics" : {
    "tag" : "metrics",
    "listen": "[::1]:49227",
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

注意：

1. 当进行测试延迟或验证配置时，确保 `metrics` 为 `null`。
2. 当打开统计时，Xray-core 实例需要在 **子进程** 运行。

### validation

验证 Xray 配置。

### xray

启动和停止 Xray 实例。


## nodep_wrapper

导出 nodep 。

### xray_wrapper

导出 xray 。

# 致谢

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)

# License

本仓库基于 MIT License 。
