# libXray

This is an Xray wrapper focusing on improving the experience of [Xray-core](https://github.com/XTLS/Xray-core) mobile development.

# Note

1. This repository has very limited maintainers. If you're not reporting a bug, or making a PR, your question will most likely be ignored.

2. This lib does not guarantee the stability of the api, you need to adapt it by yourself.

3. If your issue is about some Platform development, like iOS or Android, it will be just closed.

# Features

## nodep

### clash.go

转换 Clash yaml，Clash.Meta yaml 为 Xray Json。

### file.go

文件写入。

### measure.go

基于 http/socks5 proxy 进行延迟测试。

基于 geoip.dat 进行 geolocation。

TCPPing。

### memory.go

强制 GC。

### port.go

获取空闲端口。

### share.go

转换 v2rayN 订阅为 Xray Json。

转换 VMessAEAD/VLESS 分享为 Xray Json。

### subscription.go

转换 Xray Json 为订阅文本。

### vmess.go

转换 VMessQRCode 为 Xray Json。

### xray_json.go

Xray 配置的子集，为出口节点添加了 Name 字段，便于 App 内进行解析。

支持 flatten outbounds 。

## lib package

### build.sh

编译脚本。一键生成 xcframework 和 aar。


### controller.go

实验性的 Android 支持 。

### geo_cut.go

剪切 geosite.data 和 geoip.data 。

### geo.go

读取 geo site dat，生成类别名称文件并统计规则数量，包含 Attribute。

读取 geo ip dat，生成类别名称文件并统计规则数量。

### nodep_wrapper.go

获取空闲端口。

转换分享文本为 Xray Json。

转换 Xray Json 为分享文本。

### ping.go

测速。

### uuid.go

转换自定义文本为 uuid。

### validation.go

测试 Xray 配置文件。

### xray.go

启动和停止 Xray 。

## Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)

[SeeIP](https://seeip.org/)
