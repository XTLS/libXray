# How to use stats.go

Here is a config sample.

```json
{
  "api" : {
    "services" : [
      "StatsService"
    ],
    "tag" : "api"
  },
  "inbounds" : [
    {
      "listen" : "[::1]",
      "port" : 63822,
      "protocol" : "socks",
      "settings" : {
        "auth" : "noauth",
        "udp" : true
      },
      "tag" : "socks"
    },
    {
      "listen" : "[::1]",
      "port" : 63823,
      "protocol" : "dokodemo-door",
      "settings" : {
        "address" : "[::1]"
      },
      "tag" : "api"
    }
  ],
  "outbounds" : [
    {
      "protocol" : "freedom",
      "tag" : "direct"
    }
  ],
  "policy" : {
    "system" : {
      "statsInboundDownlink" : true,
      "statsInboundUplink" : true,
      "statsOutboundDownlink" : true,
      "statsOutboundUplink" : true
    }
  },
  "routing" : {
    "domainStrategy" : "IpIfNonMatch",
    "rules" : [
      {
        "inboundTag" : [
          "api"
        ],
        "outboundTag" : "api",
        "type" : "field"
      },
      {
        "domain" : [
          "geosite:PRIVATE"
        ],
        "outboundTag" : "direct",
        "type" : "field"
      },
      {
        "ip" : [
          "geoip:PRIVATE",
        ],
        "outboundTag" : "direct",
        "type" : "field"
      }
    ]
  },
  "stats" : {}
}
```

Then call `func QueryStats(server string, dir string) string` in your app.

The server should be "[::1]:63823".

Your will get result json files in the "dir".

