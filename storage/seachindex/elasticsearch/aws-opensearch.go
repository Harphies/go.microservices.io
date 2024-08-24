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
6. Add more templates for different sort of indices
7. Periodic Record Cleanup
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

func (s *SearchIndex) IndexRecord(baseIndexName string, recordId string, item interface{}, date time.Time) error {
	// Ensure the index template exists
	if err := s.ensureIndexTemplate(baseIndexName); err != nil {
		return fmt.Errorf("failed to ensure index template: %w", err)
	}

	indexName := s.getIndexName(baseIndexName, date)

	// Check if the Index exists before indexing the record
	exists, err := s.checkIndex(indexName)
	if err != nil {
		return fmt.Errorf("failed to check index: %w", err)
	}

	if !exists {
		s.logger.Info("Index does not exist. It will be created automatically based on the template.", zap.String("index", indexName))
	}

	// Index records
	record, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: recordId,
		Body:       strings.NewReader(string(record)),
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

func (s *SearchIndex) ensureIndexTemplate(baseIndexName string) error {
	s.templatesMutex.Lock()
	defer s.templatesMutex.Unlock()

	if s.createdTemplates[baseIndexName] {
		return nil // Template already created
	}

	if err := s.createIndexTemplate(baseIndexName); err != nil {
		return err
	}

	s.createdTemplates[baseIndexName] = true
	return nil
}

func (s *SearchIndex) createIndexTemplate(baseIndexName string) error {
	template := fmt.Sprintf(`{
		"index_patterns": ["%s-*"],
		"template": {
			"settings": {
				"number_of_shards": 3,
				"number_of_replicas": 0,
				"refresh_interval": "1s"
			},
			"mappings": {
				"properties": {
					"timestamp": {
						"type": "date"
					}
				}
			}
		}
	}`, baseIndexName)

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

func (s *SearchIndex) SearchDateRange(baseIndexName string, startDate, endDate time.Time, queryParams map[string]string) ([]map[string]interface{}, error) {
	var indexNames []string
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		indexNames = append(indexNames, s.getIndexName(baseIndexName, d))
	}

	query := buildSearchQuery(queryParams)

	res, err := s.client.Search(
		s.client.Search.WithContext(s.ctx),
		s.client.Search.WithIndex(indexNames...),
		s.client.Search.WithBody(strings.NewReader(query)),
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
	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	results := make([]map[string]interface{}, len(hits))

	for i, hit := range hits {
		results[i] = hit.(map[string]interface{})["_source"].(map[string]interface{})
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

func buildSearchQuery(queryParams map[string]string) string {
	var conditions []string
	for key, value := range queryParams {
		conditions = append(conditions, fmt.Sprintf(`{"match": {"%s": "%s"}}`, key, value))
	}
	query := fmt.Sprintf(`
	{
		"query": {
			"bool": {
				"must": [
					%s
				]
			}
		}
	}`, strings.Join(conditions, ","))
	return query
}

// BulkIndex performs bulk indexing of documents
func (s *SearchIndex) BulkIndex(baseIndexName string, documents []map[string]interface{}) error {
	// Ensure the index template exists
	if err := s.ensureIndexTemplate(baseIndexName); err != nil {
		return fmt.Errorf("failed to ensure index template: %w", err)
	}

	var (
		wg        sync.WaitGroup
		batchSize = 1000
		batches   = (len(documents) + batchSize - 1) / batchSize
	)

	for i := 0; i < batches; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			end := start + batchSize
			if end > len(documents) {
				end = len(documents)
			}

			bulk := &strings.Builder{}
			for j := start; j < end; j++ {
				doc := documents[j]
				timestamp, ok := doc["timestamp"].(time.Time)
				if !ok {
					s.logger.Error("Document missing timestamp", zap.Int("index", j))
					continue
				}
				indexName := s.getIndexName(baseIndexName, timestamp)
				meta := []byte(fmt.Sprintf(`{"index":{"_index":"%s","_id":"%d"}}%s`, indexName, j, "\n"))
				data, _ := json.Marshal(doc)
				data = append(data, "\n"...)

				bulk.Write(meta)
				bulk.Write(data)
			}

			res, err := s.client.Bulk(strings.NewReader(bulk.String()))
			if err != nil {
				s.logger.Error("Bulk indexing failed", zap.Error(err))
				return
			}
			defer func() {
				err = res.Body.Close()
			}()

			if res.IsError() {
				s.logger.Error("Bulk indexing request failed", zap.String("response", res.String()))
			}
		}(i * batchSize)
	}

	wg.Wait()
	return nil
}
