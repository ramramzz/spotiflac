package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type EmbeddedLyrics struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Lyrics string `json:"lyrics"`
	Source string `json:"source"`
	Synced bool   `json:"synced"`
	Error  string `json:"error,omitempty"`
}

var lrcTimestampRe = regexp.MustCompile(`\[\d{1,2}:\d{2}(?:\.\d{1,3})?\]`)

func isSyncedLyrics(lyrics string) bool {
	return lrcTimestampRe.MatchString(lyrics)
}

func ReadEmbeddedLyrics(filePath string) (*EmbeddedLyrics, error) {
	if !fileExists(filePath) {
		return nil, fmt.Errorf("file does not exist")
	}

	result := &EmbeddedLyrics{
		Path: filePath,
		Name: filepath.Base(filePath),
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	var lyrics string
	var err error

	switch ext {
	case ".lrc", ".txt":
		var content []byte
		content, err = os.ReadFile(filePath)
		if err == nil {
			lyrics = string(content)
			result.Source = "lrc"
		}
	case ".flac":
		lyrics, err = readFlacLyrics(filePath)
		result.Source = "embedded"
	case ".mp3":
		lyrics, err = readMp3Lyrics(filePath)
		result.Source = "embedded"
	case ".m4a", ".aac", ".opus", ".ogg":
		lyrics, err = readLyricsWithFFprobe(filePath)
		result.Source = "embedded"
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	lyrics = strings.TrimSpace(lyrics)
	if lyrics == "" {
		result.Error = "No lyrics found in this file"
		return result, nil
	}

	result.Lyrics = lyrics
	result.Synced = isSyncedLyrics(lyrics)
	return result, nil
}

func readFlacLyrics(filePath string) (string, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	for _, block := range f.Meta {
		if block.Type != flac.VorbisComment {
			continue
		}
		cmt, err := flacvorbis.ParseFromMetaDataBlock(*block)
		if err != nil {
			continue
		}
		for _, comment := range cmt.Comments {
			parts := strings.SplitN(comment, "=", 2)
			if len(parts) != 2 {
				continue
			}
			fieldName := strings.ToUpper(parts[0])
			switch fieldName {
			case "LYRICS", "UNSYNCEDLYRICS", "SYNCEDLYRICS", "LYRICS-XXX":
				if strings.TrimSpace(parts[1]) != "" {
					return parts[1], nil
				}
			}
		}
	}

	return "", nil
}

func readMp3Lyrics(filePath string) (string, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return "", fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	frames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	for _, frame := range frames {
		uslf, ok := frame.(id3v2.UnsynchronisedLyricsFrame)
		if !ok {
			continue
		}
		if strings.TrimSpace(uslf.Lyrics) != "" {
			return uslf.Lyrics, nil
		}
	}

	return "", nil
}

func readLyricsWithFFprobe(filePath string) (string, error) {
	ffprobePath, err := GetFFprobePath()
	if err != nil {
		return "", err
	}

	if err := ValidateExecutable(ffprobePath); err != nil {
		return "", fmt.Errorf("invalid ffprobe executable: %w", err)
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	setHideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var probe struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Tags map[string]string `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probe); err != nil {
		return "", err
	}

	collect := func(tags map[string]string) string {
		for key, value := range tags {
			lk := strings.ToLower(key)
			if lk == "lyrics" || strings.HasPrefix(lk, "lyrics-") || lk == "unsyncedlyrics" {
				if strings.TrimSpace(value) != "" {
					return value
				}
			}
		}
		return ""
	}

	if lyrics := collect(probe.Format.Tags); lyrics != "" {
		return lyrics, nil
	}
	for _, stream := range probe.Streams {
		if lyrics := collect(stream.Tags); lyrics != "" {
			return lyrics, nil
		}
	}

	return "", nil
}

type ExtractLyricsResult struct {
	Path          string `json:"path"`
	OutputPath    string `json:"output_path,omitempty"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
	AlreadyExists bool   `json:"already_exists,omitempty"`
}

func ExtractLyricsToLRC(filePath string, overwrite bool) (*ExtractLyricsResult, error) {
	result := &ExtractLyricsResult{Path: filePath}

	embedded, err := ReadEmbeddedLyrics(filePath)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	if embedded.Error != "" {
		result.Error = embedded.Error
		return result, nil
	}

	if strings.TrimSpace(embedded.Lyrics) == "" {
		result.Error = "No lyrics found in this file"
		return result, nil
	}

	dir := filepath.Dir(filePath)
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	outputPath := filepath.Join(dir, base+".lrc")
	result.OutputPath = outputPath

	if !overwrite {
		if info, statErr := os.Stat(outputPath); statErr == nil && info.Size() > 0 {
			result.AlreadyExists = true
			result.Error = "LRC file already exists"
			return result, nil
		}
	}

	content := embedded.Lyrics
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	if writeErr := os.WriteFile(outputPath, []byte(content), 0644); writeErr != nil {
		result.Error = fmt.Sprintf("failed to write LRC file: %v", writeErr)
		return result, nil
	}

	result.Success = true
	return result, nil
}

func SelectLyricsFiles(ctx context.Context) ([]string, error) {
	return runtime.OpenMultipleFilesDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Lyrics or Audio Files",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Lyrics & Audio (*.lrc, *.flac, *.mp3, *.m4a, *.opus)",
				Pattern:     "*.lrc;*.flac;*.mp3;*.m4a;*.aac;*.opus;*.ogg;*.txt",
			},
			{
				DisplayName: "LRC Files (*.lrc)",
				Pattern:     "*.lrc",
			},
			{
				DisplayName: "Audio Files (*.flac, *.mp3, *.m4a, *.opus)",
				Pattern:     "*.flac;*.mp3;*.m4a;*.aac;*.opus;*.ogg",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
}

func SelectLyricsFolder(ctx context.Context) (string, error) {
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Folder with Lyrics or Audio Files",
	})
}

var lyricsFolderExtensions = map[string]bool{
	".lrc": true, ".txt": true,
	".flac": true, ".mp3": true, ".m4a": true,
	".aac": true, ".opus": true, ".ogg": true,
}

func ScanLyricsFolder(dir string) ([]string, error) {
	dir = NormalizePath(dir)
	if dir == "" {
		return nil, fmt.Errorf("folder path is required")
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a valid folder: %s", dir)
	}

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if lyricsFolderExtensions[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}

type SaveLyricsResult struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func SaveLyrics(filePath string, lyrics string) (*SaveLyricsResult, error) {
	result := &SaveLyricsResult{Path: filePath}

	if !fileExists(filePath) {
		result.Error = "file does not exist"
		return result, nil
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".lrc", ".txt":
		content := lyrics
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			result.Error = fmt.Sprintf("failed to write file: %v", err)
			return result, nil
		}
	case ".flac", ".mp3", ".m4a":
		if err := EmbedLyricsOnlyUniversal(filePath, lyrics); err != nil {
			result.Error = err.Error()
			return result, nil
		}
	default:
		result.Error = fmt.Sprintf("saving is not supported for %s files", ext)
		return result, nil
	}

	result.Success = true
	return result, nil
}
