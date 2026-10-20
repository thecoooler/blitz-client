# Расчёт ТТХ ангара: Tanks Blitz 26.10.0

Для Leopard 1 цепочка подтверждена машинным кодом установленного клиента:

```text
экипаж 100%, амуниция отсутствует
нормализованный уровень = 100 / 100 = 1
эффективность командира = 0.57 + 0.43 × 1 = 1
эффективность заряжающего = 0.57 + 0.43 × (1 + 1 / 10) = 1.043
досылатель: множитель = 1 + (-7) / 100 = 0.93
перезарядка = (0.93 / 1.043) × 6.5 = 5.7957813998… с
округление до 2 знаков → 5.80 с

ДПМ = 360 × (60 / неокруглённая_перезарядка)
    = 3726.848635…
округление до 0 знаков → 3727
```

Это не `ceil(3726.34)`: исходное неокруглённое значение другое. Константы
`0.875` / `0.00375` не используются в найденном расчёте мастерства этой версии.
Из XML прочитаны `reloadTime=6.5` и урон базового снаряда `360`.

## Что сохранено

- [Исполняемая реконструкция функций и помощников](../scripts/reproduce-hangar-ttx.py).
- [Выдержки дизассемблера с адресами](hangar-ttx-disassembly.txt).

Реконструкция написана по инструкциям x86_64; её имена функций выбраны для
читаемости, это не восстановленный оригинальный C++-исходник. В бинарнике
сохранились строки `TankParams::GetDetailParamsConfig`,
`TankParams::BaseHangarTankParamsModel` и путь
`Classes/Lobby/Common/ModulesConfiguration/ModulesConfiguration.cpp`, но
символы большинства функций удалены. Адреса ниже — preferred virtual address,
без ASLR slide.

Исследован `/Applications/Tanks Blitz.app/Contents/MacOS/Tanks Blitz`,
`CFBundleShortVersionString=26.10.0`, SHA-256:

```text
94de0345b00322fc9036105f477075f867eefa3d7cdddefe22423822c9fd69e2
```

Распакованные ресурсы в `../repo/Data` сверены побайтово с установленными DVPL:
`Leopard1.xml`, немецкие `components/shells.xml`, `optional_devices.xml`,
`hangarParamsConfig.yaml`, `VehicleParamsPanel/Params/NumberValue.yaml` совпадают.
Расчёты не запускались внутри процесса игры: численная проверка выполнена
отдельной реконструкцией. Состояние аккаунта и экипажные навыки не считывались.

## Цепочка от XML до UI

| Адрес | Назначение / подтверждённая операция |
| --- | --- |
| `0x1010c72e0` | Парсер орудия; ветка `0x1010c7f51` читает `reloadTime` в `gun+0x1c4` |
| `0x1008fddf0` | Сборка модификаторов → эффективность экипажа → расчёт параметров |
| `0x1008fe3a0` | Эффективность экипажа, командирский бонус, усреднение совмещённых ролей |
| `0x1008fea10` | Сборка `ModulesConfigurationData`; вызывает расчёты ниже |
| `0x1008ff2f0` | Перезарядка, скорострельность, разброс; reload записывается в `data+0xac` |
| `0x1010c8ff0` | Помощник скорострельности с учётом барабана/очереди |
| `0x100900380` | Урон и ДПМ; обычный ДПМ записывается в `data+0xcc` |
| `0x1008ac1c0` | `GetDetailParamsConfig`: связывает строковые ключи с getter-лямбдами |
| `0x1008d1d40` | Getter `gunShellReloadTime`: возвращает `data+0xac` |
| `0x1008d1590` | Getter `gunCommonDamagePerMin`: возвращает `data+0xcc` |
| `0x1008a84e0` | Обновление UI-модели: getter → перевод единиц → округление → `param.value` |
| `0x101896960` | Перевод единиц через таблицу коэффициентов |
| `0x100fec530` | Округление по `param.digits` |

В функции обновления UI существенные адреса:

```text
0x1008a85fd  вызов getter через vtable+0x30
0x1008a865b  перевод единиц, call 0x101896960
0x1008a8660  чтение param.digits из param+0x3c
0x1008a8665  округление, call 0x100fec530
0x1008a86c5  запись округлённого значения в param.value (param+0x30)
```

