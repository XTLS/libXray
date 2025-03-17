import os.path
import shutil
import subprocess
import random
import time

from app.build import Builder
from app.cmd import create_dir_if_not_exists, delete_dir_if_exists


# https://github.com/golang/mobile/blob/master/cmd/gomobile/build_darwin_test.go


class AppleTarget(object):
    def __init__(
        self, platform: str, go_arch: str, apple_arch: str, sdk: str, min_version: str
    ):
        self.platform = platform
        self.go_arch = go_arch
        self.apple_arch = apple_arch
        self.sdk = sdk
        self.min_version = min_version


class AppleGoBuilder(Builder):
    def __init__(self, build_dir: str):
        super().__init__(build_dir)
        self.framework_dir = os.path.join(self.lib_dir, "apple_xcframework")
        delete_dir_if_exists(self.framework_dir)
        create_dir_if_not_exists(self.framework_dir)
        self.lib_file = "libXray.a"
        self.lib_header_file = "libXray.h"

        self.ios_targets = [
            AppleTarget(
                "ios",
                "arm64",
                "arm64",
                "iphoneos",
                "15.0",
            ),
            AppleTarget(
                "ios",
                "amd64",
                "x86_64",
                "iphonesimulator",
                "15.0",
            ),
            AppleTarget(
                "ios",
                "arm64",
                "arm64",
                "iphonesimulator",
                "15.0",
            ),
        ]

        # keep same with flutter
        self.macos_targets = [
            AppleTarget(
                "darwin",
                "amd64",
                "x86_64",
                "macosx",
                "10.14",
            ),
            AppleTarget(
                "darwin",
                "arm64",
                "arm64",
                "macosx",
                "10.14",
            ),
        ]

        self.tvos_targets = [
            AppleTarget(
                "ios",
                "arm64",
                "arm64",
                "appletvos",
                "17.0",
            ),
            AppleTarget(
                "ios",
                "amd64",
                "x86_64",
                "appletvsimulator",
                "17.0",
            ),
            AppleTarget(
                "ios",
                "arm64",
                "arm64",
                "appletvsimulator",
                "17.0",
            ),
        ]

    def before_build(self):
        super().before_build()
        self.clean_lib_dirs(["LibXray.xcframework"])
        self.prepare_static_lib()

    def build(self):
        self.before_build()
        # build ios
        self.build_targets(self.ios_targets)
        self.merge_static_lib(
            self.ios_targets[1].sdk,
            [self.ios_targets[1].apple_arch, self.ios_targets[2].apple_arch],
        )
        # build macos
        self.build_targets(self.macos_targets)
        self.merge_static_lib(
            self.macos_targets[0].sdk,
            [self.macos_targets[0].apple_arch, self.macos_targets[1].apple_arch],
        )
        # build tvos
        self.build_targets(self.tvos_targets)
        self.merge_static_lib(
            self.tvos_targets[1].sdk,
            [self.tvos_targets[1].apple_arch, self.tvos_targets[2].apple_arch],
        )

        self.after_build()

        self.create_include_dir()
        self.create_framework()

    def build_targets(self, targets: list[AppleTarget]):
        for target in targets:
            self.run_build_cmd(
                target.platform,
                target.go_arch,
                target.apple_arch,
                target.sdk,
                target.min_version,
            )

    def run_build_cmd(
        self, platform: str, go_arch: str, apple_arch: str, sdk: str, min_version: str
    ):
        output_dir = os.path.join(self.framework_dir, f"{sdk}-{apple_arch}")
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, self.lib_file)
        sdk_path = self.get_sdk_dir_path(sdk)
        min_version_flag = f"-m{sdk}-version-min={min_version}"
        flags = f"-isysroot {sdk_path} {min_version_flag} -arch {apple_arch}"
        run_env = os.environ.copy()
        run_env["GOOS"] = platform
        run_env["GOARCH"] = go_arch
        run_env["GOFLAGS"] = f"-tags={platform}"
        run_env["CC"] = f"xcrun --sdk {sdk} --toolchain {sdk} clang"
        run_env["CXX"] = f"xcrun --sdk {sdk} --toolchain {sdk} clang++"
        run_env["CGO_CFLAGS"] = flags
        run_env["CGO_CXXFLAGS"] = flags
        run_env["CGO_LDFLAGS"] = f"{flags} -Wl,-Bsymbolic-functions"
        run_env["CGO_ENABLED"] = "1"
        run_env["DARWIN_SDK"] = sdk

        # Enhanced obfuscation for uniqueness of each build
        ldflags = "-s -w"  # Strip symbol table and DWARF debug info

        # Add build timestamp with microseconds for greater uniqueness
        build_timestamp = time.strftime("%Y%m%d%H%M%S") + str(
            int(time.time() * 1000) % 1000
        )
        ldflags += f" -X 'main.buildTime={build_timestamp}'"

        # Add random identifier for each build
        random_id = "".join(
            random.choices(
                "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", k=16
            )
        )
        ldflags += f" -X 'main.buildID={random_id}'"

        # Add git commit hash if available
        try:
            git_hash = (
                subprocess.check_output(
                    ["git", "rev-parse", "--short", "HEAD"], stderr=subprocess.DEVNULL
                )
                .decode()
                .strip()
            )
            ldflags += f" -X 'main.buildVersion={git_hash}'"
        except:
            # If git is not available, use a random hash
            fake_hash = "".join(random.choices("0123456789abcdef", k=7))
            ldflags += f" -X 'main.buildVersion={fake_hash}'"

        # Add random unused variables to change binary fingerprint
        for i in range(5):
            var_name = f"unusedVar{i}"
            var_value = "".join(
                random.choices(
                    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
                    k=8,
                )
            )
            ldflags += f" -X 'main.{var_name}={var_value}'"

        # Use compatible optimization flags
        gcflags = "all=-l=4"  # Limit inlining but keep other optimizations

        cmd = [
            "go",
            "build",
            "-trimpath",  # Remove all file system paths
            "-ldflags",
            ldflags,
            "-gcflags",
            gcflags,
            f"-o={output_file}",
            "-buildmode=c-archive",
        ]
        os.chdir(self.lib_dir)
        print(cmd)
        ret = subprocess.run(cmd, env=run_env)
        if ret.returncode != 0:
            raise Exception(f"run_build_cmd for {platform} {apple_arch} {sdk} failed")

    def get_sdk_dir_path(self, sdk: str) -> str:
        cmd = [
            "xcrun",
            "--sdk",
            sdk,
            "--show-sdk-path",
        ]
        print(cmd)
        ret = subprocess.run(cmd, capture_output=True)
        if ret.returncode != 0:
            raise Exception(f"get_sdk_dir_path for {sdk} failed")
        return ret.stdout.decode().replace("\n", "")

    def merge_static_lib(self, sdk: str, arches: list[str]):
        cmd = [
            "lipo",
            "-create",
        ]
        for arch in arches:
            lib_dir = os.path.join(self.framework_dir, f"{sdk}-{arch}")
            lib_file = os.path.join(lib_dir, self.lib_file)
            cmd.extend(["-arch", arch, lib_file])
        arch = "-".join(arches)
        output_dir = os.path.join(self.framework_dir, f"{sdk}-{arch}")
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, self.lib_file)
        cmd.extend(["-output", output_file])
        print(cmd)
        ret = subprocess.run(cmd)
        if ret.returncode != 0:
            raise Exception(f"merge_static_lib for {sdk} failed")

    def create_include_dir(self):
        include_dir = os.path.join(self.framework_dir, "include")
        create_dir_if_not_exists(include_dir)

        target = self.ios_targets[0]
        header_file = os.path.join(
            self.framework_dir,
            f"{target.sdk}-{target.apple_arch}",
            self.lib_header_file,
        )
        shutil.copy(header_file, include_dir)

    def create_framework(self):
        libs = [
            f"{self.ios_targets[0].sdk}-{self.ios_targets[0].apple_arch}",
            f"{self.ios_targets[1].sdk}-{self.ios_targets[1].apple_arch}-{self.ios_targets[2].apple_arch}",
            f"{self.macos_targets[0].sdk}-{self.macos_targets[0].apple_arch}-{self.macos_targets[1].apple_arch}",
            f"{self.tvos_targets[0].sdk}-{self.tvos_targets[0].apple_arch}",
            f"{self.tvos_targets[1].sdk}-{self.tvos_targets[1].apple_arch}-{self.tvos_targets[2].apple_arch}",
        ]
        include_dir = os.path.join(self.framework_dir, "include")
        cmd = ["xcodebuild", "-create-xcframework"]
        for lib in libs:
            lib_path = os.path.join(self.framework_dir, lib, self.lib_file)
            cmd.extend(["-library", lib_path, "-headers", include_dir])

        output_file = os.path.join(self.lib_dir, "LibXray.xcframework")

        cmd.extend(["-output", output_file])

        print(cmd)
        ret = subprocess.run(cmd)
        if ret.returncode != 0:
            raise Exception(f"create_framework failed")

    def after_build(self):
        super().after_build()
        self.reset_files()

        # Modify framework metadata for additional uniqueness
        try:
            framework_path = os.path.join(self.lib_dir, "LibXray.xcframework")
            info_plist = os.path.join(framework_path, "Info.plist")
            if os.path.exists(info_plist):
                # Add random comment to Info.plist
                random_comment = f"<!-- Build ID: {random.randint(10000, 99999)} -->"
                with open(info_plist, "r") as f:
                    content = f.read()

                if "<!DOCTYPE plist" in content:
                    content = content.replace(
                        "<!DOCTYPE plist", f"{random_comment}\n<!DOCTYPE plist"
                    )
                    with open(info_plist, "w") as f:
                        f.write(content)
        except:
            # Ignore errors when modifying metadata
            pass
