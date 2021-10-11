#!/usr/bin/env python3
from PIL import Image

# (a, b, c, d) is
# a b
# c d
blocksToUnicode = {
    (0, 0, 0, 0): " ",
    (0, 0, 0, 1): "▗",
    (0, 0, 1, 0): "▖",
    (0, 0, 1, 1): "▄",
    (0, 1, 0, 0): "▝",
    (0, 1, 0, 1): "▐",
    (0, 1, 1, 0): "▞",
    (0, 1, 1, 1): "▟",
    (1, 0, 0, 0): "▘",
    (1, 0, 0, 1): "▚",
    (1, 0, 1, 0): "▌",
    (1, 0, 1, 1): "▙",
    (1, 1, 0, 0): "▀",
    (1, 1, 0, 1): "▜",
    (1, 1, 1, 0): "▛",
    (1, 1, 1, 1): "█",
}

assert len(set(blocksToUnicode.values())) == len(blocksToUnicode)

im = Image.open("archie.png")
pixels = im.load()

frameWidth = 32
frameHeight = 16

print("frames := [][]string{")
for frame in range(4):
    print("{")
    for row in range(1, 1 + frameHeight, 2):
        print("\"", end="")
        for col in range(frameWidth * frame, frameWidth * frame + frameWidth, 2):
            blocks = (
                pixels[col, row],
                pixels[col+1, row],
                pixels[col, row+1],
                pixels[col+1, row+1],
            )
            blocks = tuple(rgba[3] // 255 for rgba in blocks)

            print(blocksToUnicode[blocks], end="")
        print("\",")
    print("},")
print("}")
