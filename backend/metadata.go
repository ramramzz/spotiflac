package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	pathfilepath "path/filepath"
	"strconv"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"golang.org/x/text/unicode/norm"
)

type Metadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Separator   string
	Date        string
	ReleaseDate string
	TrackNumber int
	TotalTracks int
	DiscNumber  int
	TotalDiscs  int
	URL         string
	Comment     string
	Copyright   string
	Publisher   string
	Composer    string
	Lyrics      string
	Description string
	ISRC        string
	UPC         string
	Genre       string
}

func resolveMetadataSeparator(separator string) string {
	if normalized := normalizeArtistSeparator(separator); normalized != "" {
		return normalized
	}

	return normalizeArtistSeparator(GetSeparator())
}

func EmbedMetadata(filepath string, metadata Metadata, coverPath string) error {
	return TagFile(filepath, metadata, coverPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func resolveMetadataComment(metadata Metadata) string {
	if comment := strings.TrimSpace(metadata.Comment); comment != "" {
		return comment
	}

	return strings.TrimSpace(metadata.URL)
}

func EmbedLyricsOnly(filepath string, lyrics string) error {
	if lyrics == "" {
		return nil
	}
	f, err := flac.ParseFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	var cmtIdx = -1
	var existingCmt *flacvorbis.MetaDataBlockVorbisComment
	for idx, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmtIdx = idx
			existingCmt, err = flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				existingCmt = nil
			}
			break
		}
	}

	cmt := flacvorbis.New()

	if existingCmt != nil {
		for _, comment := range existingCmt.Comments {
			parts := strings.SplitN(comment, "=", 2)
			if len(parts) == 2 {
				fieldName := strings.ToUpper(parts[0])
				if fieldName != "LYRICS" && fieldName != "UNSYNCEDLYRICS" && fieldName != "SYNCEDLYRICS" {
					_ = cmt.Add(parts[0], parts[1])
				}
			}
		}
	}

	_ = cmt.Add("LYRICS", lyrics)

	cmtBlock := cmt.Marshal()
	if cmtIdx < 0 {
		f.Meta = append(f.Meta, &cmtBlock)
	} else {
		f.Meta[cmtIdx] = &cmtBlock
	}

	if err := f.Save(filepath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %w", err)
	}

	return nil
}

func ExtractCoverArt(filePath string) (string, error) {
	filePath = norm.NFC.String(filePath)
	ext := strings.ToLower(pathfilepath.Ext(filePath))

	var coverPath string
	var err error

	if coverPath, err = extractCoverArtWithTagLib(filePath); err == nil && coverPath != "" {
		return coverPath, nil
	}

	switch ext {
	case ".mp3":
		coverPath, err = extractCoverFromMp3(filePath)
	case ".m4a", ".flac":
		coverPath, err = extractCoverFromM4AOrFlac(filePath)
	default:
		return "", fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil || coverPath == "" {
		fmt.Printf("[ExtractCoverArt] Library extraction failed for %s, trying FFmpeg fallback...\n", filePath)
		ffmpegCover, ffmpegErr := extractCoverWithFFmpeg(filePath)
		if ffmpegErr == nil {
			return ffmpegCover, nil
		}
		return coverPath, err
	}

	return coverPath, nil
}

func extractCoverWithFFmpeg(filePath string) (string, error) {
	ffmpegPath, err := GetFFmpegPath()
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "cover-*.jpg")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	cmd := exec.Command(ffmpegPath,
		"-i", filePath,
		"-an",
		"-vframes", "1",
		"-f", "image2",
		"-update", "1",
		"-y",
		tmpPath,
	)

	setHideWindow(cmd)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("ffmpeg cover extraction failed: %v, output: %s", err, string(output))
	}

	if info, err := os.Stat(tmpPath); err != nil || info.Size() == 0 {
		os.Remove(tmpPath)
		return "", fmt.Errorf("ffmpeg produced empty cover file")
	}

	return tmpPath, nil
}

