# Palettes

`codeviz` includes several colour palettes for visualising metrics.
Each palette can be applied to fill or border colours using the `--fill-palette` and `--border-palette` flags.

Ordered palettes map low-to-high metric values across a colour gradient.
The categorization palette assigns visually distinct colours to discrete categories.

## Categorization

**Name:** `categorization` · **Steps:** 12 · **Ordered:** No

Twelve visually distinct colours for categorical data such as file type.
Colours are drawn from the ColorBrewer *Paired* scheme.

| Step | Swatch | Hex | RGB |
|------|--------|-----|-----|
| 1 | ![](https://via.placeholder.com/15/A6CEE3/A6CEE3.png) | `#A6CEE3` | 166, 206, 227 |
| 2 | ![](https://via.placeholder.com/15/1F78B4/1F78B4.png) | `#1F78B4` | 31, 120, 180 |
| 3 | ![](https://via.placeholder.com/15/B2DF8A/B2DF8A.png) | `#B2DF8A` | 178, 223, 138 |
| 4 | ![](https://via.placeholder.com/15/33A02C/33A02C.png) | `#33A02C` | 51, 160, 44 |
| 5 | ![](https://via.placeholder.com/15/FB9A99/FB9A99.png) | `#FB9A99` | 251, 154, 153 |
| 6 | ![](https://via.placeholder.com/15/E31A1C/E31A1C.png) | `#E31A1C` | 227, 26, 28 |
| 7 | ![](https://via.placeholder.com/15/FDBF6F/FDBF6F.png) | `#FDBF6F` | 253, 191, 111 |
| 8 | ![](https://via.placeholder.com/15/FF7F00/FF7F00.png) | `#FF7F00` | 255, 127, 0 |
| 9 | ![](https://via.placeholder.com/15/CAB2D6/CAB2D6.png) | `#CAB2D6` | 202, 178, 214 |
| 10 | ![](https://via.placeholder.com/15/6A3D9A/6A3D9A.png) | `#6A3D9A` | 106, 61, 154 |
| 11 | ![](https://via.placeholder.com/15/FFFF99/FFFF99.png) | `#FFFF99` | 255, 255, 153 |
| 12 | ![](https://via.placeholder.com/15/B15928/B15928.png) | `#B15928` | 177, 89, 40 |

## Temperature

**Name:** `temperature` · **Steps:** 11 · **Ordered:** Yes

A diverging palette from dark blue through white to bright red,
based on the ColorBrewer *RdBu* scheme.
Suitable for metrics where a neutral midpoint is meaningful.

| Step | Swatch | Hex | RGB |
|------|--------|-----|-----|
| 1 | ![](https://via.placeholder.com/15/053061/053061.png) | `#053061` | 5, 48, 97 |
| 2 | ![](https://via.placeholder.com/15/2166AC/2166AC.png) | `#2166AC` | 33, 102, 172 |
| 3 | ![](https://via.placeholder.com/15/4393C3/4393C3.png) | `#4393C3` | 67, 147, 195 |
| 4 | ![](https://via.placeholder.com/15/92C5DE/92C5DE.png) | `#92C5DE` | 146, 197, 222 |
| 5 | ![](https://via.placeholder.com/15/D1E5F0/D1E5F0.png) | `#D1E5F0` | 209, 229, 240 |
| 6 | ![](https://via.placeholder.com/15/F7F7F7/F7F7F7.png) | `#F7F7F7` | 247, 247, 247 |
| 7 | ![](https://via.placeholder.com/15/FDDBC7/FDDBC7.png) | `#FDDBC7` | 253, 219, 199 |
| 8 | ![](https://via.placeholder.com/15/F4A582/F4A582.png) | `#F4A582` | 244, 165, 130 |
| 9 | ![](https://via.placeholder.com/15/D6604D/D6604D.png) | `#D6604D` | 214, 96, 77 |
| 10 | ![](https://via.placeholder.com/15/B2182B/B2182B.png) | `#B2182B` | 178, 24, 43 |
| 11 | ![](https://via.placeholder.com/15/67001F/67001F.png) | `#67001F` | 103, 0, 31 |

## Good/Bad

**Name:** `good-bad` · **Steps:** 13 · **Ordered:** Yes

A diverging palette from red through yellow to green,
based on the ColorBrewer *RdYlGn* scheme.
Use when low values are undesirable and high values are desirable (or vice versa).

| Step | Swatch | Hex | RGB |
|------|--------|-----|-----|
| 1 | ![](https://via.placeholder.com/15/A50026/A50026.png) | `#A50026` | 165, 0, 38 |
| 2 | ![](https://via.placeholder.com/15/D73027/D73027.png) | `#D73027` | 215, 48, 39 |
| 3 | ![](https://via.placeholder.com/15/F46D43/F46D43.png) | `#F46D43` | 244, 109, 67 |
| 4 | ![](https://via.placeholder.com/15/FDAE61/FDAE61.png) | `#FDAE61` | 253, 174, 97 |
| 5 | ![](https://via.placeholder.com/15/FEE08B/FEE08B.png) | `#FEE08B` | 254, 224, 139 |
| 6 | ![](https://via.placeholder.com/15/FFFFBF/FFFFBF.png) | `#FFFFBF` | 255, 255, 191 |
| 7 | ![](https://via.placeholder.com/15/FFFFFF/FFFFFF.png) | `#FFFFFF` | 255, 255, 255 |
| 8 | ![](https://via.placeholder.com/15/D9EF8B/D9EF8B.png) | `#D9EF8B` | 217, 239, 139 |
| 9 | ![](https://via.placeholder.com/15/A6D96A/A6D96A.png) | `#A6D96A` | 166, 217, 106 |
| 10 | ![](https://via.placeholder.com/15/66BD63/66BD63.png) | `#66BD63` | 102, 189, 99 |
| 11 | ![](https://via.placeholder.com/15/1A9850/1A9850.png) | `#1A9850` | 26, 152, 80 |
| 12 | ![](https://via.placeholder.com/15/006837/006837.png) | `#006837` | 0, 104, 55 |
| 13 | ![](https://via.placeholder.com/15/00441B/00441B.png) | `#00441B` | 0, 68, 27 |

## Neutral

**Name:** `neutral` · **Steps:** 9 · **Ordered:** Yes

A monochromatic palette from black to white.
Useful when you want to show magnitude without implying value judgements.

| Step | Swatch | Hex | RGB |
|------|--------|-----|-----|
| 1 | ![](https://via.placeholder.com/15/000000/000000.png) | `#000000` | 0, 0, 0 |
| 2 | ![](https://via.placeholder.com/15/202020/202020.png) | `#202020` | 32, 32, 32 |
| 3 | ![](https://via.placeholder.com/15/404040/404040.png) | `#404040` | 64, 64, 64 |
| 4 | ![](https://via.placeholder.com/15/606060/606060.png) | `#606060` | 96, 96, 96 |
| 5 | ![](https://via.placeholder.com/15/808080/808080.png) | `#808080` | 128, 128, 128 |
| 6 | ![](https://via.placeholder.com/15/A0A0A0/A0A0A0.png) | `#A0A0A0` | 160, 160, 160 |
| 7 | ![](https://via.placeholder.com/15/C0C0C0/C0C0C0.png) | `#C0C0C0` | 192, 192, 192 |
| 8 | ![](https://via.placeholder.com/15/E0E0E0/E0E0E0.png) | `#E0E0E0` | 224, 224, 224 |
| 9 | ![](https://via.placeholder.com/15/FFFFFF/FFFFFF.png) | `#FFFFFF` | 255, 255, 255 |

## Foliage

**Name:** `foliage` · **Steps:** 11 · **Ordered:** Yes

A plant-health palette from black (dead) through brown and orange to intense green (healthy).
Use for metrics where higher values represent vitality or activity.

| Step | Swatch | Hex | RGB | Description |
|------|--------|-----|-----|-------------|
| 1 | ![](https://via.placeholder.com/15/0F0A05/0F0A05.png) | `#0F0A05` | 15, 10, 5 | Near black (dead) |
| 2 | ![](https://via.placeholder.com/15/2D190A/2D190A.png) | `#2D190A` | 45, 25, 10 | Very dark brown |
| 3 | ![](https://via.placeholder.com/15/552D0F/552D0F.png) | `#552D0F` | 85, 45, 15 | Dark brown |
| 4 | ![](https://via.placeholder.com/15/824614/824614.png) | `#824614` | 130, 70, 20 | Brown |
| 5 | ![](https://via.placeholder.com/15/AF5F19/AF5F19.png) | `#AF5F19` | 175, 95, 25 | Dark orange |
| 6 | ![](https://via.placeholder.com/15/D2821E/D2821E.png) | `#D2821E` | 210, 130, 30 | Orange |
| 7 | ![](https://via.placeholder.com/15/E6AF28/E6AF28.png) | `#E6AF28` | 230, 175, 40 | Yellow-orange |
| 8 | ![](https://via.placeholder.com/15/F0D732/F0D732.png) | `#F0D732` | 240, 215, 50 | Yellow |
| 9 | ![](https://via.placeholder.com/15/A5C832/A5C832.png) | `#A5C832` | 165, 200, 50 | Yellow-green |
| 10 | ![](https://via.placeholder.com/15/50A528/50A528.png) | `#50A528` | 80, 165, 40 | Medium green |
| 11 | ![](https://via.placeholder.com/15/197814/197814.png) | `#197814` | 25, 120, 20 | Intense green |
