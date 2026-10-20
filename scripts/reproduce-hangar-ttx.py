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


def crew_efficiency(mastery=100, crew_increase=0):
    """0x1008fe3a0: one separate commander and one tanker per other role.

    For combined roles the native code averages contributions from the crew
    members serving that role; this convenience wrapper covers Leopard 1.
    """
    level = div(add(mastery, crew_increase), 100)
    commander = add(mul(0.43, level), 0.57)
    others = add(mul(0.43, add(div(level, 10), level)), 0.57)
    return {"commander": commander, "loader": others,
            "gunner": others, "driver": others}


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
    """0x1008ffd70: raw chassis/turret rotation speeds, m/s for movement.

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
    dispersion = div(mul(div(0.3, 100), 100), crew["gunner"])
    values = {"reload": (reload, 2), "dpm": (dpm, 0),
              "full_aim": (aim, 1), "hull_rotation": (rotation, 2),
              "dispersion": (dispersion, 3)}
    for name, value in zip(("firm", "medium", "soft"), terrain):
        values["permeability_" + name] = (mul(div(1, value), 100), 0)
    return {"crew": crew, "values": {
        name: {"raw_float32": value,
               "ui": f"{ui_round(value, digits):.{digits}f}"}
        for name, (value, digits) in values.items()}}


if __name__ == "__main__":
    print(json.dumps(leopard_example(), indent=2))
