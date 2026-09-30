"""Run: python3 build/test_build.py. No Go or platform build is run."""
from pathlib import Path
import runpy
import shutil
import subprocess
import unittest
from unittest.mock import call, patch
from uuid import uuid4

from app.android import AndroidBuilder
from app.apple_go import AppleGoBuilder
from app.apple_gomobile import AppleGoMobileBuilder
from app.build import LIBXRAY_MOD_NAME, XRAY_CORE_MOD_NAME, DEFAULT_XRAY_CORE_VERSION
from app.linux import LinuxBuilder
from app.windows import WindowsBuilder
from main import parse_build_args


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

    def test_no_snapshot_retains_modules_on_success_and_failure(self):
        for exists in (False, True):
            for fails in (False, True):
                with self.subTest(exists=exists, fails=fails):
                    self.builder = AndroidBuilder(
                        str(self.root / "build"), keep_go_env_snapshot=False
                    )
                    for name in ("go.mod", "go.sum"):
                        path = self.root / name
                        if exists:
                            path.write_text("original\n")
                        else:
                            path.unlink(missing_ok=True)

                    def prepare():
                        self.assertIsNone(self.builder._go_env_snapshot)
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

                    self.assertEqual((self.root / "go.mod").read_text(), "effective module\n")
                    self.assertEqual((self.root / "go.sum").read_text(), "effective sums\n")
                    self.assertIsNone(self.builder._go_env_snapshot)

    def test_build_args(self):
        for args, expected in (
            ([], (False, True)),
            (["local"], (True, True)),
            (["--no-snapshot"], (False, False)),
            (["local", "--no-snapshot"], (True, False)),
            (["--no-snapshot", "local"], (True, False)),
        ):
            with self.subTest(args=args):
                self.assertEqual(parse_build_args(args), expected)
        for args in (
            ["unknown"], ["local", "local"], ["--no-snapshot", "--no-snapshot"],
        ):
            with self.subTest(args=args):
                with self.assertRaisesRegex(Exception, "unsupported args"):
                    parse_build_args(args)

    def test_no_snapshot_reinitializes_modules_before_dependency_commands(self):
        for use_local in (False, True):
            with self.subTest(use_local=use_local):
                (self.root / "go.mod").write_text("old module\n")
                (self.root / "go.sum").write_text("old sums\n")
                builder = AndroidBuilder(str(self.root / "build"), use_local, False)

                def run(args):
                    if args == ["go", "mod", "init", LIBXRAY_MOD_NAME]:
                        self.assertFalse((self.root / "go.mod").exists())
                        self.assertFalse((self.root / "go.sum").exists())
                        (self.root / "go.mod").write_text("new module\n")
                    else:
                        self.assertEqual((self.root / "go.mod").read_text(), "new module\n")
                    return subprocess.CompletedProcess(args, 0)

                with patch("app.build.os.chdir"), patch(
                    "app.build.subprocess.run", side_effect=run
                ) as commands:
                    builder.init_go_env()
                expected = [call(["go", "mod", "init", LIBXRAY_MOD_NAME])]
                if use_local:
                    expected.append(call([
                        "go", "mod", "edit", f"-replace={XRAY_CORE_MOD_NAME}=../Xray-core",
                    ]))
                else:
                    expected.extend([
                        call(["go", "mod", "edit", f"-dropreplace={XRAY_CORE_MOD_NAME}"]),
                        call(["go", "get", f"{XRAY_CORE_MOD_NAME}@{DEFAULT_XRAY_CORE_VERSION}"]),
                    ])
                expected.append(call(["go", "mod", "tidy"]))
                self.assertEqual(commands.call_args_list, expected)

    def test_default_initialization_keeps_existing_modules(self):
        (self.root / "go.mod").write_text("original module\n")
        (self.root / "go.sum").write_text("original sums\n")
        with patch("app.build.os.chdir"), patch(
            "app.build.subprocess.run", return_value=subprocess.CompletedProcess([], 0)
        ) as commands:
            self.builder.init_go_env()
        self.assertEqual((self.root / "go.mod").read_text(), "original module\n")
        self.assertEqual((self.root / "go.sum").read_text(), "original sums\n")
        self.assertNotIn(call(["go", "mod", "init", LIBXRAY_MOD_NAME]), commands.call_args_list)

    def test_all_builders_accept_snapshot_option(self):
        for builder_type in (
            AndroidBuilder, AppleGoBuilder, AppleGoMobileBuilder,
            LinuxBuilder, WindowsBuilder,
        ):
            for keep_snapshot in (False, True):
                with self.subTest(builder=builder_type.__name__, keep_snapshot=keep_snapshot):
                    builder = builder_type(
                        str(self.root / "build"), True, keep_snapshot
                    )
                    self.assertTrue(builder.use_local_xray_core)
                    self.assertEqual(builder.keep_go_env_snapshot, keep_snapshot)

    def test_cli_passes_options_to_every_builder(self):
        script = Path(__file__).with_name("main.py")
        for target, builder_path in (
            (["android"], "app.android.AndroidBuilder"),
            (["apple", "go"], "app.apple_go.AppleGoBuilder"),
            (["apple", "gomobile"], "app.apple_gomobile.AppleGoMobileBuilder"),
            (["linux"], "app.linux.LinuxBuilder"),
            (["windows"], "app.windows.WindowsBuilder"),
        ):
            for options in ([], ["local"], ["--no-snapshot"], ["local", "--no-snapshot"], ["--no-snapshot", "local"]):
                with self.subTest(target=target, options=options), patch(
                    builder_path
                ) as builder, patch(
                    "sys.argv", [str(script), *target, *options]
                ), patch("builtins.print"):
                    runpy.run_path(str(script), run_name="__main__")
                    builder.assert_called_once_with(
                        str(script.parent), *parse_build_args(options)
                    )
                    builder.return_value.build.assert_called_once_with()

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
