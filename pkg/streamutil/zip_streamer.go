package streamutil

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// CSVRowHandler is called for each parsed CSV record.
type CSVRowHandler func(record []string) error

// StreamZipCSVFilter reads matching CSV files inside a .zip file and invokes handler on each record.
// Automatically supports Latin1 (ISO-8859-1) decode when isLatin1 is true.
// If filter is non-nil, only entry names for which filter returns true will be processed.
func StreamZipCSVFilter(ctx context.Context, zipPath string, delimiter rune, isLatin1 bool, filter func(entryName string) bool, onRow CSVRowHandler) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file %s: %w", zipPath, err)
	}
	defer r.Close()

	for _, file := range r.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		baseName := filepath.Base(file.Name)
		ext := strings.ToLower(filepath.Ext(baseName))
		if ext != ".csv" && ext != ".txt" {
			continue
		}

		if filter != nil && !filter(baseName) {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open entry %s: %w", file.Name, err)
		}

		var reader io.Reader = rc
		if isLatin1 {
			reader = charmap.ISO8859_1.NewDecoder().Reader(rc)
		}

		csvReader := csv.NewReader(bufio.NewReaderSize(reader, 128*1024))
		csvReader.Comma = delimiter
		csvReader.LazyQuotes = true
		csvReader.FieldsPerRecord = -1 // Allow variable number of columns

		// Skip header line
		_, err = csvReader.Read()
		if err != nil {
			rc.Close()
			if err == io.EOF {
				continue
			}
			return fmt.Errorf("error reading header from %s: %w", file.Name, err)
		}

		for {
			select {
			case <-ctx.Done():
				rc.Close()
				return ctx.Err()
			default:
			}

			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				// Skip individual malformed line to preserve rest of file
				continue
			}

			if err := onRow(record); err != nil {
				rc.Close()
				return err
			}
		}

		rc.Close()
	}

	return nil
}

// StreamZipCSV reads all CSV files inside a .zip file and invokes handler on each record.
func StreamZipCSV(ctx context.Context, zipPath string, delimiter rune, isLatin1 bool, onRow CSVRowHandler) error {
	return StreamZipCSVFilter(ctx, zipPath, delimiter, isLatin1, nil, onRow)
}
