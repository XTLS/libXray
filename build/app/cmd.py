import os
import shutil


def create_dir_if_not_exists(work_dir: str):
    if not os.path.exists(work_dir):
        os.makedirs(work_dir)


def delete_dir_if_exists(work_dir: str):
    if os.path.exists(work_dir):
        shutil.rmtree(work_dir)


def delete_file_if_exists(file_path: str):
    if os.path.exists(file_path):
        os.remove(file_path)
