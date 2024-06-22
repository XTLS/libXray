# libXray build

## usage

```shell
python3 build/main.py android
python3 build/main.py apple gomobile
python3 build/main.py apple go
```

Other platforms are WIP.

## Android

use [gomobile](https://github.com/golang/mobile)

## iOS && macOS

### 1. use gomobile

It is the best choice for most cases. Good apis, no conflicts
with other frameworks.

But it does NOT support to set minimal macOS version of xcframework.

### 2. use cgo

More controls in building progress, c header file output.

Useful for ffi, like swift, dart.

DO NOT contain **module.modulemap**. You need create a bridge header file when using it in swift. 

