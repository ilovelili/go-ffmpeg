package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ErrFFMPEGNotFound is returned when the ffmpeg binary was not found
	ErrFFMPEGNotFound     = errors.New("ffmpeg bin not found")
	ffmpegBinPath         = "ffmpeg"
	extractingImageOption *ExtractingImagesOption
)

// SetFFMPEGBinPath sets the global path to find and execute the ffmpeg program
func SetFFMPEGBinPath(newBinPath string) {
	ffmpegBinPath = newBinPath
}

// ExtractingImagesOption extracting images option
type ExtractingImagesOption struct {
	FrameRate    string
	OutputWidth  *uint
	OutputHeight *uint
	filePath     string
}

// DefaultExtractingImagesOption set default extract image optin
func DefaultExtractingImagesOption(filePath string) {
	extractingImageOption = &ExtractingImagesOption{
		FrameRate: "1",
		filePath:  filePath,
	}
}

// NewExtractingImagesOption sets the global extract image optin
func NewExtractingImagesOption(option *ExtractingImagesOption) {
	extractingImageOption = option
}

// ExtractingImages is used for retrieve the first frame of given media file using ffmpeg with a set timeout.
// The timeout can be provided to kill the process if it takes too long to determine
// the files information.
// Note: It is probably better to use Context with GetFirstFrameContext() these days as it is more flexible.
func ExtractingImages(timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return ExtractingImagesContext(ctx)
}

// ExtractingImagesContext is used for retrieve the first frame of given media file using ffmpeg.
// It takes a context to allow killing the ffmpeg process if it takes too long or in case of shutdown.
// // ffmpeg -i intro.mp4 -r 0.5 -s 640x320 -f image2 intro-%03d.jpeg
func ExtractingImagesContext(ctx context.Context) (err error) {
	if extractingImageOption == nil {
		return fmt.Errorf("option not set")
	}
	outputFileFormat := resolveOutputFileFormat(extractingImageOption.filePath)
	resize := extractingImageOption.OutputWidth != nil && extractingImageOption.OutputHeight != nil

	var args []string
	if resize {
		args = []string{
			"-i", extractingImageOption.filePath,
			"-r", extractingImageOption.FrameRate,
			"-s", fmt.Sprintf("%dx%d", *extractingImageOption.OutputWidth, *extractingImageOption.OutputHeight),
			"-f", "image2",
			outputFileFormat,
		}
	} else {
		args = []string{
			"-i", extractingImageOption.filePath,
			"-r", extractingImageOption.FrameRate,
			"-f", "image2",
			outputFileFormat,
		}
	}

	cmd := exec.Command(
		ffmpegBinPath,
		args...,
	)

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf

	err = cmd.Start()
	if err == exec.ErrNotFound {
		return ErrFFMPEGNotFound
	} else if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		err = cmd.Process.Kill()
		if err == nil {
			return ErrTimeout
		}
		return err
	case err = <-done:
		if err != nil {
			return err
		}
	}

	return nil
}

func resolveOutputFileFormat(filePath string) string {
	baseFileName := filepath.Base(filePath)
	return strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName)) + "-%03d.jpeg"
}
