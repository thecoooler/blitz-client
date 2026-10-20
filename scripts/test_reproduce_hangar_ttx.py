"""Binary-derived arithmetic and supplied UI controls; no live-process claim."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location(
    "hangar", Path(__file__).with_name("reproduce-hangar-ttx.py"))
h = importlib.util.module_from_spec(spec)
spec.loader.exec_module(h)


class HangarReconstructionTests(unittest.TestCase):
    def test_supplied_ui_controls(self):
        values = h.leopard_example()["values"]
        self.assertEqual(values["turret_rotation"]["ui"], "37.55")
        self.assertEqual(values["reload"]["ui"], "5.80")
        self.assertEqual(values["dpm"]["ui"], "3727")
        self.assertEqual(h.control_examples()["reload_controls"],
                         {"IS-7": "10.70", "Grille 15": "10.51"})
        # Keep the disputed raw-descriptor result visible; matching the next
        # hypothesis is NOT evidence that the game supplies that descriptor.
        self.assertEqual(values["full_aim"]["ui"], "3.9")
        aims = h.control_examples()["full_aim_HYPOTHESIS_aiming_time_percent_0"]
        self.assertEqual([row["ui"] for row in aims], ["4.1", "3.8", "4.1", "3.8"])

    def test_role_members_not_role_names_are_averaged(self):
        base = h.crew_efficiency()
        two = h.crew_efficiency(members=[("commander",), ("loader",), ("loader",)])
        mixed = h.crew_efficiency(members=[("commander", "loader"), ("loader",)])
        self.assertEqual(two["loader"], base["loader"])
        self.assertEqual(mixed["loader"], h.f32(1.0215))
        boosted = h.crew_efficiency(100, 15)
        self.assertGreater(boosted["commander"], 1)
        self.assertGreater(boosted["gunner"], boosted["commander"])

    def test_modifier_order_and_driver_resistance_are_distinct(self):
        gunner = h.crew_efficiency()["gunner"]
        # Same-key -10 and -5 => -15, not 0.9 * 0.95.
        percent = h.modifier_sum([-10, -5])
        self.assertEqual(h.engine_power(100, percent), h.f32(85))
        self.assertNotEqual(h.engine_power(100, percent), h.mul(90, 0.95))
        self.assertEqual(h.turret_rotation(36, gunner, 10),
                         h.mul(h.mul(36, h.percent_factor(10)), gunner))
        terrain = h.terrain_resistances((0.6, 0.7, 1), gunner, (0, 10, 0))
        ratio = h.div(0.7, terrain[1])
        self.assertNotEqual(ratio, gunner)
        self.assertAlmostEqual(ratio, gunner * 1.1, places=6)

    def test_multishot_cycles(self):
        # Manual cycles: 3 shots / (20+2*2), 2 / (10+0.5),
        # 4 / (20+0.5+2). Native helper scales only magazine reload.
        self.assertEqual(h.shots_per_minute(20, 1, clip_count=3, clip_interval=2), 7.5)
        self.assertEqual(h.shots_per_minute(10, 1, burst_count=2, burst_interval=0.5), h.f32(120/10.5))
        self.assertEqual(h.shots_per_minute(20, 1, burst_count=2, clip_count=4,
                                           burst_interval=0.5, clip_interval=2), h.f32(240/22.5))
        self.assertEqual(h.shots_per_minute(20, 0.5, clip_count=3, clip_interval=2), h.f32(180/14))
        self.assertEqual(h.shots_per_minute(20, 1, burst_interval=99, clip_interval=99), 3)

    def test_camo_and_speed_boundaries(self):
        still, moving, shot = h.camouflage(0.247, 0.185, 0, 0.209, skill_level=7)
        self.assertAlmostEqual(still, 0.457, places=6)
        self.assertAlmostEqual(moving, 0.395, places=6)
        self.assertAlmostEqual(shot, 0.457 * 0.209, places=6)
        self.assertEqual(h.average_speed(735500, 10000, 1, 1), 1)
        with self.assertRaises(ValueError):
            h.average_speed(0, 10000, 2, 20)
        self.assertEqual(h.average_speed(0, 10000, 2, 20, 10), h.mul(10, 0.27778))

    def test_ui_rounding_float32(self):
        self.assertEqual(h.ui_round(h.f32(1.125), 2), h.f32(1.13))
        self.assertEqual(h.ui_round(h.f32(-1.125), 2), h.f32(-1.13))
        self.assertEqual(h.ui_round(12.49, 0), 12)
        self.assertEqual(h.ui_round(12.5, 0), 13)


if __name__ == "__main__":
    unittest.main()
