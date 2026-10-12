"""Run with: python3 -B -m unittest discover -s scripts -p test_simulate_replay.py."""

from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
import zipfile


ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "scripts/simulate-replay.py"


class SimulateReplayTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        with zipfile.ZipFile(ROOT / "testdata/sample.tbreplay") as archive:
            cls.data = archive.read("data.replay")

    def run_script(self, *args, env=None):
        return subprocess.run(
            [sys.executable, str(SCRIPT), *map(str, args),
             "--start-delay", "0", "--interval-ms", "0"],
            capture_output=True, text=True, timeout=10, env=env,
        )

    def test_recording_grows_and_preserves_real_replay_bytes(self):
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp) / "recording_TestPlayer.tbreplay/data.replay"
            with subprocess.Popen(
                [sys.executable, str(SCRIPT), "--replay-dir", temp,
                 "--nickname", "TestPlayer", "--start-delay", "0.2",
                 "--chunk-size", "4096", "--interval-ms", "1"],
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,
            ) as process:
                try:
                    for _ in range(4):
                        self.assertTrue(process.stdout.readline())
                    self.assertEqual(output.stat().st_size, 0)
                    deadline = time.monotonic() + 5
                    while output.stat().st_size == 0 and time.monotonic() < deadline:
                        time.sleep(0.01)
                    size = output.stat().st_size
                    self.assertGreater(size, 0)
                    self.assertLess(size, len(self.data))
                    _, stderr = process.communicate(timeout=10)
                    self.assertEqual(process.returncode, 0, stderr)
                finally:
                    if process.poll() is None:
                        process.kill()
                        process.communicate()
            self.assertEqual(output.read_bytes(), self.data)
            self.assertEqual([p.name for p in output.parent.iterdir()], ["data.replay"])

    def test_existing_recording_directory_is_never_reused(self):
        with tempfile.TemporaryDirectory() as temp:
            recording = Path(temp) / "recording_ReplaySimulator.tbreplay"
            recording.mkdir()
            for existing_data in [False, True]:
                if existing_data:
                    (recording / "data.replay").write_bytes(b"real battle")
                result = self.run_script("--replay-dir", temp)
                self.assertNotEqual(result.returncode, 0)
                if existing_data:
                    self.assertEqual((recording / "data.replay").read_bytes(), b"real battle")
                else:
                    self.assertEqual(list(recording.iterdir()), [])

    def test_output_mode_still_copies_and_rejects_overwriting(self):
        with tempfile.TemporaryDirectory() as temp:
            output = Path(temp) / "data.replay"
            self.assertEqual(self.run_script("--output", output).returncode, 0)
            self.assertEqual(output.read_bytes(), self.data)
            self.assertNotEqual(self.run_script("--output", output).returncode, 0)
            self.assertEqual(output.read_bytes(), self.data)

    def test_invalid_nickname_and_conflicting_destinations_are_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            for args in [
                ("--replay-dir", temp, "--nickname", "../escape"),
                ("--replay-dir", temp, "--nickname", "bad\\name"),
                ("--nickname", "TestPlayer"),
                ("--replay-dir", temp, "--game-replays"),
                ("--replay-dir", temp, "--output", Path(temp) / "data.replay"),
            ]:
                self.assertNotEqual(self.run_script(*args).returncode, 0)
            self.assertEqual(list(Path(temp).iterdir()), [])

    @unittest.skipUnless(sys.platform == "darwin", "macOS default path")
    def test_game_replays_uses_standard_os_folder(self):
        import os
        with tempfile.TemporaryDirectory() as temp:
            result = self.run_script("--game-replays", env={**os.environ, "HOME": temp})
            self.assertEqual(result.returncode, 0, result.stderr)
            output = (Path(temp) / "Library/Containers/com.tanksblitz/Data/Documents"
                      / "tb-project/replays/recording_ReplaySimulator.tbreplay/data.replay")
            self.assertEqual(output.read_bytes(), self.data)


if __name__ == "__main__":
    unittest.main()
