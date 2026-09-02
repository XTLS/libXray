"""Run: python3 build/test_build_metadata.py. No Go or platform build is run."""
import hashlib
import io
import json
from pathlib import Path
import shutil
import subprocess
import unittest
from unittest.mock import patch
from uuid import uuid4

from app.android import AndroidBuilder
from app.build import Builder


class BuildMetadataTest(unittest.TestCase):
    def setUp(self):
        # Keep every test fixture in the permitted references tree, never /tmp.
        self.root = (
            Path(__file__).resolve().parents[2]
            / "references"
            / "onexray-refactor-validation"
            / "build-metadata"
            / uuid4().hex
        )
        (self.root / "build").mkdir(parents=True)
        self.addCleanup(shutil.rmtree, self.root)
        (self.root / "go.mod").write_text("original module\n")
        (self.root / "go.sum").write_text("original sums\n")
        self.builder = AndroidBuilder(str(self.root / "build"))
        run_patch = patch("app.build.subprocess.run", side_effect=self.command)
        self.run = run_patch.start()
        self.addCleanup(run_patch.stop)

    def command(self, command, **kwargs):
        command = list(command)
        outputs = {
            ("git", "rev-parse", "HEAD"): "abc123\n",
            ("git", "status", "--porcelain", "--untracked-files=no"): " M go.mod\n",
            ("go", "version"): "go version go1.26.6 darwin/arm64\n",
            ("go", "list", "-mod=readonly", "-m", "all"): (
                "github.com/xtls/libxray\ngolang.org/x/mobile v0.0.0-resolved\n"
            ),
            (
                "go", "list", "-m", "-f", "{{.Version}}",
                "golang.org/x/mobile@latest",
            ): "v0.0.0-resolved\n",
            ("go", "version", "-m", "/fixture/gomobile"): (
                "/fixture/gomobile: go1.26.6\n"
                "\tmod\tgolang.org/x/mobile\tv0.0.0-actual\th1:fixture\n"
            ),
        }
        return subprocess.CompletedProcess(command, 0, outputs.get(tuple(command), ""), "")

    def read(self, builder="android"):
        path = self.root / "build" / f"build-metadata-{builder}.json"
        return json.loads(path.read_text())

    def test_effective_inputs_are_captured_before_restoration(self):
        self.builder.snapshot_go_env()
        (self.root / "go.mod").write_text("effective module\n")
        (self.root / "go.sum").write_text("effective sums\n")
        with patch.dict("app.build.os.environ", {"LIBXRAY_GOMOBILE_VERSION": ""}):
            self.builder.prepare_gomobile()
        with patch("app.build.shutil.which", return_value="/fixture/gomobile"):
            self.builder.after_build()
        self.builder.restore_go_env()
        metadata = self.read()
        self.assertEqual(metadata["evidence"], "build-inputs-only")
        self.assertEqual(metadata["builder"], "android")
        self.assertEqual(metadata["libXrayCommit"], "abc123")
        self.assertTrue(metadata["libXrayDirty"])
        self.assertIn("go1.26.6", metadata["goVersion"])
        self.assertIn("golang.org/x/mobile v0.0.0-resolved", metadata["modules"])
        self.assertEqual(
            metadata["goModSha256"], hashlib.sha256(b"effective module\n").hexdigest()
        )
        self.assertEqual(
            metadata["goSumSha256"], hashlib.sha256(b"effective sums\n").hexdigest()
        )
        self.assertEqual(metadata["gomobile"]["resolvedVersion"], "v0.0.0-resolved")
        self.assertEqual(metadata["gomobile"]["usedVersion"], "v0.0.0-actual")
        self.assertEqual(metadata["errors"], [])
        self.assertEqual((self.root / "go.mod").read_text(), "original module\n")
        self.assertEqual((self.root / "go.sum").read_text(), "original sums\n")
        for call in self.run.call_args_list:
            if "check" in call.kwargs:
                self.assertEqual(call.kwargs["cwd"], str(self.root))

    def test_non_gomobile_build_has_its_own_name(self):
        builder = type("AppleGoBuilder", (Builder,), {})(str(self.root / "build"))
        builder.after_build()
        metadata = self.read("apple-go")
        self.assertIsNone(metadata["gomobile"])
        self.assertEqual(metadata["errors"], [])

    def test_gomobile_version_environment_selects_resolution_query(self):
        version = "v0.0.0-20260821190718-4776eadac327"
        with (
            patch.dict("app.build.os.environ", {"LIBXRAY_GOMOBILE_VERSION": version}),
            patch(
                "app.build.subprocess.run",
                return_value=subprocess.CompletedProcess([], 0, version + "\n", ""),
            ) as run,
        ):
            self.builder.prepare_gomobile()
        self.assertEqual(run.call_args_list[0].args[0], [
            "go", "list", "-m", "-f", "{{.Version}}",
            f"golang.org/x/mobile@{version}",
        ])
        self.assertEqual(self.builder._gomobile_version, version)

    def test_collection_failure_keeps_original_build_error(self):
        with (
            patch.object(
                self.builder, "before_build",
                side_effect=RuntimeError("original build failed"),
            ),
            patch("app.build.subprocess.run", side_effect=OSError("tool unavailable")),
            patch("sys.stderr", new_callable=io.StringIO),
        ):
            with self.assertRaisesRegex(RuntimeError, "original build failed"):
                self.builder.build()
        metadata = self.read()
        self.assertIsNone(metadata["libXrayCommit"])
        self.assertIsNone(metadata["modules"])
        self.assertTrue(metadata["errors"])
        self.assertIsNone(self.builder._go_env_snapshot)

    def test_write_failure_keeps_original_build_error(self):
        blocked = self.root / "not-a-directory"
        blocked.write_text("fixture")
        self.builder.build_dir = str(blocked)
        with (
            patch.object(
                self.builder, "before_build",
                side_effect=RuntimeError("original build failed"),
            ),
            patch("sys.stderr", new_callable=io.StringIO),
        ):
            with self.assertRaisesRegex(RuntimeError, "original build failed"):
                self.builder.build()
        self.assertIsNone(self.builder._go_env_snapshot)


if __name__ == "__main__":
    unittest.main()
