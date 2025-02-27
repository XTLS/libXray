# libXray build

## usage

```shell
python3 build/main.py android
python3 build/main.py apple gomobile
python3 build/main.py apple go
python3 build/main.py linux
python3 build/main.py windows
```

## Android

use [gomobile](https://github.com/golang/mobile)

## iOS && macOS

### 1. use gomobile

Need "iOS Simulator Runtime".

It is the best choice for most cases. Good apis, no conflicts
with other frameworks.

Support iOS, iOSSimulator, macOS, macCatalyst.

But it does NOT support to set minimal macOS version of xcframework, and has no tvOS support.

### 2. use cgo

Need "iOS Simulator Runtime" and "tvOS Simulator Runtime".

More controls in building progress, c header file output.

Useful for ffi, like swift, kotlin, dart.

Support iOS, iOSSimulator, macOS, tvOS.

DO NOT contain **module.modulemap**. You need create a bridging-header file when using it in swift. 

## Linux

depends on gcc && g++

## Windows

depends on [LLVM MinGW](https://github.com/mstorsjo/llvm-mingw), you can install it by

```shell
winget install MartinStorsjo.LLVM-MinGW.UCRT
```