func extractCoverFromMp3(filePath string) (string, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return "", fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	pictures := tag.GetFrames(tag.CommonID("Attached picture"))
	if len(pictures) == 0 {
		return "", fmt.Errorf("no cover art found")
	}

	pic, ok := pictures[0].(id3v2.PictureFrame)
	if !ok {
		return "", fmt.Errorf("invalid picture frame")
	}

	tmpFile, err := os.CreateTemp("", "cover-*.jpg")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(pic.Picture); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write cover art: %w", err)
	}

	return tmpFile.Name(), nil
}

func extractCoverFromM4AOrFlac(filePath string) (string, error) {
	ext := strings.ToLower(pathfilepath.Ext(filePath))

	if ext == ".flac" {
		f, err := flac.ParseFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse FLAC file: %w", err)
		}

		for _, block := range f.Meta {
			if block.Type == flac.Picture {
				pic, err := flacpicture.ParseFromMetaDataBlock(*block)
				if err != nil {
					continue
				}

				tmpFile, err := os.CreateTemp("", "cover-*.jpg")
				if err != nil {
					return "", fmt.Errorf("failed to create temp file: %w", err)
				}
				defer tmpFile.Close()

				if _, err := tmpFile.Write(pic.ImageData); err != nil {
					os.Remove(tmpFile.Name())
					return "", fmt.Errorf("failed to write cover art: %w", err)
				}

				return tmpFile.Name(), nil
			}
		}
		return "", fmt.Errorf("no cover art found")
	}

	return "", nil
}

func ExtractLyrics(filePath string) (string, error) {
	filePath = norm.NFC.String(filePath)
	ext := strings.ToLower(pathfilepath.Ext(filePath))

	var lyrics string
	var err error

	if lyrics, err = extractLyricsWithTagLib(filePath); err == nil && lyrics != "" {
		return lyrics, nil
	}

	switch ext {
	case ".mp3":
		lyrics, err = extractLyricsFromMp3(filePath)
	case ".flac":
		lyrics, err = extractLyricsFromFlac(filePath)
	case ".m4a":
		return "", nil
	default:
		return "", fmt.Errorf("unsupported file format: %s", ext)
	}

	if (err != nil || lyrics == "") && ext != ".m4a" {
		fmt.Printf("[ExtractLyrics] Library extraction failed for %s, trying ffprobe fallback...\n", filePath)
		ffprobeLyrics, ffprobeErr := extractLyricsWithFFprobe(filePath)
		if ffprobeErr == nil && ffprobeLyrics != "" {
			return ffprobeLyrics, nil
		}
	}

	return lyrics, err
}

func extractLyricsWithFFprobe(filePath string) (string, error) {
	ffprobePath, err := GetFFprobePath()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-show_entries", "format_tags=lyrics:format_tags=unsyncedlyrics:format_tags=lyric",
		"-of", "json",
		filePath,
	)

	setHideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	tags := result.Format.Tags
	for _, key := range []string{"lyrics", "unsyncedlyrics", "lyric", "LYRICS", "UNSYNCEDLYRICS", "LYRIC"} {
		if val, ok := tags[key]; ok && val != "" {
			return val, nil
		}
	}

	return "", nil
}

func extractLyricsFromMp3(filePath string) (string, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return "", fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	usltFrames := tag.GetFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))
	if len(usltFrames) == 0 {
		fmt.Printf("[ExtractLyrics] No USLT frames found in MP3: %s\n", filePath)
		return "", nil
	}

	uslt, ok := usltFrames[0].(id3v2.UnsynchronisedLyricsFrame)
	if !ok {
		fmt.Printf("[ExtractLyrics] USLT frame type assertion failed in MP3: %s\n", filePath)
		return "", nil
	}

	if uslt.Lyrics == "" {
		fmt.Printf("[ExtractLyrics] USLT frame has empty lyrics in MP3: %s\n", filePath)
		return "", nil
	}

	fmt.Printf("[ExtractLyrics] Successfully extracted lyrics from MP3: %s (%d characters)\n", filePath, len(uslt.Lyrics))
	return uslt.Lyrics, nil
}

