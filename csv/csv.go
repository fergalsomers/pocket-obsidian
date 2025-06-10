package csv

import (
	"encoding/csv"
	"fmt"
	"os"
)

func ReadCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	return records, nil
}

func WriteCSV(filepath string, header []string, records [][]string) error {
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(header)
	if err != nil {
		return err
	}
	for _, data := range records {
		err = writer.Write(data)
		if err != nil {
			return err
		}
	}
	return nil
}
