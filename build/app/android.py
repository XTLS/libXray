import os
import subprocess

from app.build import Builder


class AndroidBuilder(Builder):
    def before_build(self):
        super().before_build()
        self.prepare_gomobile()

    def build(self):
        self.before_build()

        clean_files = ["libXray-sources.jar", "libXray.aar"]
        self.clean_lib_files(clean_files)
        os.chdir(self.lib_dir)
        env = os.environ.copy()
        env["CGO_LDFLAGS"] = "-O2 -g -s -w -Wl,-z,max-page-size=16384"
        ret = subprocess.run(
            ["gomobile", "bind", "-target", "android", "-androidapi", "21"],
            env=env
        )
        if ret.returncode != 0:
            raise Exception("build failed")
