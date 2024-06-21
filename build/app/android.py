import os
import subprocess

from app.build import Builder


class AndroidBuilder(Builder):
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
        self.prepare_gomobile()

    def build(self):
        super().build()

        clean_files = ["libXray.jar", "libXray.aar"]
        self.clean_lib_files(clean_files)
        os.chdir(self.lib_dir)
        ret = subprocess.run(
            ["gomobile", "bind", "-target", "android", "-androidapi", "28"]
        )
        if ret.returncode != 0:
            raise Exception("build failed")
