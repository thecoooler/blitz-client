# Real replay fixture

`sample.tbreplay` is a copy of a local E-100 battle replay recorded on
2026-09-27 with client `26.8.0_ruby_apple`. It contains the original archive
entries, including a 2,648,215-byte `data.replay` stream. The recording includes
player and battle information from that match.

`scripts/simulate-replay.py` uses this file by default. The source copy is read
only; every run writes a separate growing `data.replay` file.
