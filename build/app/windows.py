import os.path
import subprocess

from app.build import Builder
from app.cmd import create_dir_if_not_exists, delete_dir_if_exists


class WindowsBuilder(Builder):
    def __init__(self, build_dir: str):
        super().__init__(build_dir)
        self.framework_dir = os.path.join(self.lib_dir, "windows_dll")
        delete_dir_if_exists(self.framework_dir)
        create_dir_if_not_exists(self.framework_dir)
        self.lib_file = "libXray.dll"
        self.lib_header_file = "libXray.h"

    def before_build(self):
        super().before_build()
        self.prepare_static_lib()

    def build(self):
        self.before_build()
        self.build_windows()
        self.after_build()

    def build_windows(self):
        output_dir = self.framework_dir
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, self.lib_file)
        run_env = os.environ.copy()
        run_env["CC"] = "clang.exe"
        run_env["CXX"] = "clang++.exe"
        run_env["CGO_ENABLED"] = "1"

        cmd = [
            "go",
            "build",
            "-trimpath",
            "-ldflags",
            "-s -w",
            f"-o={output_file}",
            "-buildmode=c-shared",
        ]
        os.chdir(self.lib_dir)
        print(cmd)
        ret = subprocess.run(cmd, env=run_env)
        if ret.returncode != 0:
            raise Exception(f"run_build_cmd failed")

    def after_build(self):
        super().after_build()
        self.reset_files()
