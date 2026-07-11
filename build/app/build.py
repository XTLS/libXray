import os.path
import subprocess

from app.cmd import (
    delete_file_if_exists,
    delete_dir_if_exists,
)

LIBXRAY_MOD_NAME = "github.com/xtls/libxray"
XRAY_CORE_MOD_NAME = "github.com/xtls/xray-core"
# This pseudo-version pins the Xray-core main commit that added root env config.
DEFAULT_XRAY_CORE_VERSION = "v1.260327.1-0.20260710025649-d5bc58dc6b76"
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
        self._go_env_snapshot = None

    def snapshot_go_env(self):
        paths = [
            os.path.join(self.lib_dir, "go.mod"),
            os.path.join(self.lib_dir, "go.sum"),
        ]
        snapshot = {}
        for path in paths:
            if not os.path.exists(path):
                snapshot[path] = None
                continue
            with open(path, "rb") as file:
                snapshot[path] = file.read()
        self._go_env_snapshot = snapshot

    def restore_go_env(self):
        if self._go_env_snapshot is None:
            return
        snapshot = self._go_env_snapshot
        for path, content in snapshot.items():
            if content is None:
                delete_file_if_exists(path)
                continue
            with open(path, "wb") as file:
                file.write(content)
        self._go_env_snapshot = None

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
        result = subprocess.run(
            [
                "go",
                "list",
                "-m",
                "-f",
                "{{.Version}}",
                "golang.org/x/mobile@latest",
            ],
            capture_output=True,
            text=True,
        )
        version = result.stdout.strip()
        if result.returncode != 0 or not version:
            raise Exception("resolve latest gomobile version failed")

        ret = subprocess.run(
            [
                "go",
                "get",
                "-tool",
                f"golang.org/x/mobile/cmd/gobind@{version}",
            ]
        )
        if ret.returncode != 0:
            raise Exception("add gobind tool dependency failed")

        ret = subprocess.run(
            [
                "go",
                "install",
                f"golang.org/x/mobile/cmd/gomobile@{version}",
            ]
        )
        if ret.returncode != 0:
            raise Exception("go install gomobile failed")
        ret = subprocess.run(["gomobile", "init"])
        if ret.returncode != 0:
            raise Exception("gomobile init failed")

    def prepare_static_lib(self):
        main_file = os.path.join(self.lib_dir, "cgo_bridge", "main.go")
        if not os.path.isfile(main_file):
            raise Exception("cgo bridge entrypoint is missing")

    def main_package(self) -> str:
        return "./cgo_bridge"

    def before_build(self):
        self.prepare_xray_core()
        self.init_go_env()
        self.download_geo()

    def build(self):
        pass

    def after_build(self):
        pass
