import os.path
import re
import shutil
import subprocess

from app.cmd import (
    create_dir_if_not_exists,
    delete_file_if_exists,
    delete_dir_if_exists,
)

XRAY_CORE_REPO = "https://github.com/XTLS/Xray-core.git"
XRAY_CORE_VERSION = "v26.2.6"
XRAY_CORE_DIR_NAME = "Xray-core-libXray"
LIBXRAY_MOD_NAME = "github.com/xtls/libxray"


class Builder(object):
    def __init__(self, build_dir: str):
        self.build_dir = build_dir
        self.lib_dir = os.path.join(self.build_dir, "..")
        self.bin_file = "xray"

    def clean_lib_files(self, files: list[str]):
        for file in files:
            file_path = os.path.join(self.lib_dir, file)
            delete_file_if_exists(file_path)

    def clean_lib_dirs(self, dirs: list[str]):
        for dir_name in dirs:
            dir_path = os.path.join(self.lib_dir, dir_name)
            delete_dir_if_exists(dir_path)

    def clone_xray_core(self):
        xray_core_dir = os.path.join(self.lib_dir, "..", XRAY_CORE_DIR_NAME)
        delete_dir_if_exists(xray_core_dir)
        ret = subprocess.run(
            [
                "git",
                "clone",
                "--depth",
                "1",
                "--branch",
                XRAY_CORE_VERSION,
                XRAY_CORE_REPO,
                xray_core_dir,
            ]
        )
        if ret.returncode != 0:
            raise Exception("git clone Xray-core failed")

    def init_go_env(self):
        os.chdir(self.lib_dir)
        self.clean_lib_files(["go.mod", "go.sum"])
        ret = subprocess.run(["go", "mod", "init", LIBXRAY_MOD_NAME])
        if ret.returncode != 0:
            raise Exception("go mod init failed")
        ret = subprocess.run(
            [
                "go",
                "mod",
                "edit",
                "-replace",
                f"github.com/xtls/xray-core=../{XRAY_CORE_DIR_NAME}",
            ]
        )
        if ret.returncode != 0:
            raise Exception("go mod edit failed")

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
        files = ["main.go"]
        for file in files:
            src_file = os.path.join(self.build_dir, "template", file)
            shutil.copy(src_file, self.lib_dir)

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
        self.clone_xray_core()
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

    def build_desktop_bin(self):
        bin_dir = os.path.join(self.lib_dir, "bin")
        create_dir_if_not_exists(bin_dir)
        output_file = os.path.join(bin_dir, self.bin_file)
        run_env = os.environ.copy()
        run_env["CGO_ENABLED"] = "0"

        cmd = [
            "go",
            "build",
            "-trimpath",
            "-ldflags",
            "-s -w",
            f"-o={output_file}",
            "./desktop_bin",
        ]
        os.chdir(self.lib_dir)
        print(cmd)
        ret = subprocess.run(cmd, env=run_env)
        if ret.returncode != 0:
            raise Exception(f"build_desktop_bin failed")

    def revert_go_env(self):
        os.chdir(self.lib_dir)
        self.clean_lib_files(["go.mod", "go.sum"])
        ret = subprocess.run(["go", "mod", "init", LIBXRAY_MOD_NAME])
        if ret.returncode != 0:
            raise Exception("go mod init failed")

        ret = subprocess.run(
            [
                "go",
                "mod",
                "tidy",
            ]
        )
        if ret.returncode != 0:
            raise Exception("go mod tidy failed")
