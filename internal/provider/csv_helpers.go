package provider

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

type csvSnapshot struct {
	Headers []string
	Rows    [][]string
}

func parseCSVContent(content string) (csvSnapshot, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return csvSnapshot{}, fmt.Errorf("could not parse csv_content: %w", err)
	}
	if len(records) == 0 {
		return csvSnapshot{}, fmt.Errorf("csv_content must include at least a header row")
	}

	return csvSnapshot{
		Headers: records[0],
		Rows:    records[1:],
	}, nil
}

func renderCSVContent(headers []string, rows [][]string) (string, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("could not render csv headers: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("could not render csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("could not finalize csv content: %w", err)
	}

	return buffer.String(), nil
}

func csvRowsToUpdatePayload(rows [][]string) ([]string, error) {
	payload := make([]string, 0, len(rows))
	for _, row := range rows {
		buffer := &bytes.Buffer{}
		writer := csv.NewWriter(buffer)
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("could not encode csv row: %w", err)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, fmt.Errorf("could not finalize csv row: %w", err)
		}
		payload = append(payload, strings.TrimSuffix(buffer.String(), "\n"))
	}

	return payload, nil
}

func computeColumnChanges(currentHeaders []string, desiredHeaders []string) []map[string]interface{} {
	changes := make([]map[string]interface{}, 0)

	if len(desiredHeaders) > len(currentHeaders) {
		for index := len(currentHeaders); index < len(desiredHeaders); index++ {
			changes = append(changes, map[string]interface{}{
				"changeKind": "Add",
				"index":      index,
			})
		}
	}

	if len(currentHeaders) > len(desiredHeaders) {
		for index := len(currentHeaders) - 1; index >= len(desiredHeaders); index-- {
			changes = append(changes, map[string]interface{}{
				"changeKind": "Remove",
				"index":      index,
			})
		}
	}

	return changes
}