func extractLyricsFromFlac(filePath string) (string, error) {
	f, err := flac.ParseFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse FLAC file: %w", err)
	}

	for _, block := range f.Meta {
		if block.Type == flac.VorbisComment {
			cmt, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}

			for _, comment := range cmt.Comments {
				parts := strings.SplitN(comment, "=", 2)
				if len(parts) == 2 {
					fieldName := strings.ToUpper(parts[0])
					if fieldName == "LYRICS" || fieldName == "UNSYNCEDLYRICS" {
						lyrics := parts[1]
						fmt.Printf("[ExtractLyrics] Successfully extracted lyrics from FLAC: %s (%d characters)\n", filePath, len(lyrics))
						return lyrics, nil
					}
				}
			}
		}
	}

	fmt.Printf("[ExtractLyrics] No lyrics found in FLAC: %s\n", filePath)
	return "", nil
}

func EmbedCoverArtOnly(filePath string, coverPath string) error {
	if coverPath == "" || !fileExists(coverPath) {
		return nil
	}

	ext := strings.ToLower(pathfilepath.Ext(filePath))

	switch ext {
	case ".mp3":
		return embedCoverToMp3(filePath, coverPath)
	case ".m4a":

		return nil
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

func embedCoverToMp3(filePath string, coverPath string) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	tag.DeleteFrames(tag.CommonID("Attached picture"))

	artwork, err := os.ReadFile(coverPath)
	if err != nil {
		return fmt.Errorf("failed to read cover art: %w", err)
	}

	pic := id3v2.PictureFrame{
		Encoding:    id3v2.EncodingUTF8,
		MimeType:    "image/jpeg",
		PictureType: id3v2.PTFrontCover,
		Description: "Front cover",
		Picture:     artwork,
	}
	tag.AddAttachedPicture(pic)

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	return nil
}

func EmbedLyricsOnlyMP3(filepath string, lyrics string) error {
	if lyrics == "" {
		return nil
	}

	validatedLyrics, err := validateLyricsDuration(lyrics, filepath)
	if err != nil {
		fmt.Printf("[EmbedLyricsOnlyMP3] Warning: Failed to validate lyrics duration: %v, using original lyrics\n", err)
		validatedLyrics = lyrics
	}
	lyrics = validatedLyrics

	tag, err := id3v2.Open(filepath, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer tag.Close()

	tag.DeleteFrames(tag.CommonID("Unsynchronised lyrics/text transcription"))

	usltFrame := id3v2.UnsynchronisedLyricsFrame{
		Encoding:          id3v2.EncodingUTF8,
		Language:          "eng",
		ContentDescriptor: "",
		Lyrics:            lyrics,
	}
	tag.AddUnsynchronisedLyricsFrame(usltFrame)

	if err := tag.Save(); err != nil {
		return fmt.Errorf("failed to save MP3 tags: %w", err)
	}

	return nil
}

func embedLyricsToM4A(filepath string, lyrics string) error {

	validatedLyrics, err := validateLyricsDuration(lyrics, filepath)
	if err != nil {
		fmt.Printf("[embedLyricsToM4A] Warning: Failed to validate lyrics duration: %v, using original lyrics\n", err)
		validatedLyrics = lyrics
	}
	lyrics = validatedLyrics

	ffmpegPath, err := GetFFmpegPath()
	if err != nil {
		return fmt.Errorf("ffmpeg not found: %w", err)
	}

	if err := ValidateExecutable(ffmpegPath); err != nil {
		return fmt.Errorf("invalid ffmpeg executable: %w", err)
	}

	tmpOutputFile := strings.TrimSuffix(filepath, pathfilepath.Ext(filepath)) + ".tmp" + pathfilepath.Ext(filepath)
	defer func() {

		if _, err := os.Stat(tmpOutputFile); err == nil {
			os.Remove(tmpOutputFile)
		}
	}()

	cmd := exec.Command(ffmpegPath,
		"-i", filepath,
		"-map", "0",
		"-map_metadata", "0",
		"-metadata", "lyrics-eng="+lyrics,
		"-metadata", "lyrics="+lyrics,
		"-codec", "copy",
		"-f", "ipod",
		"-y",
		tmpOutputFile,
	)

	setHideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[FFmpeg] Error embedding lyrics to M4A: %s\n", string(output))
		return fmt.Errorf("ffmpeg failed to embed lyrics: %s - %w", string(output), err)
	}

	if err := os.Rename(tmpOutputFile, filepath); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	fmt.Printf("[FFmpeg] Lyrics embedded to M4A successfully: %d characters\n", len(lyrics))
	return nil
}

func EmbedLyricsOnlyUniversal(filepath string, lyrics string) error {
	if lyrics == "" {
		return nil
	}

	validatedLyrics, err := validateLyricsDuration(lyrics, filepath)
	if err != nil {
		fmt.Printf("[EmbedLyricsOnlyUniversal] Warning: Failed to validate lyrics duration: %v, using original lyrics\n", err)
		validatedLyrics = lyrics
	}
	lyrics = validatedLyrics

	ext := strings.ToLower(pathfilepath.Ext(filepath))
	switch ext {
	case ".mp3":
		return EmbedLyricsOnlyMP3(filepath, lyrics)
	case ".flac":
		return EmbedLyricsOnly(filepath, lyrics)
	case ".m4a":
		return embedLyricsToM4A(filepath, lyrics)
	default:
		return fmt.Errorf("unsupported file format for lyrics embedding: %s", ext)
	}
}

func GetAudioDuration(filepath string) (float64, error) {
	ext := strings.ToLower(pathfilepath.Ext(filepath))

	if ext == ".flac" {
		duration, err := getFlacDuration(filepath)
		if err == nil && duration > 0 {
			return duration, nil
		}
	}

	return getDurationWithFFprobe(filepath)
}

func getFlacDuration(filepath string) (float64, error) {
	f, err := flac.ParseFile(filepath)
	if err != nil {
		return 0, err
	}

	if len(f.Meta) > 0 {
		streamInfo := f.Meta[0]
		if streamInfo.Type == flac.StreamInfo {
			data := streamInfo.Data
			if len(data) >= 18 {

				sampleRate := uint32(data[10])<<12 | uint32(data[11])<<4 | uint32(data[12])>>4

				totalSamples := uint64(data[13]&0x0F)<<32 |
					uint64(data[14])<<24 |
					uint64(data[15])<<16 |
					uint64(data[16])<<8 |
					uint64(data[17])

				if sampleRate > 0 {
					return float64(totalSamples) / float64(sampleRate), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not extract duration from FLAC file")
}

func getDurationWithFFprobe(filepath string) (float64, error) {
	ffprobePath, err := GetFFprobePath()
	if err != nil {
		return 0, err
	}

	if err := ValidateExecutable(ffprobePath); err != nil {
		return 0, fmt.Errorf("invalid ffprobe executable: %w", err)
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filepath,
	)

	setHideWindow(cmd)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return 0, err
	}

	if result.Format.Duration == "" {
		return 0, fmt.Errorf("duration not found in ffprobe output")
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func validateLyricsDuration(lyrics string, filepath string) (string, error) {

	duration, err := GetAudioDuration(filepath)
	if err != nil {

		fmt.Printf("[ValidateLyrics] Warning: Could not get audio duration: %v, skipping validation\n", err)
		return lyrics, nil
	}

	if duration <= 0 {

		fmt.Printf("[ValidateLyrics] Warning: Invalid duration (%f seconds), skipping validation\n", duration)
		return lyrics, nil
	}

	durationMs := int64(duration * 1000)

	lines := strings.Split(lyrics, "\n")
	var validLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			validLines = append(validLines, line)
			continue
		}

		if strings.HasPrefix(trimmedLine, "[") {

			closeBracket := strings.Index(trimmedLine, "]")
			if closeBracket > 0 {
				timestampStr := trimmedLine[1:closeBracket]

				ms := parseLRCTimestamp(timestampStr)
				if ms >= 0 {
					if ms <= durationMs {
						validLines = append(validLines, line)
					} else {
						fmt.Printf("[ValidateLyrics] Filtered out line with timestamp %s (exceeds duration %d ms): %s\n", timestampStr, durationMs, trimmedLine)
					}
				} else {

					validLines = append(validLines, line)
				}
				continue
			}
		} else {

			validLines = append(validLines, line)
		}
	}

	return strings.Join(validLines, "\n"), nil
}

func parseLRCTimestamp(timestamp string) int64 {
	var minutes, seconds, centiseconds int64
	n, _ := fmt.Sscanf(timestamp, "%d:%d.%d", &minutes, &seconds, &centiseconds)
	if n >= 2 {
		return minutes*60*1000 + seconds*1000 + centiseconds*10
	}
	return -1
}

func ExtractFullMetadataFromFile(filePath string) (Metadata, error) {
	filePath = norm.NFC.String(filePath)
	if metadata, err := extractFullMetadataWithTagLib(filePath); err == nil && hasMeaningfulMetadata(metadata) {
		if ffprobeMetadata, ffprobeErr := extractFullMetadataWithFFprobe(filePath); ffprobeErr == nil {
			return mergeExtractedMetadata(metadata, ffprobeMetadata), nil
		}
		return metadata, nil
	}

	return extractFullMetadataWithFFprobe(filePath)
}

func extractFullMetadataWithFFprobe(filePath string) (Metadata, error) {
	filePath = norm.NFC.String(filePath)
	var metadata Metadata

	ffprobePath, err := GetFFprobePath()
	if err != nil {
		return metadata, err
	}

	if err := ValidateExecutable(ffprobePath); err != nil {
		return metadata, fmt.Errorf("invalid ffprobe executable: %w", err)
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
		return metadata, err
	}

	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			Tags map[string]string `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return metadata, err
	}

	allTags := make(map[string]string)

	for _, stream := range result.Streams {
		for key, value := range stream.Tags {
			allTags[strings.ToLower(key)] = value
		}
	}

	for key, value := range result.Format.Tags {
		allTags[strings.ToLower(key)] = value
	}

	for key, value := range allTags {
		switch key {
		case "title":
			metadata.Title = value
		case "artist":
			metadata.Artist = value
		case "album":
			metadata.Album = value
		case "album_artist", "albumartist":
			metadata.AlbumArtist = value
		case "date", "year":
			if metadata.Date == "" || len(value) > len(metadata.Date) {
				metadata.Date = value
			}
		case "track":

			parts := strings.Split(value, "/")
			if len(parts) > 0 {
				if num, err := strconv.Atoi(parts[0]); err == nil {
					metadata.TrackNumber = num
				}
			}
			if len(parts) > 1 {
				if num, err := strconv.Atoi(parts[1]); err == nil {
					metadata.TotalTracks = num
				}
			}
		case "disc":

			parts := strings.Split(value, "/")
			if len(parts) > 0 {
				if num, err := strconv.Atoi(parts[0]); err == nil {
					metadata.DiscNumber = num
				}
			}
			if len(parts) > 1 {
				if num, err := strconv.Atoi(parts[1]); err == nil {
					metadata.TotalDiscs = num
				}
			}
		case "copyright", "tcop":
			metadata.Copyright = value
		case "publisher", "tpub", "label":
			metadata.Publisher = value
		case "composer", "writer", "wm/composer", "©wrt":
			metadata.Composer = value
		case "genre", "tcon":
			metadata.Genre = value
		case "url":
			metadata.URL = value
		case "isrc", "tsrc":
			metadata.ISRC = value
		case "comment", "comments":
			if metadata.Comment == "" {
				metadata.Comment = value
			}
		case "description":
			if metadata.Description == "" {
				metadata.Description = value
			}
		}
	}

	metadata.UPC = firstPreferredFFprobeUPCValue(allTags)

	return metadata, nil
}

func EmbedMetadataToConvertedFile(filePath string, metadata Metadata, coverPath string) error {
	filePath = norm.NFC.String(filePath)
	ext := strings.ToLower(pathfilepath.Ext(filePath))

	switch ext {
	case ".flac", ".mp3", ".m4a", ".ogg":
		return TagFile(filePath, metadata, coverPath)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}
