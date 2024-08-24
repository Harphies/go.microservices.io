package elasticsearch

/*
doc := map[string]interface{}{
    "id": "123",
    "content": "Sample content",
    "timestamp": time.Now(),
}
err := searchIndex.IndexRecord("myindex", "123", doc, time.Now())
if err != nil {
    log.Fatal(err)
}


    FileSize:         321092,
    // ... other field values ...
}

err = searchIndex.IndexRecord("diagnoses", "unique-id-2", diagnosisRecord)
if err != nil {
    log.Fatal(err)
}
This implementation now handles multiple record types dynamically, creating appropriate templates and mappings for each type of record. It's more flexible and easier to use with different data structures.

Is there anything else you'd like me to explain or modify in this implementation?

Copy
Retry


Claude does not have the ability to run the code it generates yet.
Claude can make mistakes. Please double-check responses.



No file chosen


3.5 Sonnet

AWS OpenSearch with Go Implementation

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v2/signer/awsv2"
	"go.uber.org/zap"
)

type SearchIndex struct {
	client           *opensearch.Client
	logger           *zap.Logger
	ctx              context.Context
	templatesMutex   sync.Mutex
	createdTemplates map[string]bool
}

func NewSearchIndex(logger *zap.Logger, endpoint string) (*SearchIndex, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	signer, err := requestsigner.NewSigner(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{endpoint},
		Signer:    signer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	info, err := client.Info()
	if err != nil {
		logger.Error("failed to establish connection with AWS OpenSearch Cluster", zap.Error(err))
		return nil, err
	}

	var r map[string]interface{}
	if err := json.NewDecoder(info.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode cluster info: %w", err)
	}

	version := r["version"].(map[string]interface{})
	logger.Info("Connection established with AWS OpenSearch Cluster", zap.String("version", version["number"].(string)))

	return &SearchIndex{
		client:           client,
		logger:           logger,
		ctx:              ctx,
		createdTemplates: make(map[string]bool),
	}, nil
}

func (s *SearchIndex) getIndexName(baseIndexName string, date time.Time) string {
	return fmt.Sprintf("%s-%s", baseIndexName, date.Format("2006.01.02"))
}

func (s *SearchIndex) IndexRecord(baseIndexName string, recordId string, record interface{}) error {
	// Ensure the index template exists
	if err := s.ensureIndexTemplate(baseIndexName, record); err != nil {
		return fmt.Errorf("failed to ensure index template: %w", err)
	}

	// Determine the timestamp for indexing
	timestamp := time.Now()
	if t, ok := getTimeField(record); ok {
		timestamp = t
	}

	indexName := s.getIndexName(baseIndexName, timestamp)

	// Prepare the document to be indexed
	doc, err := prepareDocument(record)
	if err != nil {
		return fmt.Errorf("failed to prepare document: %w", err)
	}

	// Index the document
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: recordId,
		Body:       strings.NewReader(string(body)),
	}

	res, err := req.Do(s.ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index request failed: %s", res.String())
	}

	// Refresh the index to make the documents searchable
	_, err = s.client.Indices.Refresh(s.client.Indices.Refresh.WithIndex(indexName))
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}

	s.logger.Info("Record indexed and index refreshed", zap.String("recordId", recordId), zap.String("index", indexName))
	return nil
}

func (s *SearchIndex) ensureIndexTemplate(baseIndexName string, record interface{}) error {
	s.templatesMutex.Lock()
	defer s.templatesMutex.Unlock()

	if s.createdTemplates[baseIndexName] {
		return nil // Template already created
	}

	if err := s.createIndexTemplate(baseIndexName, record); err != nil {
		return err
	}

	s.createdTemplates[baseIndexName] = true
	return nil
}

func (s *SearchIndex) createIndexTemplate(baseIndexName string, record interface{}) error {
	mappings, err := generateMappings(record)
	if err != nil {
		return fmt.Errorf("failed to generate mappings: %w", err)
	}

	template := fmt.Sprintf(`{
		"index_patterns": ["%s-*"],
		"template": {
			"settings": {
				"number_of_shards": 3,
				"number_of_replicas": 0,
				"refresh_interval": "1s"
			},
			"mappings": {
				"properties": %s
			}
		}
	}`, baseIndexName, mappings)

	req := opensearchapi.IndicesPutIndexTemplateRequest{
		Name: baseIndexName + "-template",
		Body: strings.NewReader(template),
	}

	res, err := req.Do(s.ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to create index template: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("create index template request failed: %s", res.String())
	}

	s.logger.Info("Index template created successfully", zap.String("template", baseIndexName+"-template"))
	return nil
}

func generateMappings(record interface{}) (string, error) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("record must be a struct")
	}

	mappings := make(map[string]interface{})
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		fieldType := getOpenSearchType(field.Type)
		mappings[fieldName] = map[string]string{"type": fieldType}
	}

	jsonMappings, err := json.Marshal(mappings)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mappings: %w", err)
	}

	return string(jsonMappings), nil
}

func getOpenSearchType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "long"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		return "keyword"
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return "date"
		}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.String {
			return "keyword"
		}
	}
	return "text"
}

func prepareDocument(record interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return doc, nil
}

func getTimeField(record interface{}) (time.Time, bool) {
	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return time.Time{}, false
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		if t.Field(i).Type == reflect.TypeOf(time.Time{}) {
			return v.Field(i).Interface().(time.Time), true
		}
	}

	return time.Time{}, false
}

// SearchDateRange and other methods remain the same...

// Example usage
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	searchIndex, err := NewSearchIndex(logger, "https://your-opensearch-endpoint.amazonaws.com")
	if err != nil {
		logger.Fatal("Failed to create SearchIndex", zap.Error(err))
	}

	// Example 1: Treatment Record
	treatmentRecord := struct {
		Certified   bool     `json:"certified"`
		CertifiedBy []string `json:"certifiedBy"`
		CreatedAt   string   `json:"createdAt"`
		Cure        string   `json:"cure"`
		Description string   `json:"description"`
		Image       string   `json:"image"`
		Name        string   `json:"name"`
		Price       string   `json:"price"`
		Type        string   `json:"type"`
	}{
		Certified:   true,
		CertifiedBy: []string{"BBH"},
		CreatedAt:   "1724482468775",
		Cure:        "typhoid",
		Description: "A treatment for colon cancer",
		Image:       "image from s3",
		Name:        "stm-mmm",
		Price:       "237",
		Type:        "medical",
	}

	err = searchIndex.IndexRecord("treatments", "unique-id-1", treatmentRecord)
	if err != nil {
		logger.Fatal("Failed to index treatment record", zap.Error(err))
	}

	// Example 2: Diagnosis Record
	diagnosisRecord := struct {
		DiagnosisSummary string    `json:"diagnosis_summary"`
		FileName         string    `json:"file_name"`
		FileSize         int       `json:"file_size"`
		FileType         string    `json:"file_type"`
		ID               int       `json:"id"`
		UploadPathS3     string    `json:"upload_path_s3"`
		UploadTime       time.Time `json:"upload_time"`
		UserID           string    `json:"user_id"`
	}{
		DiagnosisSummary: "check how I am feeling",
		FileName:         "carbon (17).png",
		FileSize:         321092,
		FileType:         "image",
		ID:               0,
		UploadPathS3:     "https://stm-streamex-dev-eu-west-2-services-object-store.s3.eu-west-2.amazonaws.com/diagnosis/diagnosis/d672a2a4-80d1-70c7-968d-2aeb7b04d917/2024-08-24/image/carbon (17).png",
		UploadTime:       time.Now(),
		UserID:           "d672a2a4-80d1-70c7-968d-2aeb7b04d917",
	}

	err = searchIndex.IndexRecord("diagnoses", "unique-id-2", diagnosisRecord)
	if err != nil {
		logger.Fatal("Failed to index diagnosis record", zap.Error(err))
	}

	logger.Info("Records indexed successfully")
}
*/

