# Real replay fixture

`sample.tbreplay` is a copy of the newest local replay at the time of replacement:
`20261001_1455__iCooLER_G121_Grille_15_L63_14044716647467574.tbreplay`.

The battle was recorded on 2026-10-01 by iCooLER in G121_Grille_15_L63,
with client `26.10.0_ruby_apple`. The archive contains a 3,435,929-byte
`data.replay` stream and the original player and battle information.

`scripts/simulate-replay.py` uses this file by default. The source copy is read
only; every run writes a separate growing `data.replay` file.
