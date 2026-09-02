from datetime import datetime, timezone
import hashlib
import json
import os.path
import shutil
import subprocess
import sys

from app.cmd import (
    create_dir_if_not_exists,
    delete_file_if_exists,
    delete_dir_if_exists,
)

LIBXRAY_MOD_NAME = "github.com/xtls/libxray"
XRAY_CORE_MOD_NAME = "github.com/xtls/xray-core"
# Go modules resolve the Xray-core v26.7.28 release tag through this version.
DEFAULT_XRAY_CORE_VERSION = "v1.260327.1-0.20260728075948-5ca6f4b7d4dc"
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
        self._gomobile_version = None

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
        requested_version = os.environ.get("LIBXRAY_GOMOBILE_VERSION") or "latest"
        result = subprocess.run(
            [
                "go",
                "list",
                "-m",
                "-f",
                "{{.Version}}",
                f"golang.org/x/mobile@{requested_version}",
            ],
            capture_output=True,
            text=True,
        )
        version = result.stdout.strip()
        if result.returncode != 0 or not version:
            raise Exception("resolve gomobile version failed")
        self._gomobile_version = version

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

    def build_desktop_bin(self, file_name: str):
        output_dir = os.path.join(self.lib_dir, "bin")
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, file_name)
        run_env = os.environ.copy()
        run_env["CGO_ENABLED"] = "0"
        cmd = [
            "go",
            "build",
            "-trimpath",
            "-buildvcs=false",
            "-ldflags",
            "-s -w -buildid=",
            f"-o={output_file}",
            "./desktop_bin",
        ]
        print(cmd)
        ret = subprocess.run(cmd, cwd=self.lib_dir, env=run_env)
        if ret.returncode != 0:
            raise Exception("build_desktop_bin failed")

    def before_build(self):
        self.prepare_xray_core()
        self.init_go_env()
        self.download_geo()

    def build(self):
        pass

    def after_build(self):
        # Called in finally, before restoring the effective module files. This
        # records inputs even on failure; it never certifies an artifact.
        try:
            builder = {
                "AndroidBuilder": "android",
                "AppleGoBuilder": "apple-go",
                "AppleGoMobileBuilder": "apple-gomobile",
                "LinuxBuilder": "linux",
                "WindowsBuilder": "windows",
            }.get(type(self).__name__, type(self).__name__)
            errors = []

            def capture(label, operation):
                try:
                    return operation()
                except Exception as error:
                    errors.append(f"{label}: {error}")
                    return None

            def output(*command):
                return subprocess.run(
                    command,
                    cwd=self.lib_dir,
                    check=True,
                    capture_output=True,
                    text=True,
                    timeout=60,
                ).stdout.strip()

            def file_hash(name):
                with open(os.path.join(self.lib_dir, name), "rb") as file:
                    return hashlib.sha256(file.read()).hexdigest()

            metadata = {
                "schemaVersion": 1,
                "evidence": "build-inputs-only",
                "builder": builder,
                "recordedAt": datetime.now(timezone.utc).isoformat(),
                "goModSha256": capture("go.mod", lambda: file_hash("go.mod")),
                "goSumSha256": capture("go.sum", lambda: file_hash("go.sum")),
                "libXrayCommit": capture(
                    "git commit", lambda: output("git", "rev-parse", "HEAD")
                ),
                "goVersion": capture("Go version", lambda: output("go", "version")),
                "modules": capture(
                    "Go modules",
                    lambda: output("go", "list", "-mod=readonly", "-m", "all"),
                ),
                "gomobile": None,
                "errors": errors,
            }
            status = capture(
                "git status",
                lambda: output("git", "status", "--porcelain", "--untracked-files=no"),
            )
            metadata["libXrayDirty"] = None if status is None else bool(status)
            if self._gomobile_version is not None:
                binary = shutil.which("gomobile")
                build_info = None
                used_version = None
                if binary is None:
                    errors.append("gomobile binary: not found in PATH")
                else:
                    build_info = capture(
                        "gomobile build info",
                        lambda: output("go", "version", "-m", binary),
                    )
                    for line in (build_info or "").splitlines():
                        fields = line.split()
                        if (
                            fields[:2] == ["mod", "golang.org/x/mobile"]
                            and len(fields) >= 3
                        ):
                            used_version = fields[2]
                            break
                    if build_info is not None and used_version is None:
                        errors.append("gomobile build info: module version missing")
                metadata["gomobile"] = {
                    "resolvedVersion": self._gomobile_version,
                    "usedVersion": used_version,
                    "binary": binary,
                    "buildInfo": build_info,
                }
            path = os.path.join(self.build_dir, f"build-metadata-{builder}.json")
            with open(path, "w", encoding="utf-8") as file:
                json.dump(metadata, file, indent=2)
                file.write("\n")
            if errors:
                print(f"Build input metadata is incomplete: {path}", file=sys.stderr)
        except Exception as error:
            # A metadata failure must not replace the original build exception.
            print(f"Unable to record build input metadata: {error}", file=sys.stderr)
