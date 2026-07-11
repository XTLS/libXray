# app/android.py
import os
import subprocess

from app.build import Builder


class AndroidBuilder(Builder):

    def before_build(self):
        super().before_build()
        self.prepare_gomobile()

    def build(self):
        self.snapshot_go_env()
        try:
            self.before_build()

            clean_files = ["libXray-sources.jar", "libXray.aar"]
            self.clean_lib_files(clean_files)
            os.chdir(self.lib_dir)
            # keep same with flutter
            ret = subprocess.run(
                [
                    "gomobile",
                    "bind",
                    "-target",
                    "android",
                    "-androidapi",
                    "21",
                    "-ldflags=-checklinkname=0 -extldflags=-Wl,-z,max-page-size=16384",
                ]
            )
            if ret.returncode != 0:
                raise Exception("build failed")
        finally:
            try:
                self.after_build()
            finally:
                self.restore_go_env()
