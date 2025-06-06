# main.py
import os
import sys

from app.android import AndroidBuilder
from app.apple_go import AppleGoBuilder
from app.apple_gomobile import AppleGoMobileBuilder
from app.linux import LinuxBuilder
from app.windows import WindowsBuilder


def build_dir_path():
    file_dir = os.path.dirname(__file__)
    dir_path = os.path.abspath(file_dir)
    return dir_path


if __name__ == "__main__":
    print(sys.argv)
    platform = sys.argv[1]

    if platform == "apple":
        tool = sys.argv[2]
        if tool == "go":
            builder = AppleGoBuilder(build_dir_path())
            builder.build()
        elif tool == "gomobile":
            builder = AppleGoMobileBuilder(build_dir_path())
            builder.build()
        else:
            raise Exception(f"platform {platform} tool {tool} not supported")

    elif platform == "android":
        android_api = sys.argv[2] if len(sys.argv) > 2 else "21"
        builder = AndroidBuilder(build_dir_path(), android_api)
        builder.build()

    elif platform == "linux":
        builder = LinuxBuilder(build_dir_path())
        builder.build()

    elif platform == "windows":
        builder = WindowsBuilder(build_dir_path())
        builder.build()

    else:
        raise Exception(f"platform {platform} not supported")
