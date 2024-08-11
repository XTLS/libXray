import os.path
import shutil
import subprocess

from app.build import Builder
from app.cmd import create_dir_if_not_exists, delete_dir_if_exists


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
        self.module_map_file = "module.modulemap"

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
        self.build_targets(self.ios_targets)
        self.merge_static_lib(
            self.ios_targets[1].sdk,
            [self.ios_targets[1].apple_arch, self.ios_targets[2].apple_arch],
        )
        self.build_targets(self.macos_targets)
        self.merge_static_lib(
            self.macos_targets[0].sdk,
            [self.macos_targets[0].apple_arch, self.macos_targets[1].apple_arch],
        )
        self.build_targets(self.tvos_targets)
        self.merge_static_lib(
            self.tvos_targets[1].sdk,
            [self.tvos_targets[1].apple_arch, self.tvos_targets[2].apple_arch],
        )
        self.after_build()
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
        header_dir = os.path.join(output_dir, "Headers")
        create_dir_if_not_exists(header_dir)
        header_file = os.path.join(header_dir, self.lib_header_file)
        sdk_path = self.get_sdk_dir_path(sdk)
        min_version_flag = f"-m{sdk}-version-min={min_version}"
        flags = f"-isysroot {sdk_path} {min_version_flag} -arch {apple_arch}"
        run_env = os.environ.copy()
        run_env.update({
            "GOOS": platform,
            "GOARCH": go_arch,
            "GOFLAGS": f"-tags={platform}",
            "CC": f"xcrun --sdk {sdk} --toolchain {sdk} clang",
            "CXX": f"xcrun --sdk {sdk} --toolchain {sdk} clang++",
            "CGO_CFLAGS": flags,
            "CGO_CXXFLAGS": flags,
            "CGO_LDFLAGS": f"${flags} -Wl,-Bsymbolic-functions",
            "CGO_ENABLED": "1",
            "DARWIN_SDK": sdk
        })

        cmd = [
            "go",
            "build",
            "-ldflags=-w",
            f"-o={output_file}",
            "-buildmode=c-archive",
        ]
        os.chdir(self.lib_dir)
        ret = subprocess.run(cmd, env=run_env)
        if ret.returncode != 0:
            raise Exception(f"run_build_cmd for {platform} {apple_arch} {sdk} failed")
        
        generated_header = os.path.join(output_dir, self.lib_header_file)
        if os.path.exists(generated_header):
            shutil.move(generated_header, header_file)
        else:
            raise Exception(f"Generated header file {generated_header} not found")

    def get_sdk_dir_path(self, sdk: str) -> str:
        cmd = ["xcrun", "--sdk", sdk, "--show-sdk-path"]
        ret = subprocess.run(cmd, capture_output=True)
        if ret.returncode != 0:
            raise Exception(f"get_sdk_dir_path for {sdk} failed")
        return ret.stdout.decode().strip()

    def merge_static_lib(self, sdk: str, arches: list[str]):
        cmd = ["lipo", "-create"]
        for arch in arches:
            lib_dir = os.path.join(self.framework_dir, f"{sdk}-{arch}")
            lib_file = os.path.join(lib_dir, self.lib_file)
            cmd.extend(["-arch", arch, lib_file])
        arch = "-".join(arches)
        output_dir = os.path.join(self.framework_dir, f"{sdk}-{arch}")
        create_dir_if_not_exists(output_dir)
        output_file = os.path.join(output_dir, self.lib_file)
        cmd.extend(["-output", output_file])
        ret = subprocess.run(cmd)
        if ret.returncode != 0:
            raise Exception(f"merge_static_lib for {sdk} failed")

        header_dir = os.path.join(output_dir, "Headers")
        create_dir_if_not_exists(header_dir)
        source_header = os.path.join(self.framework_dir, f"{sdk}-{arches[0]}", "Headers", self.lib_header_file)
        target_header = os.path.join(header_dir, self.lib_header_file)
        shutil.copy(source_header, target_header)

    def create_framework(self):
        libs = [
            f"{self.ios_targets[0].sdk}-{self.ios_targets[0].apple_arch}",
            f"{self.ios_targets[1].sdk}-{self.ios_targets[1].apple_arch}-{self.ios_targets[2].apple_arch}",
            f"{self.macos_targets[0].sdk}-{self.macos_targets[0].apple_arch}-{self.macos_targets[1].apple_arch}",
            f"{self.tvos_targets[0].sdk}-{self.tvos_targets[0].apple_arch}",
            f"{self.tvos_targets[1].sdk}-{self.tvos_targets[1].apple_arch}-{self.tvos_targets[2].apple_arch}",
        ]
        cmd = ["xcodebuild", "-create-xcframework"]
        
        for lib in libs:
            lib_path = os.path.join(self.framework_dir, lib, self.lib_file)
            header_dir = os.path.join(self.framework_dir, lib, "Headers")
            header_file = os.path.join(header_dir, self.lib_header_file)
            framework_name = "LibXray.framework"
            temp_framework_dir = os.path.join(self.framework_dir, lib, framework_name)
            
            if not os.path.exists(lib_path) or not os.path.exists(header_file):
                raise Exception(f"Library or header file not found for {lib}")
            
            create_dir_if_not_exists(temp_framework_dir)
            temp_headers_dir = os.path.join(temp_framework_dir, "Headers")
            create_dir_if_not_exists(temp_headers_dir)
            create_dir_if_not_exists(os.path.join(temp_framework_dir, "Modules"))
            
            shutil.copy(lib_path, os.path.join(temp_framework_dir, "LibXray"))
            shutil.copy(header_file, os.path.join(temp_headers_dir, self.lib_header_file))
            
            module_map_content = """
framework module LibXray {
    umbrella header "libXray.h"
    export *
    module * { export * }
}
"""
            with open(os.path.join(temp_framework_dir, "Modules", "module.modulemap"), 'w') as f:
                f.write(module_map_content)
            
            cmd.extend(["-framework", temp_framework_dir])

        output_xcframework = os.path.join(self.lib_dir, "LibXray.xcframework")
        cmd.extend(["-output", output_xcframework])

        ret = subprocess.run(cmd)
        if ret.returncode != 0:
            raise Exception(f"create_framework failed")

    def after_build(self):
        super().after_build()
        self.reset_files()