import os
import subprocess

from app.build import Builder


class AppleGoMobileBuilder(Builder):
    def prepare_gomobile(self):
        ret = subprocess.run(
            ["go", "install", "golang.org/x/mobile/cmd/gomobile@latest"]
        )
        if ret.returncode != 0:
            raise Exception("go install gomobile failed")
        ret = subprocess.run(["gomobile", "init"])
        if ret.returncode != 0:
            raise Exception("gomobile init failed")
        ret = subprocess.run(["go", "get", "-d", "golang.org/x/mobile/cmd/gomobile"])
        if ret.returncode != 0:
            raise Exception("gomobile init failed")

    def before_build(self):
        super().before_build()
        self.clean_lib_dirs(["LibXray.xcframework"])
        self.prepare_gomobile()

    def build(self):
        self.before_build()

        os.chdir(self.lib_dir)
        ret = subprocess.run(
            [
                "gomobile",
                "bind",
                "-target",
                "ios,iossimulator,macos",
                "-iosversion",
                "15.0",
            ]
        )
        if ret.returncode != 0:
            raise Exception("build failed")