Затем ресурс
`UI/Screens3/Lobby/Common/VehicleParamsPanel/Params/NumberValue.yaml:17`
вызывает `str1000Separated(param.value, param.digits, false)`, а строка 85
передаёт полученную строку в `UITextComponent.text`.

Настройки `hangarParamsConfig.yaml`: reload — 2 знака, full aim — 1,
chassiRotationSpeed — 2, dispersion — 3. ДПМ и проходимость без переопределения
`digits`, то есть 0. В UI-модели значение уже округлено до форматирования строки.

## Экипаж, константы и досылатель

Для отдельных членов экипажа с одинаковым мастерством:

```python
level = (mastery + crewLevelIncrease) / 100
commander = 0.57 + 0.43 * level
other_role = 0.57 + 0.43 * (level + level / 10)
```

Константы лежат по адресам `0x1048dd108` (`float32(0.43)`),
`0x1048dd10c` (`float32(0.57)`), `0x10481ac68` (`10.0`),
`0x10481d228` (`100.0`). В `0x1008fe62c` видны деление `level / 10`,
сложение с `level`, умножение на `0.43`, прибавление `0.57`.

Это формула для конфигурации Leopard 1: отдельные commander, gunner, driver,
loader; командир дополнительно выполняет роль radioman. Для совмещённых ролей
в `0x1008fe980` код делит сумму вкладов на количество членов экипажа,
выполняющих эту роль. Простую формулу `other_role` нельзя безусловно применять
ко всем схемам экипажа.

`XML/item_defs/vehicles/common/optional_devices.xml:8` задаёт досылатель как
`StaticFactorDevice`, атрибут `miscAttrs/gunReloadTimeFactor`, `percent=-7`.
Расчёт `0x1008ff2f0` берёт этот процент из карты модификаторов и эффективность
роли `loader` из карты экипажа:

```python
reload_scale = (1 + reload_percent / 100) / loader_efficiency
reload = reload_scale * gun.reloadTime
```

У обычного орудия без особой механики это вся цепочка reload. В той же функции
есть дополнительные ветки для специальных множителей и нескольких времён
перезарядки; они сохранены в выдержке дизассемблера.

## Скорострельность, ДПМ и округление

Помощник `0x1010c8ff0` принимает descriptor орудия и `reload_scale`. Для
обычного орудия с одним выстрелом он сводится к:

```python
shots_per_minute = 60 / (reload_scale * gun.reloadTime)
dpm = shell.damage * shots_per_minute
```

ДПМ рассчитывается заново из descriptor и коэффициента в `0x100900380`,
а не из уже округлённого UI-reload. Ветви для очереди/барабана учитывают
количество снарядов и интервалы; помощник воспроизведён в Python.

Помощник округления `0x100fec530` добавляет половину шага со знаком исходного
числа. Затем отделяет целую часть от дробной, округляет дробную часть вниз
после масштабирования и собирает число обратно. Для обычных положительных
ТТХ это округление к ближайшему с половинами вверх, а не постоянный `ceil`.
Порядок операций с `float32` сохранён в `ui_round` реконструкции.

Числа при воспроизведении порядка SSE-операций:

```text
loader = 1.0429999828338623
reload = 5.79578161239624       → 5.80
dpm    = 3726.848388671875      → 3727
```

Последние биты отличаются от формул в начале документа, вычисленных в double.
На показанные значения это не влияет. Подмена констант или промежуточное
округление при расчёте ДПМ не нужны.

## Полное время сведения

Функция `0x1008ffd70`, результат `data+0xbc`; getter `0x1008d1010`
связан с `gunAimingTime`. Сокращённая эквивалентная формула:

```python
rotation_bloom = chassis.vehicleRotation * chassis.rotationSpeed
turret_bloom = gun.turretRotation * turret.rotationSpeed
movement_bloom = chassis.vehicleMovement * min(maxFrontSpeed_mps, 8.3334)
stabilization = 1 + additiveShotDispersionFactor_percent / 100
bloom = sqrt(1 + stabilization**2 * (
    rotation_bloom**2 + turret_bloom**2 + movement_bloom**2))
aim_speed_factor = 1 + gunAimingTimeFactor_percent / -100
full_aim = (gun.aimingTime / gunner_efficiency / aim_speed_factor) * log(bloom)
```

