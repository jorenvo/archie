TODO
====
- auto-detect single character or word mode by first byte
- non-ascii character rendering has too much space between runes (echo '这是什么' | go run .)
- utf-8 with lots of ligatures (e.g. persian) is not rendered correctly (echo 'است باید' | go run .)
- don't spin when paused
- adjusting reading speed with a default key repeat rate
