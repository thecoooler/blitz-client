#!/usr/bin/env python3
"""Reconstruction of selected Tanks Blitz 26.10.0 hangar calculations.

Research aid, not a general vehicle calculator. See docs/hangar-ttx.md for
binary addresses, field mappings, scope, and disassembly evidence.
"""

import json
import math
import struct


def f32(value):
    return struct.unpack("<f", struct.pack("<f", value))[0]


def add(a, b):
    return f32(f32(a) + f32(b))


def mul(a, b):
    return f32(f32(a) * f32(b))


def div(a, b):
    return f32(f32(a) / f32(b))


def percent_factor(percent):
    return add(div(percent, 100), 1)


def crew_efficiency(mastery=100, crew_increase=0, members=None):
    """0x1008fe3a0: common mastery, primary role + extra roles per member.

    Members is a sequence of tuples, not a role-keyed dict: two loaders must
    remain two people. A commander's additional roles retain commander mastery.
    No mastery cap is present in this calculation. Invalid/missing roles are
    outside this reconstruction (the native role lookup asserts if absent).
    """
    if members is None:
        members = [("commander", "radioman"), ("gunner",),
                   ("driver",), ("loader",)]
    level = div(add(mastery, crew_increase), 100)
    commander = add(mul(0.43, level), 0.57)
    others = add(mul(0.43, add(div(level, 10), level)), 0.57)
    contributions = {}
    for roles in members:
        efficiency = commander if roles[0] == "commander" else others
        for role in roles:
            contributions.setdefault(role, []).append(efficiency)
    result = {}
    for role, values in contributions.items():
        total = f32(0)
        for value in values:
            total = add(total, value)
        result[role] = div(total, len(values))
    return result


def modifier_sum(values):
    """0x1008fe0c0: sum contributions to ONE resolved modifier key."""
    total = f32(0)
    for value in values:
        total = add(total, value)
    return total


def turret_rotation(base, gunner, percent=0):
    """0x1008feb6a..bd3, degrees/s in this convenience interface."""
    return mul(mul(base, percent_factor(percent)), gunner)


def engine_power(base, percent=0):
    """0x1008fed02..d4f; same units as base, no crew multiplier."""
    return mul(base, percent_factor(percent))


def stationary_dispersion(radius, gunner, percent=0):
    """0x1008ff4a3..514; radius in metres at 100 m (XML units)."""
    return div(mul(mul(div(radius, 100), 100), percent_factor(percent)), gunner)


def average_speed(power_watts, mass_kg, medium_resistance, max_speed_mps,
                  fallback_speed_kph=None):
    """0x1008fef88..ff025: empirical hangar estimate, not physics simulation.

    Coefficients read from binary, not fitted. Negative/nonpositive adjusted
    power requires the runtime model's fallback; never invent a default.
    """
    specific = div(div(power_watts, 735.5), div(mass_kg, 1000))
    resistance = div(add(1, -medium_resistance), 0.1)
    adjusted = add(mul(add(mul(specific, -0.0371), 1.3115), resistance), specific)
    if adjusted > 0:
        speed = mul(f32(math.pow(adjusted, f32(0.5225))), 7.5587)
    elif fallback_speed_kph is not None:
        speed = f32(fallback_speed_kph)
    else:
        raise ValueError("runtime fallback speed required for adjusted power <= 0")
    return min(mul(speed, 0.27778), f32(max_speed_mps))


def camouflage(still, moving, camouflage_bonus, gun_at_shot,
               detectability=0, skill_level=0, at_shot_percent=0):
    """0x1008ff7c0, ordinary descriptor branch, all results are fractions.

    camouflage_bonus must be resolved by the caller; equipped camouflage can
    select a different value through the runtime service at vtable+0x68/+0x30.
    """
    skill = mul(skill_level, 0.03)
    stationary = add(add(add(still, camouflage_bonus),
                         add(detectability, detectability)), skill)
    mobile = add(add(add(moving, camouflage_bonus), detectability), skill)
    at_shot = mul(gun_at_shot, add(div(at_shot_percent, -100), 1))
    return stationary, mobile, mul(stationary, at_shot)


def reload_time(base_reload, loader, reload_percent=0):
    """0x1008ff2f0, ordinary single-shot gun, no special reload multiplier."""
    return mul(div(percent_factor(reload_percent), loader), base_reload)


def shots_per_minute(base_reload, reload_scale, burst_count=1,
                     clip_count=1, burst_interval=0, clip_interval=0):
    """0x1010c8ff0; counts/intervals are the native gun descriptor fields."""
    burst_interval = burst_interval if burst_count >= 2 else 0
    clip_interval = clip_interval if clip_count >= 2 else 0
    clips_per_burst = div(clip_count, burst_count if clip_count >= 2 else 1)
    numerator = mul(mul(burst_count, clips_per_burst), 60)
    duration = add(mul(reload_scale, base_reload),
                   mul(burst_count - 1, burst_interval))
    duration = add(mul(add(clips_per_burst, -1), clip_interval), duration)
    return div(numerator, duration)


def ui_round(value, digits):
    """0x100fec530: sign, half-step, split integer/fraction, floor fraction.

    Kept in float32 and in the observed instruction order. The native helper
    uses signed int32 conversion, so this covers ordinary finite TTX values.
    """
    sign = -1 if value < 0 else 1
    half_step, scale = mul(0.5, sign), f32(1)
    for _ in range(digits):
        half_step = div(half_step, 10)
        scale = mul(scale, 10)
    magnitude = abs(add(value, half_step))
    integer = math.trunc(magnitude)
    fraction = add(magnitude, -integer)
    fraction = div(math.floor(mul(fraction, scale)), scale)
    return mul(add(integer, fraction), sign)


