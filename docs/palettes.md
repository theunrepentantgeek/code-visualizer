# Palettes

`codeviz` includes several colour palettes for visualising metrics.
Each palette can be applied to fill or border colours using the `--fill-palette` and `--border-palette` flags.

Ordered palettes map low-to-high metric values across a colour gradient.
The categorization palette assigns visually distinct colours to discrete categories.

## Categorization

**Name:** `categorization` · **Steps:** 12 · **Ordered:** No

Twelve visually distinct colours for categorical data such as file type.
Colours are drawn from the ColorBrewer *Paired* scheme.

![Categorization palette swatch](palette-categorization.png)

| Step | Hex | RGB |
|------|-----|-----|
| 1 | `#A6CEE3` | 166, 206, 227 |
| 2 | `#1F78B4` | 31, 120, 180 |
| 3 | `#B2DF8A` | 178, 223, 138 |
| 4 | `#33A02C` | 51, 160, 44 |
| 5 | `#FB9A99` | 251, 154, 153 |
| 6 | `#E31A1C` | 227, 26, 28 |
| 7 | `#FDBF6F` | 253, 191, 111 |
| 8 | `#FF7F00` | 255, 127, 0 |
| 9 | `#CAB2D6` | 202, 178, 214 |
| 10 | `#6A3D9A` | 106, 61, 154 |
| 11 | `#FFFF99` | 255, 255, 153 |
| 12 | `#B15928` | 177, 89, 40 |

## Temperature

**Name:** `temperature` · **Steps:** 11 · **Ordered:** Yes

A diverging palette from dark blue through white to bright red,
based on the ColorBrewer *RdBu* scheme.
Suitable for metrics where a neutral midpoint is meaningful.

![Temperature palette swatch](palette-temperature.png)

| Step | Hex | RGB |
|------|-----|-----|
| 1 | `#053061` | 5, 48, 97 |
| 2 | `#2166AC` | 33, 102, 172 |
| 3 | `#4393C3` | 67, 147, 195 |
| 4 | `#92C5DE` | 146, 197, 222 |
| 5 | `#D1E5F0` | 209, 229, 240 |
| 6 | `#F7F7F7` | 247, 247, 247 |
| 7 | `#FDDBC7` | 253, 219, 199 |
| 8 | `#F4A582` | 244, 165, 130 |
| 9 | `#D6604D` | 214, 96, 77 |
| 10 | `#B2182B` | 178, 24, 43 |
| 11 | `#67001F` | 103, 0, 31 |

## Good/Bad

**Name:** `good-bad` · **Steps:** 13 · **Ordered:** Yes

A diverging palette from red through yellow to green,
based on the ColorBrewer *RdYlGn* scheme.
Use when low values are undesirable and high values are desirable (or vice versa).

![Good/Bad palette swatch](palette-good-bad.png)

| Step | Hex | RGB |
|------|-----|-----|
| 1 | `#A50026` | 165, 0, 38 |
| 2 | `#D73027` | 215, 48, 39 |
| 3 | `#F46D43` | 244, 109, 67 |
| 4 | `#FDAE61` | 253, 174, 97 |
| 5 | `#FEE08B` | 254, 224, 139 |
| 6 | `#FFFFBF` | 255, 255, 191 |
| 7 | `#FFFFFF` | 255, 255, 255 |
| 8 | `#D9EF8B` | 217, 239, 139 |
| 9 | `#A6D96A` | 166, 217, 106 |
| 10 | `#66BD63` | 102, 189, 99 |
| 11 | `#1A9850` | 26, 152, 80 |
| 12 | `#006837` | 0, 104, 55 |
| 13 | `#00441B` | 0, 68, 27 |

## Neutral

**Name:** `neutral` · **Steps:** 9 · **Ordered:** Yes

A monochromatic palette from black to white.
Useful when you want to show magnitude without implying value judgements.

![Neutral palette swatch](palette-neutral.png)

| Step | Hex | RGB |
|------|-----|-----|
| 1 | `#000000` | 0, 0, 0 |
| 2 | `#202020` | 32, 32, 32 |
| 3 | `#404040` | 64, 64, 64 |
| 4 | `#606060` | 96, 96, 96 |
| 5 | `#808080` | 128, 128, 128 |
| 6 | `#A0A0A0` | 160, 160, 160 |
| 7 | `#C0C0C0` | 192, 192, 192 |
| 8 | `#E0E0E0` | 224, 224, 224 |
| 9 | `#FFFFFF` | 255, 255, 255 |

## Foliage

**Name:** `foliage` · **Steps:** 11 · **Ordered:** Yes

A plant-health palette from black (dead) through brown and orange to intense green (healthy).
Use for metrics where higher values represent vitality or activity.

![Foliage palette swatch](palette-foliage.png)

| Step | Hex | RGB | Description |
|------|-----|-----|-------------|
| 1 | `#0F0A05` | 15, 10, 5 | Near black (dead) |
| 2 | `#2D190A` | 45, 25, 10 | Very dark brown |
| 3 | `#552D0F` | 85, 45, 15 | Dark brown |
| 4 | `#824614` | 130, 70, 20 | Brown |
| 5 | `#AF5F19` | 175, 95, 25 | Dark orange |
| 6 | `#D2821E` | 210, 130, 30 | Orange |
| 7 | `#E6AF28` | 230, 175, 40 | Yellow-orange |
| 8 | `#F0D732` | 240, 215, 50 | Yellow |
| 9 | `#A5C832` | 165, 200, 50 | Yellow-green |
| 10 | `#50A528` | 80, 165, 40 | Medium green |
| 11 | `#197814` | 25, 120, 20 | Intense green |
