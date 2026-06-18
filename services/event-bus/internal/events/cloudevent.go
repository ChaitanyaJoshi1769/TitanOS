package events

import (
	"encoding/json"
	"fmt"
	"time"
)

type CloudEvent struct {
	SpecVersion     string                 `json:"specversion"`
	Type            string                 `json:"type"`
	Source          string                 `json:"source"`
	ID              string                 `json:"id"`
	Time            time.Time              `json:"time"`
	DataContentType string                 `json:"datacontenttype"`
	Subject         string                 `json:"subject,omitempty"`
	DataSchema      string                 `json:"dataschema,omitempty"`
	Data            map[string]interface{} `json:"data,omitempty"`
	Attributes      map[string]string      `json:"attributes,omitempty"`
}

func NewCloudEvent(eventType string, source string, id string, data map[string]interface{}) *CloudEvent {
	return &CloudEvent{
		SpecVersion:     "1.0",
		Type:            eventType,
		Source:          source,
		ID:              id,
		Time:            time.Now().UTC(),
		DataContentType: "application/json",
		Data:            data,
		Attributes:      make(map[string]string),
	}
}

func (ce *CloudEvent) Validate() error {
	if ce.SpecVersion != "1.0" {
		return fmt.Errorf("invalid specversion: %s", ce.SpecVersion)
	}
	if ce.Type == "" {
		return fmt.Errorf("type is required")
	}
	if ce.Source == "" {
		return fmt.Errorf("source is required")
	}
	if ce.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

func (ce *CloudEvent) MarshalJSON() ([]byte, error) {
	type Alias CloudEvent
	return json.Marshal(&struct {
		Time string `json:"time"`
		*Alias
	}{
		Time:  ce.Time.Format(time.RFC3339Nano),
		Alias: (*Alias)(ce),
	})
}

func (ce *CloudEvent) UnmarshalJSON(data []byte) error {
	type Alias CloudEvent
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(ce),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339Nano, aux.Time)
	if err != nil {
		return err
	}

	ce.Time = t
	return nil
}

func (ce *CloudEvent) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"specversion":       ce.SpecVersion,
		"type":              ce.Type,
		"source":            ce.Source,
		"id":                ce.ID,
		"time":              ce.Time.Format(time.RFC3339Nano),
		"datacontenttype":   ce.DataContentType,
		"subject":           ce.Subject,
		"dataschema":        ce.DataSchema,
		"data":              ce.Data,
		"attributes":        ce.Attributes,
	}
}
