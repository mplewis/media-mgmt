# Plan

Write a subcommand called `transcode`. It uses FFmpeg to convert one or more
video files based on existing presets.

The purpose of this is to downsize large, high-bitrate video files to smaller
ones that are equal enough in quality that viewers don't mind.

When possible, use hardware acceleration. Detect presence of VideoToolbox on the
system and use when possible.

Use `HandBrakeCLI`, i.e.

```
HandBrakeCLI -i "input.mkv" -o "input-optimized.mkv" --encoder vt_h265 --quality 80
```

# Behavior

If the file is HDR, use H265 10-bit. If not, use H265 8-bit. Use CQ 80 for
quality. Passthru everything else from source.

If an `-optimized` file is found already existing in the destination, we assume
we already completed the conversion, log it, and skip that file.

# UI

Users can provide --files=a.mkv,b.mkv,c.mkv or --file-list=my-files.txt,
containing lines with paths to convert.

Output file suffix defaults to `-optimized`, as in `mymedia-optimized.mkv`, and
can be set by the user at the CLI.

`--overwrite` lets a user overwrite existing files.
