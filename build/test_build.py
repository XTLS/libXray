"""Run: python3 build/test_build.py. No Go or platform build is run."""
from pathlib import Path
import shutil
import subprocess
import unittest
from unittest.mock import call, patch
from uuid import uuid4

from app.android import AndroidBuilder


class BuildTest(unittest.TestCase):
    def setUp(self):
        self.root = (
            Path(__file__).resolve().parents[2]
            / "references"
            / "onexray-refactor-validation"
            / "build-scripts"
            / uuid4().hex
        )
        (self.root / "build").mkdir(parents=True)
        self.addCleanup(shutil.rmtree, self.root)
        self.builder = AndroidBuilder(str(self.root / "build"))

    def test_build_restores_modules_on_success_and_failure(self):
        for fails in (False, True):
            with self.subTest(fails=fails):
                (self.root / "go.mod").write_text("original module\n")
                (self.root / "go.sum").write_text("original sums\n")

                def prepare():
                    (self.root / "go.mod").write_text("effective module\n")
                    (self.root / "go.sum").write_text("effective sums\n")
                    if fails:
                        raise RuntimeError("original build failed")

                with (
                    patch.object(self.builder, "before_build", side_effect=prepare),
                    patch("app.android.os.chdir"),
                    patch("app.android.subprocess.run", return_value=subprocess.CompletedProcess([], 0)),
                ):
                    if fails:
                        with self.assertRaisesRegex(RuntimeError, "original build failed"):
                            self.builder.build()
                    else:
                        self.builder.build()

                self.assertEqual((self.root / "go.mod").read_text(), "original module\n")
                self.assertEqual((self.root / "go.sum").read_text(), "original sums\n")
                self.assertEqual(list((self.root / "build").iterdir()), [])
                self.assertIsNone(self.builder._go_env_snapshot)

    def test_gomobile_and_gobind_use_the_same_resolved_version(self):
        version = "v0.0.0-20260821190718-4776eadac327"
        for requested in ("", version):
            with self.subTest(requested=requested), patch.dict(
                "app.build.os.environ", {"LIBXRAY_GOMOBILE_VERSION": requested}
            ), patch(
                "app.build.subprocess.run",
                return_value=subprocess.CompletedProcess([], 0, version + "\n", ""),
            ) as run:
                self.builder.prepare_gomobile()
                self.assertEqual(run.call_args_list, [
                    call(["go", "list", "-m", "-f", "{{.Version}}",
                          f"golang.org/x/mobile@{requested or 'latest'}"],
                         capture_output=True, text=True),
                    call(["go", "get", "-tool", f"golang.org/x/mobile/cmd/gobind@{version}"]),
                    call(["go", "install", f"golang.org/x/mobile/cmd/gomobile@{version}"]),
                    call(["gomobile", "init"]),
                ])


if __name__ == "__main__":
    unittest.main()
