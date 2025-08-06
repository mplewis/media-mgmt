package handbrake

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func (t *HandBrakeTranscoder) runHandBrakeCLI(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "HandBrakeCLI", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	go t.filterHandBrakeOutput(stdoutPipe)
	go t.filterHandBrakeOutput(stderrPipe)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start HandBrakeCLI: %w", err)
	}

	return cmd.Wait()
}

func (t *HandBrakeTranscoder) filterHandBrakeOutput(pipe io.ReadCloser) {
	defer pipe.Close()

	// Supported progress formats:
	// Encoding: task 1 of 1, 2.31 %
	// Encoding: task 1 of 1, 4.50 % (224.12 fps, avg 226.07 fps, ETA 00h02m48s)
	progressRegex := regexp.MustCompile(`Encoding: task \d+ of \d+, (\d+\.\d+) %(?:\s+\((\d+\.\d+) fps,.*ETA (\d+h\d+m\d+s)\))?`)

	buf := make([]byte, 1)
	var currentLine strings.Builder

	for {
		n, err := pipe.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		if n == 0 {
			continue
		}

		char := buf[0]

		if char == '\r' {
			line := currentLine.String()
			if matches := progressRegex.FindStringSubmatch(line); matches != nil {
				percent := matches[1]
				if len(matches) > 3 && matches[2] != "" {
					fps := matches[2]
					eta := matches[3]
					extraText := fmt.Sprintf(" (%s fps, ETA %s)", fps, eta)
					progressBar := t.createProgressBarWithText(percent, extraText)
					if progressBar != "" {
						fmt.Printf("\r%s %s%%%s", progressBar, percent, extraText)
					} else {
						fmt.Printf("\r%s%%%s", percent, extraText)
					}
				} else {
					progressBar := t.createProgressBar(percent)
					if progressBar != "" {
						fmt.Printf("\r%s %s%%", progressBar, percent)
					} else {
						fmt.Printf("\r%s%%", percent)
					}
				}
			}
			currentLine.Reset()
		} else if char == '\n' {
			line := currentLine.String()
			if strings.Contains(line, "Encode done!") {
				completionText := " - Encode done!"
				progressBar := t.createProgressBarWithText("100.0", completionText)
				if progressBar != "" {
					fmt.Printf("\r%s 100.0%%%s\n", progressBar, completionText)
				} else {
					fmt.Printf("\r100.0%%%s\n", completionText)
				}
			} else if strings.Contains(line, "ERROR") || strings.Contains(line, "WARNING") {
				fmt.Printf("\n%s\n", line)
			}
			currentLine.Reset()
		} else {
			currentLine.WriteByte(char)
		}
	}

	if currentLine.Len() > 0 {
		line := currentLine.String()
		fmt.Printf("%s\n", line)
	}
}

func (t *HandBrakeTranscoder) createProgressBar(percentStr string) string {
	return t.createProgressBarWithText(percentStr, "")
}

func (t *HandBrakeTranscoder) createProgressBarWithText(percentStr, extraText string) string {
	blocks := []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		return ""
	}

	termWidth := t.getTerminalWidth()

	// Calculate actual text width: " XX.X%" + extraText + space for leading space
	percentText := fmt.Sprintf(" %s%%", percentStr)
	totalTextWidth := len(percentText) + len(extraText)

	minBarWidth := 10
	minTotalWidth := minBarWidth + totalTextWidth + 2 // +2 for brackets
	if termWidth < minTotalWidth {
		return ""
	}

	barWidth := termWidth - totalTextWidth - 2 // -2 for brackets
	filled := int((percent / 100.0) * float64(barWidth))

	var bar strings.Builder
	bar.WriteRune('[')

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteRune(blocks[len(blocks)-1])
		} else if i == filled {
			partialProgress := (percent/100.0)*float64(barWidth) - float64(filled)
			if partialProgress > 0 {
				blockIndex := int(partialProgress * 8)
				if blockIndex >= len(blocks) {
					blockIndex = len(blocks) - 1
				}
				bar.WriteRune(blocks[blockIndex])
			} else {
				bar.WriteRune(' ')
			}
		} else {
			bar.WriteRune(' ')
		}
	}

	bar.WriteRune(']')
	return bar.String()
}