Машинный код использует эквивалентные `logf(1 / bloom)` и отрицательное
`-aimingTime`. Скорости вращения берутся из **исходных descriptor** ходовой и
башни, а не из итоговых `chassiRotationSpeed` / `turretRotationSpeed` ангара.
Скорость движения — м/с. Константа ограничения — именно `float32(8.3334)`
(`0x1048dd134`), не точное `30 / 3.6`.

Для Leopard 1 без прочих модификаторов: `54 × 0.1`, `36 × 0.05`,
`min(65 / 3.6, 8.3334) × 0.1`, `aimingTime=2.3`, gunner `1.043`:
`full_aim ≈ 3.891184` → **3.9 с**.

В `0x1008ff2f0` также рассчитывается стационарный разброс:
`shotDispersionRadius × (1 + shotDispersionAngleFactor_percent / 100) / gunner`.
Парсер сначала делит XML-radius на 100 (`0x1010c7f6f`), расчёт умножает
его на 100 обратно. Без других бонусов: `0.3 / 1.043 ≈ 0.287632` → **0.288**.

## Проходимость

Функция `0x1008ff660` сначала считает сопротивления грунтов:

```python
effective_resistance[i] = (
    chassis.terrainResistance[i]
    / (1 + terrain_bonus_percent[i] / 100)
    / driver_efficiency
)
permeability_percent[i] = 100 / effective_resistance[i]
```

Эффективные firm/medium/soft resistance записываются в `data+0x28/0x2c/0x30`.
Проходимость до преобразования в проценты — в `data+0x34/0x38/0x3c`.
`terrainPermeabilityFirm/Medium/Soft` используют последние поля.

Для Leopard 1, `terrainResistance = 0.6 0.7 1`, driver `1.043`, без прочих
бонусов: **174% / 149% / 104%** после округления.

## Поворот корпуса

Функция `0x1008ffb90`, результат `data+0x94`, ключ `chassiRotationSpeed`:

```python
rotation = chassis.rotationSpeed * (1 + rotationSpeedFactor_percent / 100)
hull_rotation = (
    rotation
    * current_engine_power / reference_engine_power
    * reference_mass / current_mass
    * reference_medium_resistance / current_effective_medium_resistance
    * (1 + driver_rotation_skill * 0.01)
)
```

Здесь `reference` — первые элементы списков модулей машины. Код суммирует
массу корпуса, ходовой, двигателя, бака, башни и орудия этой конфигурации.
Это следует из обращений к спискам модулей `0x1008ffbb6`–`0x1008ffbdc`
и суммирования `0x1008ffcfa`–`0x1008ffd12`.
Мастерство driver уже входит в effective resistance: повторно умножать на
его эффективность не нужно. Последний множитель связан с отдельным навыком
`virtuoso` (ключ по адресу `0x104971a20`), получаемым из данных аккаунта;
при отсутствии этого вклада он равен 1.

У Leopard 1 в XML по одному варианту модулей, поэтому без других модификаторов
отношения мощности и массы равны 1, а отношение сопротивлений равно `1.043`:
`54 × 1.043 = 56.322` → **56.32°/с**.

## Воспроизведение и границы

```sh
python3 scripts/reproduce-hangar-ttx.py
```

Скрипт выводит исходные float32 и строки UI для reload, ДПМ, полного сведения,
поворота, разброса и трёх грунтов. Пример предполагает досылатель -7%,
100% экипаж, отсутствие амуниции, остальных модификаторов и вкладов отдельных
навыков. Для полного сведения `logf` приближён через округлённый в float32
`math.log`; побитовая эмуляция системной libm не заявляется.

Это исследовательская реконструкция конкретной версии клиента, не подключённый
к экспорту универсальный калькулятор. При другой версии бинарника нужно заново
проверять адреса, константы и расположение полей.
