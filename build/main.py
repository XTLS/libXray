import os
import sys

from app.android import AndroidBuilder
from app.apple import AppleBuilder


def build_dir_path():
    file_dir = os.path.dirname(__file__)
    dir_path = os.path.abspath(file_dir)
    return dir_path


if __name__ == "__main__":
    print(sys.argv)
    platform = sys.argv[1]
    if platform == "apple":
        builder = AppleBuilder(build_dir_path())
        builder.build()
    elif platform == "android":
        builder = AndroidBuilder(build_dir_path())
        builder.build()
    else:
        raise Exception(f"platform {platform} not supported")
