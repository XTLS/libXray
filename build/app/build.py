import os.path
import re
import shutil
import subprocess

from app.cmd import delete_file_if_exists


class Builder(object):
    def __init__(self, build_dir: str):
        self.build_dir = build_dir
        self.lib_dir = os.path.join(self.build_dir, "..")

    def clean_lib_files(self, files: list[str]):
        for file in files:
            file_path = os.path.join(self.lib_dir, file)
            delete_file_if_exists(file_path)

    def prepare_go(self):
        clean_files = ["go.mod", "go.sum"]
        self.clean_lib_files(clean_files)
        os.chdir(self.lib_dir)
        ret = subprocess.run(["go", "mod", "init", "github.com/xtls/libxray"])
        if ret.returncode != 0:
            raise Exception("go mod init failed")
        ret = subprocess.run(["go", "mod", "tidy"])
        if ret.returncode != 0:
            raise Exception("go mod tidy failed")

    def download_geo(self):
        os.chdir(self.lib_dir)
        main_path = os.path.join("main", "main.go")
        ret = subprocess.run(["go", "run", main_path])
        if ret.returncode != 0:
            raise Exception("go mod init failed")

    def prepare_static_lib(self):
        self.copy_go_main_file()
        self.fix_package_name()

    def copy_go_main_file(self):
        src_file = os.path.join(self.build_dir, "template", "main.go")
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
                    new_line = 'package main\n\nimport "C"\n'
                new_lines.append(new_line)
        with open(file_path, "w") as f:
            f.writelines(new_lines)

    def before_build(self):
        self.prepare_go()
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
                if re.match(r'^import\s+"C"', line):
                    new_line = ""
                new_lines.append(new_line)
        with open(file_path, "w") as f:
            f.writelines(new_lines)
