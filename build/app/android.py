# app/android.py
import os
import subprocess

from app.build import Builder


class AndroidBuilder(Builder):
    def __init__(self, build_dir, android_api="21"):
        super().__init__(build_dir)
        self.android_api = android_api

    def before_build(self):
        super().before_build()
        self.prepare_gomobile()

    def build(self):
        self.before_build()

        clean_files = ["libXray-sources.jar", "libXray.aar"]
        self.clean_lib_files(clean_files)
        os.chdir(self.lib_dir)
        ret = subprocess.run(
            ["gomobile", "bind", "-target", "android", "-androidapi", self.android_api]
        )
        if ret.returncode != 0:
            raise Exception("build failed")
