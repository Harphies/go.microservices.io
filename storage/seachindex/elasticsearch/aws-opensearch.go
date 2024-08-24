package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	requestsigner "github.com/opensearch-project/opensearch-go/v2/signer/awsv2"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"sync"
	"time"
)

/*
Improvements
1. OpenSearch Connection Pooling - reusing connection efficiently
2. Batch processing of operations
3. Caching of frequently assessed data
4. Index optimization, fine-tune index settings and mappings
5. time-based indices - Index partitioning and Lifecycle management
6. Add more templates and mappings for different services payload indices or Automatically generate template base on the structure of the incoming record to index
7. Periodic Record Cleanup
8. Consider distributed Locking for the locking mechanism when running multiple replicas of th service using this package
9. Pagination: Add support for result pagination using 'from' and 'size' parameters
10. Advanced Querying: Implement more complex query types (e.g., range queries, fuzzy matching).
11. Consider other efficient index patterns for faster ingestion and efficient query/retrieval performance
12. Search across indexes based on index prefix
*/

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
	if err = json.NewDecoder(info.Body).Decode(&r); err != nil {
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

	defer func() {
		err = res.Body.Close()
	}()

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
	defer func() {
		err = res.Body.Close()
	}()

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

// Search searches across all indices with a given prefix
func (s *SearchIndex) Search(baseIndexName string, queryParams map[string]interface{}) ([]map[string]interface{}, error) {
	// Construct the index pattern to match all indices with the given prefix
	indexPattern := fmt.Sprintf("%s-*", baseIndexName)

	// Build the search query
	query := buildSearchQuery(queryParams)

	// Perform the search request
	res, err := s.client.Search(
		s.client.Search.WithContext(s.ctx),
		s.client.Search.WithIndex(indexPattern),
		s.client.Search.WithBody(strings.NewReader(query)),
		s.client.Search.WithSize(1000),             // Adjust this value based on your needs
		s.client.Search.WithSort("createdAt:desc"), // Assuming there's a createdAt field, adjust if needed
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	defer func() {
		err = res.Body.Close()
	}()

	if res.IsError() {
		return nil, fmt.Errorf("search request failed: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	results := make([]map[string]interface{}, len(hits))

	for i, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		index := hit.(map[string]interface{})["_index"].(string)
		source["_index"] = index // Include the index name in the result
		results[i] = source
	}

	return results, nil
}

func (s *SearchIndex) checkIndex(indexName string) (bool, error) {
	res, err := s.client.Indices.Exists([]string{indexName})
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}
	defer func() {
		err = res.Body.Close()
	}()

	return !res.IsError(), nil
}

// buildSearchQuery constructs the OpenSearch query from the provided parameters
func buildSearchQuery(queryParams map[string]interface{}) string {
	must := []map[string]interface{}{}

	for key, value := range queryParams {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				key: value,
			},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}

	queryJSON, _ := json.Marshal(query)
	return string(queryJSON)
}

// BulkIndex performs bulk indexing of documents
func (s *SearchIndex) BulkIndex(baseIndexName string, records []interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// Ensure the index template exists based on the first record
	if err := s.ensureIndexTemplate(baseIndexName, records[0]); err != nil {
		return fmt.Errorf("failed to ensure index template: %w", err)
	}

	var (
		wg        sync.WaitGroup
		batchSize = 1000
		batches   = (len(records) + batchSize - 1) / batchSize
		errChan   = make(chan error, batches)
	)

	for i := 0; i < batches; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			end := start + batchSize
			if end > len(records) {
				end = len(records)
			}

			bulk := &strings.Builder{}
			for j := start; j < end; j++ {
				record := records[j]

				// Determine the timestamp for indexing
				timestamp := time.Now()
				if t, ok := getTimeField(record); ok {
					timestamp = t
				}
				indexName := s.getIndexName(baseIndexName, timestamp)

				// Prepare the document
				doc, err := prepareDocument(record)
				if err != nil {
					errChan <- fmt.Errorf("failed to prepare document at index %d: %w", j, err)
					return
				}

				// Create the action line (index instruction)
				action := map[string]interface{}{
					"index": map[string]interface{}{
						"_index": indexName,
						"_id":    fmt.Sprintf("%s-%d", baseIndexName, j), // You might want to use a more meaningful ID
					},
				}
				actionLine, err := json.Marshal(action)
				if err != nil {
					errChan <- fmt.Errorf("failed to marshal action at index %d: %w", j, err)
					return
				}

				// Create the document line
				docLine, err := json.Marshal(doc)
				if err != nil {
					errChan <- fmt.Errorf("failed to marshal document at index %d: %w", j, err)
					return
				}

				// Append action and document lines to the bulk request
				bulk.Write(actionLine)
				bulk.WriteString("\n")
				bulk.Write(docLine)
				bulk.WriteString("\n")
			}

			// Perform the bulk index request
			res, err := s.client.Bulk(strings.NewReader(bulk.String()))
			if err != nil {
				errChan <- fmt.Errorf("bulk indexing failed for batch starting at %d: %w", start, err)
				return
			}
			defer res.Body.Close()

			if res.IsError() {
				errChan <- fmt.Errorf("bulk indexing request failed for batch starting at %d: %s", start, res.String())
				return
			}

			// Parse the response to check for individual document errors
			var bulkResponse map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
				errChan <- fmt.Errorf("failed to parse bulk response for batch starting at %d: %w", start, err)
				return
			}

			if bulkResponse["errors"].(bool) {
				for _, item := range bulkResponse["items"].([]interface{}) {
					index := item.(map[string]interface{})["index"].(map[string]interface{})
					if index["error"] != nil {
						errChan <- fmt.Errorf("error indexing document %s: %v", index["_id"], index["error"])
					}
				}
			}
		}(i * batchSize)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errors []string
	for err := range errChan {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("bulk indexing encountered errors: %s", strings.Join(errors, "; "))
	}

	// Refresh the index to make the documents searchable
	_, err := s.client.Indices.Refresh(s.client.Indices.Refresh.WithIndex(fmt.Sprintf("%s-*", baseIndexName)))
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}

	s.logger.Info("Bulk indexing completed and index refreshed", zap.String("baseIndexName", baseIndexName), zap.Int("recordCount", len(records)))
	return nil
}
