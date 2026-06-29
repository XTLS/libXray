import os.path
import re
import shutil
import subprocess

from app.cmd import (
    delete_file_if_exists,
    delete_dir_if_exists,
)

LIBXRAY_MOD_NAME = "github.com/xtls/libxray"
XRAY_CORE_MOD_NAME = "github.com/xtls/xray-core"
DEFAULT_XRAY_CORE_TAG = "v26.6.27"
# Xray-core CalVer tags cannot be used directly as Go module versions because
# its module path does not include /v26. This pseudo-version points to the tag above.
DEFAULT_XRAY_CORE_VERSION = "v1.260327.1-0.20260627131803-45cf2898ab12"
LOCAL_XRAY_CORE_DIR_NAME = "Xray-core"


class Builder(object):
    def __init__(self, build_dir: str, use_local_xray_core: bool = False):
        self.build_dir = build_dir
        self.lib_dir = os.path.abspath(os.path.join(self.build_dir, ".."))
        self.use_local_xray_core = use_local_xray_core
        self.xray_core_replace_path = f"../{LOCAL_XRAY_CORE_DIR_NAME}"
        self.xray_core_dir = os.path.abspath(
            os.path.join(self.lib_dir, self.xray_core_replace_path)
        )

    def clean_lib_files(self, files: list[str]):
        for file in files:
            file_path = os.path.join(self.lib_dir, file)
            delete_file_if_exists(file_path)

    def clean_lib_dirs(self, dirs: list[str]):
        for dir_name in dirs:
            dir_path = os.path.join(self.lib_dir, dir_name)
            delete_dir_if_exists(dir_path)

    def prepare_xray_core(self):
        if self.use_local_xray_core:
            if not os.path.isdir(self.xray_core_dir):
                raise Exception(f"local Xray-core dir not found: {self.xray_core_dir}")

    def init_go_env(self):
        os.chdir(self.lib_dir)
        if not os.path.exists(os.path.join(self.lib_dir, "go.mod")):
            ret = subprocess.run(["go", "mod", "init", LIBXRAY_MOD_NAME])
            if ret.returncode != 0:
                raise Exception("go mod init failed")

        if self.use_local_xray_core:
            ret = subprocess.run(
                [
                    "go",
                    "mod",
                    "edit",
                    f"-replace={XRAY_CORE_MOD_NAME}={self.xray_core_replace_path}",
                ]
            )
            if ret.returncode != 0:
                raise Exception("go mod edit replace failed")
        else:
            ret = subprocess.run(
                ["go", "mod", "edit", f"-dropreplace={XRAY_CORE_MOD_NAME}"]
            )
            if ret.returncode != 0:
                raise Exception("go mod edit dropreplace failed")

            ret = subprocess.run(
                ["go", "get", f"{XRAY_CORE_MOD_NAME}@{DEFAULT_XRAY_CORE_VERSION}"]
            )
            if ret.returncode != 0:
                raise Exception("go get xray-core failed")

        ret = subprocess.run(
            [
                "go",
                "mod",
                "tidy",
            ]
        )
        if ret.returncode != 0:
            raise Exception("go mod tidy failed")

    def download_geo(self):
        os.chdir(self.lib_dir)
        main_path = os.path.join("download_geo", "main.go")
        ret = subprocess.run(["go", "run", main_path])
        if ret.returncode != 0:
            raise Exception("download_geo failed")

    def prepare_gomobile(self):
        ret = subprocess.run(
            ["go", "install", "golang.org/x/mobile/cmd/gomobile@latest"]
        )
        if ret.returncode != 0:
            raise Exception("go install gomobile failed")
        ret = subprocess.run(["gomobile", "init"])
        if ret.returncode != 0:
            raise Exception("gomobile init failed")
        ret = subprocess.run(["go", "get", "golang.org/x/mobile/cmd/gomobile"])
        if ret.returncode != 0:
            raise Exception("gomobile update failed")
        ret = subprocess.run(["go", "get", "google.golang.org/genproto"])
        if ret.returncode != 0:
            raise Exception("gomobile install genproto failed")

    def prepare_static_lib(self):
        self.copy_template_file()
        self.fix_package_name()

    def copy_template_file(self):
        src_file = os.path.join(self.build_dir, "template", "main.gotemplate")
        dst_file = os.path.join(self.lib_dir, "main.go")
        shutil.copy(src_file, dst_file)

    def fix_package_name(self):
        files = os.listdir(self.lib_dir)
        for file in files:
            if file.endswith(".go"):
                self.replace_package_name(file)

    def replace_package_name(self, file_name: str):
        file_path = os.path.join(self.lib_dir, file_name)
        new_lines = []
        with open(file_path, "r") as f:
            lines = f.readlines()
            for line in lines:
                new_line = line
                if re.match(r"^package\s+libXray", line):
                    new_line = "package main\n"
                new_lines.append(new_line)
        with open(file_path, "w") as f:
            f.writelines(new_lines)

    def before_build(self):
        self.prepare_xray_core()
        self.init_go_env()
        self.download_geo()

    def build(self):
        pass

    def after_build(self):
        pass

    def reset_files(self):
        self.clean_lib_files(["main.go"])
        files = os.listdir(self.lib_dir)
        for file in files:
            if file.endswith(".go"):
                self.reset_package_name(file)

    def reset_package_name(self, file_name: str):
        file_path = os.path.join(self.lib_dir, file_name)
        new_lines = []
        with open(file_path, "r") as f:
            lines = f.readlines()
            for line in lines:
                new_line = line
                if re.match(r"^package\s+main", line):
                    new_line = "package libXray\n"
                new_lines.append(new_line)
        with open(file_path, "w") as f:
            f.writelines(new_lines)

    def revert_go_env(self):
        os.chdir(self.lib_dir)
        ret = subprocess.run(
            ["go", "mod", "edit", f"-dropreplace={XRAY_CORE_MOD_NAME}"]
        )
        if ret.returncode != 0:
            raise Exception("go mod edit dropreplace failed")

        ret = subprocess.run(
            ["go", "get", f"{XRAY_CORE_MOD_NAME}@{DEFAULT_XRAY_CORE_VERSION}"]
        )
        if ret.returncode != 0:
            raise Exception("go get xray-core failed")

        ret = subprocess.run(["go", "mod", "tidy"])
        if ret.returncode != 0:
            raise Exception("go mod tidy failed")
