# How to use stats.go

Here is a config sample.

```json
{
  "inbounds" : [
    {
      "listen" : "[::1]",
      "port" : 49227,
      "protocol" : "dokodemo-door",
      "settings" : {
        "address" : "[::1]"
      },
      "tag" : "metricsIn"
    }
  ],
  "metrics" : {
    "tag" : "metricsOut"
  },
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
          "metricsIn"
        ],
        "outboundTag" : "metricsOut",
        "type" : "field"
      }
    ]
  },
  "stats" : {}
}
```

Then call `func QueryStats(base64Text string) string` in your app.

The server should be "[::1]:49227".

Attention: 

1. never enable metrics when ping and testing.

2. always running Xray-core instance using standalone process, if you enable metrics.