// Example Bulk indexing - I can bulk index from kafka topic, to speed up CPU time more business logic and
// deal with internally needed data movements separately

/*
// Example usage in main
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	searchIndex, err := NewSearchIndex(logger, "https://your-opensearch-endpoint.amazonaws.com")
	if err != nil {
		logger.Fatal("Failed to create SearchIndex", zap.Error(err))
	}

	// Example: Bulk indexing treatment records
	treatmentRecords := []interface{}{
		struct {
			Certified   bool     `json:"certified"`
			CertifiedBy []string `json:"certifiedBy"`
			CreatedAt   string   `json:"createdAt"`
			Cure        string   `json:"cure"`
			Description string   `json:"description"`
			Image       string   `json:"image"`
			Name        string   `json:"name"`
			Price       string   `json:"price"`
			Type        string   `json:"type"`
		}{
			Certified:   true,
			CertifiedBy: []string{"BBH"},
			CreatedAt:   "1724482468775",
			Cure:        "typhoid",
			Description: "A treatment for typhoid",
			Image:       "image1.jpg",
			Name:        "stm-typhoid",
			Price:       "237",
			Type:        "medical",
		},
		struct {
			Certified   bool     `json:"certified"`
			CertifiedBy []string `json:"certifiedBy"`
			CreatedAt   string   `json:"createdAt"`
			Cure        string   `json:"cure"`
			Description string   `json:"description"`
			Image       string   `json:"image"`
			Name        string   `json:"name"`
			Price       string   `json:"price"`
			Type        string   `json:"type"`
		}{
			Certified:   false,
			CertifiedBy: []string{"NHS"},
			CreatedAt:   "1724482468780",
			Cure:        "malaria",
			Description: "A treatment for malaria",
			Image:       "image2.jpg",
			Name:        "stm-malaria",
			Price:       "300",
			Type:        "medical",
		},
	}

	err = searchIndex.BulkIndex("treatments", treatmentRecords)
	if err != nil {
		logger.Fatal("Failed to bulk index treatment records", zap.Error(err))
	}

	// Example: Bulk indexing diagnosis records
	diagnosisRecords := []interface{}{
		struct {
			DiagnosisSummary string    `json:"diagnosis_summary"`
			FileName         string    `json:"file_name"`
			FileSize         int       `json:"file_size"`
			FileType         string    `json:"file_type"`
			ID               int       `json:"id"`
			UploadPathS3     string    `json:"upload_path_s3"`
			UploadTime       time.Time `json:"upload_time"`
			UserID           string    `json:"user_id"`
		}{
			DiagnosisSummary: "check how I am feeling",
			FileName:         "carbon (17).png",
			FileSize:         321092,
			FileType:         "image",
			ID:               0,
			UploadPathS3:     "https://example.com/path1",
			UploadTime:       time.Now(),
			UserID:           "user1",
		},
		struct {
			DiagnosisSummary string    `json:"diagnosis_summary"`
			FileName         string    `json:"file_name"`
			FileSize         int       `json:"file_size"`
			FileType         string    `json:"file_type"`
			ID               int       `json:"id"`
			UploadPathS3     string    `json:"upload_path_s3"`
			UploadTime       time.Time `json:"upload_time"`
			UserID           string    `json:"user_id"`
		}{
			DiagnosisSummary: "follow-up examination",
			FileName:         "scan.jpg",
			FileSize:         500000,
			FileType:         "image",
			ID:               1,
			UploadPathS3:     "https://example.com/path2",
			UploadTime:       time.Now().Add(time.Hour),
			UserID:           "user2",
		},
	}

	err = searchIndex.BulkIndex("diagnoses", diagnosisRecords)
	if err != nil {
		logger.Fatal("Failed to bulk index diagnosis records", zap.Error(err))
	}

	logger.Info("Bulk indexing completed successfully")
}
*/
