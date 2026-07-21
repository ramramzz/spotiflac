package backend

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Eyevinn/mp4ff/mp4"
)

func decryptWithMP4FF(keySpecs []string, inputPath, outputPath string) error {
	key, keysByKID, strictKIDMode, err := parseMP4FFKeySpecs(keySpecs)
	if err != nil {
		return err
	}

	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted MP4: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create decrypted MP4: %w", err)
	}
	outClosed := false
	defer func() {
		if !outClosed {
			_ = outFile.Close()
		}
	}()

	if err := decryptMP4FFFileWithKeyMap(inFile, nil, outFile, key, keysByKID, strictKIDMode); err != nil {
		_ = outFile.Close()
		outClosed = true
		_ = os.Remove(outputPath)
		return fmt.Errorf("mp4ff decryption failed: %w", err)
	}

	if err := outFile.Close(); err != nil {
		outClosed = true
		_ = os.Remove(outputPath)
		return fmt.Errorf("failed to finalize decrypted MP4: %w", err)
	}
	outClosed = true

	return nil
}

func parseMP4FFKeySpecs(keySpecs []string) (key []byte, keysByKID map[string][]byte, strictKIDMode bool, err error) {
	normalizedSpecs := make([]string, 0, len(keySpecs))
	seenSpecs := make(map[string]struct{}, len(keySpecs))
	for _, spec := range keySpecs {
		normalized, err := normalizeMP4FFKeySpec(spec)
		if err != nil {
			return nil, nil, false, err
		}
		if normalized == "" {
			continue
		}
		if _, ok := seenSpecs[normalized]; ok {
			continue
		}
		seenSpecs[normalized] = struct{}{}
		normalizedSpecs = append(normalizedSpecs, normalized)
	}

	if len(normalizedSpecs) == 0 {
		return nil, nil, false, fmt.Errorf("no mp4ff key specs provided")
	}

	hasKIDPair := false
	hasLegacyKey := false
	for _, spec := range normalizedSpecs {
		if strings.Contains(spec, ":") {
			hasKIDPair = true
		} else {
			hasLegacyKey = true
		}
	}

	if hasKIDPair && hasLegacyKey {
		return nil, nil, false, fmt.Errorf("cannot mix legacy key and kid:key key format")
	}

	if !hasKIDPair {
		if len(normalizedSpecs) != 1 {
			return nil, nil, false, fmt.Errorf("multiple legacy keys are not supported")
		}
		key, err = mp4.UnpackKey(normalizedSpecs[0])
		if err != nil {
			return nil, nil, false, fmt.Errorf("unpacking key: %w", err)
		}
		return key, nil, false, nil
	}

	keysByKID = make(map[string][]byte, len(normalizedSpecs))
	for _, spec := range normalizedSpecs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, nil, false, fmt.Errorf("bad kid:key format %q", spec)
		}

		kid, err := mp4.UnpackKey(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, nil, false, fmt.Errorf("unpacking kid: %w", err)
		}
		kidHex := hex.EncodeToString(kid)
		if _, exists := keysByKID[kidHex]; exists {
			return nil, nil, false, fmt.Errorf("duplicate kid %s", kidHex)
		}

		parsedKey, err := mp4.UnpackKey(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, nil, false, fmt.Errorf("unpacking key for kid %s: %w", kidHex, err)
		}
		keysByKID[kidHex] = parsedKey
	}

	return nil, keysByKID, true, nil
}

func normalizeMP4FFKeySpec(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || !strings.Contains(spec, ":") {
		return spec, nil
	}

	parts := strings.SplitN(spec, ":", 2)
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left == "" || right == "" {
		return "", fmt.Errorf("bad key spec %q", spec)
	}

	if _, err := mp4.UnpackKey(left); err == nil {
		return left + ":" + right, nil
	}
	if !isDecimalString(left) {
		return "", fmt.Errorf("bad kid in key spec %q", spec)
	}

	if _, err := mp4.UnpackKey(right); err != nil {
		return "", fmt.Errorf("bad key spec %q: %w", spec, err)
	}

	return right, nil
}

func isDecimalString(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func decryptMP4FFFileWithKeyMap(r, initR io.Reader, w io.Writer, key []byte, keysByKID map[string][]byte, strictKIDMode bool) error {
	inMp4, err := mp4.DecodeFile(r)
	if err != nil {
		return err
	}
	if !inMp4.IsFragmented() {
		return fmt.Errorf("file not fragmented. Not supported")
	}

	init := inMp4.Init
	if inMp4.Init == nil {
		if initR == nil {
			return fmt.Errorf("no init segment file and no init part of file")
		}
		initSegment, err := mp4.DecodeFile(initR)
		if err != nil {
			return fmt.Errorf("could not decode init file: %w", err)
		}
		init = initSegment.Init
	}

	decryptInfo, err := mp4.DecryptInit(init)
	if err != nil {
		return err
	}

	if inMp4.Init != nil {
		if err := inMp4.Init.Encode(w); err != nil {
			return err
		}
	}

	for _, segment := range inMp4.Segments {
		if inMp4.Init == nil {
			if err := segment.ParseSenc(init); err != nil {
				return fmt.Errorf("parseSenc: %w", err)
			}
		}

		if err := decryptMP4FFSegmentWithSparseSenc(segment, decryptInfo, key, keysByKID, strictKIDMode); err != nil {
			return fmt.Errorf("decryptSegment: %w", err)
		}
		if err := segment.Encode(w); err != nil {
			return err
		}
	}

	return nil
}

func decryptMP4FFSegmentWithSparseSenc(segment *mp4.MediaSegment, decryptInfo mp4.DecryptInfo, key []byte, keysByKID map[string][]byte, strictKIDMode bool) error {
	for _, fragment := range segment.Fragments {
		if !mp4FragmentContainsSenc(fragment) {
			continue
		}
		if err := mp4.DecryptFragmentWithKeys(fragment, decryptInfo, key, keysByKID, strictKIDMode); err != nil {
			return err
		}
	}

	if len(segment.Sidxs) > 0 {
		segment.Sidx = nil
		segment.Sidxs = nil
	}

	return nil
}

func mp4FragmentContainsSenc(fragment *mp4.Fragment) bool {
	if fragment == nil || fragment.Moof == nil {
		return false
	}
	for _, traf := range fragment.Moof.Trafs {
		if traf == nil {
			continue
		}
		hasSenc, _ := traf.ContainsSencBox()
		if hasSenc {
			return true
		}
	}
	return false
}
