# app/android.py
import os
import subprocess

from app.build import Builder


class AndroidBuilder(Builder):

    def before_build(self):
        super().before_build()
        self.prepare_gomobile()

    def build(self):
        api = int(os.environ.get("LIBXRAY_ANDROID_API") or 21)
        extldflags = "-Wl,-z,max-page-size=16384"
        if api >= 28:  # DT_ANDROID_RELR, which Android loads since API 28
            extldflags += ",--pack-dyn-relocs=relr,--use-android-relr-tags"
        elif api >= 23:  # DT_ANDROID_REL(A), which Android loads since API 23
            extldflags += ",--pack-dyn-relocs=android"
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
                    str(api),
                    "-trimpath",
                    "-ldflags=-checklinkname=0 -s -w -buildid= "
                    f"-extldflags={extldflags}",
                ]
            )
            if ret.returncode != 0:
                raise Exception("build failed")
        finally:
            self.restore_go_env()
