# main.py
import os
import sys

from app.android import AndroidBuilder
from app.apple_go import AppleGoBuilder
from app.apple_gomobile import AppleGoMobileBuilder
from app.linux import LinuxBuilder
from app.windows import WindowsBuilder

LOCAL_ARG = "local"
NO_SNAPSHOT_ARG = "--no-snapshot"


def build_dir_path():
    file_dir = os.path.dirname(__file__)
    dir_path = os.path.abspath(file_dir)
    return dir_path


def parse_build_args(args: list[str]) -> tuple[bool, bool]:
    if len(args) != len(set(args)) or any(
        arg not in (LOCAL_ARG, NO_SNAPSHOT_ARG) for arg in args
    ):
        raise Exception(f"unsupported args: {args}")
    return LOCAL_ARG in args, NO_SNAPSHOT_ARG not in args


if __name__ == "__main__":
    print(sys.argv)
    platform = sys.argv[1]

    if platform == "apple":
        tool = sys.argv[2]
        use_local_xray_core, keep_go_env_snapshot = parse_build_args(sys.argv[3:])
        if tool == "go":
            builder = AppleGoBuilder(
                build_dir_path(), use_local_xray_core, keep_go_env_snapshot
            )
            builder.build()
        elif tool == "gomobile":
            builder = AppleGoMobileBuilder(
                build_dir_path(), use_local_xray_core, keep_go_env_snapshot
            )
            builder.build()
        else:
            raise Exception(f"platform {platform} tool {tool} not supported")

    elif platform == "android":
        use_local_xray_core, keep_go_env_snapshot = parse_build_args(sys.argv[2:])
        builder = AndroidBuilder(
            build_dir_path(), use_local_xray_core, keep_go_env_snapshot
        )
        builder.build()

    elif platform == "linux":
        use_local_xray_core, keep_go_env_snapshot = parse_build_args(sys.argv[2:])
        builder = LinuxBuilder(
            build_dir_path(), use_local_xray_core, keep_go_env_snapshot
        )
        builder.build()

    elif platform == "windows":
        use_local_xray_core, keep_go_env_snapshot = parse_build_args(sys.argv[2:])
        builder = WindowsBuilder(
            build_dir_path(), use_local_xray_core, keep_go_env_snapshot
        )
        builder.build()

    else:
        raise Exception(f"platform {platform} not supported")
