# app/android.py
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

        # Set environment variables for 16KB page alignment (Android 15+ compatibility)
        env = os.environ.copy()

        # Minimal 16KB page alignment configuration - only set the essential flags
        cgo_ldflags = (
            "-Wl,-z,max-page-size=0x4000 "
            "-Wl,-z,common-page-size=0x4000"
        )

        # Set only the essential environment variables
        env["CGO_LDFLAGS"] = cgo_ldflags

        # Build with minimal 16KB page alignment (no additional ldflags to preserve original size)
        print("Building with minimal 16KB page alignment for Android 15+ compatibility...")
        ret = subprocess.run(
            [
                "gomobile", "bind",
                "-target", "android",
                "-androidapi", "21"
            ],
            env=env
        )
        if ret.returncode != 0:
            raise Exception("build failed")
