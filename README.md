# libXray

This is an Xray wrapper focusing on improving the experience of [Xray-core](https://github.com/XTLS/Xray-core) mobile development.

# Features

### build.sh

编译脚本。一键生成 xcframework 和 aar。

### clash.go

转换 Clash yaml，Clash.Meta yaml 为 Xray Json。

### file.go

文件写入。

### geo.go

读取 geosite.data，生成类别名称文件，包含 Attribute。

读取 geoip.data，生成类别名称文件。

### memory.go

强制 GC。

### ping.go

测速。

### port.go

获取空闲端口。

### share.go

转换 v2rayN 订阅为 Xray Json。

转换 VMessAEAD/VLESS 分享为 Xray Json。

### subscription.go

转换 Xray Json 为订阅文本。

### uuid.go

转换自定义文本为 uuid。

### xray_json.go

Xray 配置的子集，为出口节点添加了 Name 字段，便于 App 内进行解析。

支持 flatten outbounds 。

### xray.go

启动和停止 Xray 。

# Used By

[FoXray](https://apps.apple.com/app/foxray/id6448898396)

# Contributing

[yiguo](https://yiguo.dev) wrote the original source code. Now it belongs to the Xray Community.

## Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)
