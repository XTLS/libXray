# libXray

This is an Xray wrapper focusing on improving the experience of [Xray-core](https://github.com/XTLS/Xray-core) mobile development.

# Note

1. This repository has very limited maintainers. If you're not reporting a bug, or making a PR, your question will most likely be ignored.

2. This lib does not guarantee the stability of the api, you need to adapt it by yourself.

3. If your issue is about some Platform development, like iOS or Android, it will be just closed.

# Features

## nodep

### file.go

write data to file.

### memory.go

try to control the max memory.

### port.go

get free port.

### share.go

convert v2rayN subscriptions to Xray Json.

convert VMessAEAD/VLESS sharing protocol to Xray Json.

### subscription.go

convert Xray Json to subscription links.

### vmess.go

convert VMessQRCode to Xray Json.

### xray_json.go

subset of Xray config, add name field to outbound.

support flattening outbounds.

## xray

### geo_cut.go

cut geosite.data 和 geoip.data.

### geo.go

read geosite.dat and geoip.dat, generate json file and count rules, including Attribute.

### ping.go

test the delay.

### stats.go

read the 

### uuid.go

convert custom text to uuid.

### validation.go

test Xray config.

### xray.go

start and stop Xray instance.

## lib package

### build.sh

generate xcframework and aar.

It will always use the latest Xray-core.

### controller.go

experimental Android support.

### dns.go

experimental Android support.

### nodep_wrapper.go

export nodep.

### xray_wrapper.go

export xray.

## Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)