def terrain_resistances(raw, driver, terrain_percents=(0, 0, 0)):
    """0x1008ff660: returns effective firm/medium/soft resistance."""
    return tuple(div(mul(div(1, percent_factor(p)), r), driver)
                 for r, p in zip(raw, terrain_percents))


def hull_rotation(base_rotation, engine_power, reference_engine_power,
                  reference_mass, current_mass, reference_medium_resistance,
                  current_medium_resistance, rotation_percent=0,
                  driver_rotation_skill=0):
    """0x1008ffb90. Reference = first modules in the vehicle's lists.

    reference_mass includes hull + chassis + engine + fuel tank + turret + gun.
    Driver mastery is already included in current_medium_resistance.
    """
    rotation = mul(base_rotation, percent_factor(rotation_percent))
    result = mul(div(engine_power, reference_engine_power), rotation)
    result = mul(result, div(reference_mass, current_mass))
    result = mul(div(reference_medium_resistance,
                     current_medium_resistance), result)
    return mul(result, add(mul(driver_rotation_skill, 0.01), 1))


def full_aim_time(base_aim, gunner, chassis_rotation, rotation_dispersion,
                  turret_rotation, turret_dispersion, max_speed_mps,
                  movement_dispersion, additive_dispersion_percent=0,
                  aiming_time_percent=0):
    """0x1008ffd70: descriptor chassis/turret speeds, m/s for movement.

    This function does not establish how the runtime descriptor was prepared.
    Passing driver/virtuoso-adjusted chassis rotation is a separate hypothesis.

    The logarithm is evaluated as float32(log(float32_argument)); libc logf
    can differ at the final bit across platforms. No claim of full emulation.
    """
    coefficient = div(-base_aim, gunner)
    coefficient = div(coefficient, add(div(aiming_time_percent, -100), 1))
    chassis = mul(rotation_dispersion, chassis_rotation)
    turret = mul(turret_dispersion, turret_rotation)
    movement = mul(movement_dispersion, min(f32(8.3334), max_speed_mps))
    squares = add(mul(turret, turret), mul(chassis, chassis))
    squares = add(mul(movement, movement), squares)
    factor = percent_factor(additive_dispersion_percent)
    total = add(mul(mul(factor, factor), squares), 1)
    inverse = div(1, f32(math.sqrt(total)))
    return mul(f32(math.log(inverse)), coefficient)


def leopard_example():
    crew = crew_efficiency()
    scale = div(percent_factor(-7), crew["loader"])
    reload = reload_time(6.5, crew["loader"], -7)
    dpm = mul(360, shots_per_minute(6.5, scale))
    terrain = terrain_resistances((0.6, 0.7, 1), crew["driver"])
    # Leopard has one option for each module: reference/current mass and
    # engine power cancel when no other modifiers are applied.
    rotation = hull_rotation(54, 1, 1, 1, 1, 0.7, terrain[1])
    aim = full_aim_time(2.3, crew["gunner"], 54, 0.1, 36, 0.05,
                        div(65, 3.6), 0.1)
    dispersion = stationary_dispersion(0.3, crew["gunner"])
    values = {"reload": (reload, 2), "dpm": (dpm, 0),
              "full_aim": (aim, 1), "hull_rotation": (rotation, 2),
              "turret_rotation": (turret_rotation(36, crew["gunner"]), 2),
              "dispersion": (dispersion, 3)}
    for name, value in zip(("firm", "medium", "soft"), terrain):
        values["permeability_" + name] = (mul(div(1, value), 100), 0)
    return {"crew": crew, "values": {
        name: {"raw_float32": value,
               "ui": f"{ui_round(value, digits):.{digits}f}"}
        for name, (value, digits) in values.items()}}


def control_examples():
    crew = crew_efficiency()
    # The first member of Grille 15 is commander + loader.
    grille = crew_efficiency(members=[("commander", "loader"), ("gunner",),
                                      ("driver",), ("radioman",)])
    reloads = {name: f"{ui_round(reload_time(base, loader, -7), 2):.2f}"
               for name, base, loader in [("IS-7", 12, crew["loader"]),
                                          ("Grille 15", 11.3, grille["loader"])]}
    hypothesis_rotation = mul(mul(54, crew["driver"]), add(mul(7, 0.01), 1))
    aim = []
    for mobility in ("recycledEngineSpeed", "improvedControls"):
        for stabilization in (0, -15):
            args = (2.3, crew["gunner"], hypothesis_rotation, 0.1,
                    36, 0.05, div(65, 3.6), 0.1, stabilization)
            value = full_aim_time(*args)
            aim.append({"mobility": mobility, "stabilizer": stabilization == -15,
                        "raw_float32": value, "ui": f"{ui_round(value, 1):.1f}"})
    # Synthetic descriptors isolate the native cycle arithmetic; these are
    # not claims that a particular vehicle has this equipment or gun.
    multi = {"clip": shots_per_minute(20, 1, clip_count=3, clip_interval=2),
             "burst": shots_per_minute(10, 1, burst_count=2, burst_interval=0.5),
             "clip_and_burst": shots_per_minute(20, 1, burst_count=2,
                                                clip_count=4, burst_interval=0.5,
                                                clip_interval=2)}
    return {"reload_controls": reloads,
            "full_aim_HYPOTHESIS_aiming_time_percent_0": aim,
            "synthetic_multishot_rpm": multi}


if __name__ == "__main__":
    print(json.dumps({"leopard_isolated_modifiers": leopard_example(),
                      **control_examples()}, indent=2))
