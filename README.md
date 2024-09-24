# libXray

This is an Xray wrapper focusing on improving the experience of [Xray-core](https://github.com/XTLS/Xray-core) mobile development.

# Note

1. This repository has very limited maintainers. If you're not reporting a bug, or making a PR, your question will most likely be ignored.

2. This lib does not guarantee the stability of the api, you need to adapt it by yourself.

3. If your issue is about some Platform development, like iOS or Android, it will be just closed.

# Break changes

From 3.0.0, all apis have changed to base64-encoded-string-based, including paramters and return value.

The reasons are as bellow.

1. We must be careful when using cgo. Always remember to free c-strings we pass to cgo and get from cgo. If there are many string paramters in the function, it will be a nightmare. So we just keep one string parameter and one string return value for every function.

2. The string paramter and string return value may be transfered between languages, like go -> swift/kotlin/cpp -> dart, using their ffi. Some characters may be wrong when they are encoded and decoded many times. So encoding string to ascii characters will be a better choice, and we just choose base64.

# Features

## nodep

### clash.go

parse Clash and Clash.Meta config.

### file.go

write data to file.

### generate_share.go

convert v2rayN subscriptions to Xray Json.

convert VMessAEAD/VLESS sharing protocol to Xray Json.

### measure.go

ping xray outbound.

### memory.go

try to control the max memory.

### parse_share.go

convert Xray Json to subscription links.

### port.go

get free port.

### vmess.go

convert VMessQRCode to Xray Json.

### xray_json.go

subset of Xray config, add name field to outbound.

support flattening outbounds.

## xray

### geo.go

read geosite.dat and geoip.dat, generate json file and count rules, including Attribute.

### ping.go

test the delay.

### stats.go

query inbound and outbound stats.

### uuid.go

convert custom text to uuid.

### validation.go

test Xray config.

### xray.go

start and stop Xray instance.

## lib package

### build

build libXray, currently support android and apple.

It will always use the latest Xray-core.

### controller.go

experimental Android support.

register a controller to protect all connections.

Because there is no api to reset effectiveListener, Only run them once when app running.

### dns.go

experimental Android support.

register a controller to protect default dns queries.

If the xray server address is a domain, InitDns will be useful to resolve the "first connection" problem.

### nodep_wrapper.go

export nodep.

### xray_wrapper.go

export xray.

## Credits

[Project X](https://github.com/XTLS/Xray-core)

[VMessPing](https://github.com/v2fly/vmessping)

[FreePort](https://github.com/phayes/freeport)
