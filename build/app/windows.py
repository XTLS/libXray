import os.path
import subprocess

from app.build import Builder
from app.cmd import create_dir_if_not_exists, delete_dir_if_exists


# https://ziglang.org/learn/overview/#zig-ships-with-libc


class WindowsTarget(object):
    def __init__(self, platform: str, go_arch: str, zig_target: str):
        self.platform = platform
        self.go_arch = go_arch
        self.zig_target = zig_target


class WindowsBuilder(Builder):
    def __init__(self, build_dir: str):
        super().__init__(build_dir)
        self.framework_dir = os.path.join(self.lib_dir, "windows_dll")
        delete_dir_if_exists(self.framework_dir)
        create_dir_if_not_exists(self.framework_dir)
        self.lib_file = "libXray.dll"
        self.lib_header_file = "libXray.h"

        self.targets = [
            WindowsTarget(
                "windows",
                "arm64",
                "aarch64-windows-gnu",
            ),
            WindowsTarget(
                "windows",
                "amd64",
                "x86_64-windows-gnu",
            ),
        ]

    def before_build(self):
        super().before_build()
        self.prepare_static_lib()

    def build(self):
        self.before_build()
        self.build_windows(self.targets)
        self.after_build()

    def build_windows(self, targets: list[WindowsTarget]):
        for target in targets:
            self.run_build_cmd(
                target.platform,
                target.go_arch,
                target.zig_target,
            )

    def run_build_cmd(self, platform: str, go_arch: str, zig_target: str):
        output_dir = os.path.join(self.framework_dir, go_arch)
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, self.lib_file)
        run_env = os.environ.copy()
        run_env["GOOS"] = platform
        run_env["GOARCH"] = go_arch
        run_env["CC"] = f"zig cc -target {zig_target}"
        run_env["CXX"] = f"zig c++ -target {zig_target}"
        run_env["CGO_ENABLED"] = "1"

        cmd = [
            "go",
            "build",
            "-ldflags=-w",
            f"-o={output_file}",
            "-buildmode=c-shared",
        ]
        os.chdir(self.lib_dir)
        print(run_env)
        print(cmd)
        ret = subprocess.run(cmd, env=run_env)
        if ret.returncode != 0:
            raise Exception(f"run_build_cmd for {platform} {go_arch} failed")

    def after_build(self):
        super().after_build()
        self.reset_files()
