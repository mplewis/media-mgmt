# Plan

This project is called media-mgmt.

- Language: Go
- Logging framework: Slog
  - Make sure it has pretty printing, interactive terminal colorization, debug/info/warn/error levels, -v verbose mode, and configuration of log level via LOG_LEVEL env var
- CLI app

# Purpose

This app does the following:

- Find all video files recursively from the given dir
- Analyze them for the following details:
  - Video codec
  - Bitrate
  - Audio tracks + details
  - Subtitle tracks + details
  - File size
- Write a report

# Analysis

Use CPU count to analyze in parallel, or take --parallelism to override.

Use FFprobe to get media details.

# Reporting

Reports are generated into a given target folder in the following formats:

- CSV
- MD
- JSON
- Interactive one-file HTML with sortable columns

The folder is created if it does not exist.

# Structure

Don't over-factor this app. It is a small reporting app and it will not get much bigger than a simple reporting CLI utility.

For stuff that doesn't require wild mocking, write unit tests.

# Style

Use Go idiomatic style where possible.

Write docstrings for all functions.

Don't write comments in code unless something is very complicated and the comments are absolutely necessary to understand how it works.

# Evaluation

When you're done, use the following folders which contain media to generate reports:

- /Users/mplewis/code/personal/jpguide
- /Users/mplewis/tmp/snw-season-1

Read the reports and figure out if it's working properly. If not, go back and redo the work until it does.

# Process

You are authorized to do whatever you want to get this done. Install packages, run commands, run tests, run the app, search the web for info.

Do not ask the human for permission to continue. Just go ahead until the project matches this spec.
