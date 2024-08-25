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
Relevant Links
https://github.com/opensearch-project/opensearch-go/tree/main/guides
https://medium.com/jaegertracing/using-elasticsearch-rollover-to-manage-indices-8b3d0c77915d
https://github.com/aws-samples/siem-on-amazon-opensearch-service
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
13. Dynamic Index Mapping vs statically typed mappings
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

func (s *SearchIndex) IndexRecord(baseIndexName, recordId string, item interface{}, indexProperties string) error {
	timestamp := time.Now()
	indexName := s.getIndexName(baseIndexName, timestamp)

	// Check if the index for today already exists
	exists, err := s.checkIndex(indexName)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if !exists {
		s.logger.Info(fmt.Sprintf("Index with name %s does not exist in the OpenSearch Cluster. Creating it...", indexName))
		err = s.createIndex(indexName, indexProperties)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Index record
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
		return fmt.Errorf("failed to index record: %w", err)
	}
	defer func() {
		err = res.Body.Close()
	}()

	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	// Refresh the index to make the documents searchable
	_, err = s.client.Indices.Refresh(s.client.Indices.Refresh.WithIndex(indexName))
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}

	s.logger.Info(fmt.Sprintf("Record with Id %s successfully indexed in %s and index refreshed", recordId, indexName))
	return nil
}

// getIndexName constructs the time-based index pattern
func (s *SearchIndex) getIndexName(baseIndexName string, date time.Time) string {
	return fmt.Sprintf("%s-%s", baseIndexName, date.Format("2006.01.02"))
}

// createIndex creates a new index with basic settings, including the types for index
func (s *SearchIndex) createIndex(indexName string, indexProperties string) error {
	mapping := fmt.Sprintf(`{
		"settings": {
			"number_of_shards": 3,
			"number_of_replicas": 0
		},
		"mappings": {
			"properties":%s
		}
	}`, indexProperties)

	createIndex := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}
	createIndexResponse, err := createIndex.Do(s.ctx, s.client)
	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to create Index: %v", err.Error()))
		return err
	}
	s.logger.Info(fmt.Sprintf("Index created successfully: %v", createIndexResponse))
	return nil
}

// Search searches across all indices with a given prefix
func (s *SearchIndex) Search(baseIndexName string, queryParams map[string]interface{}, sortField string) ([]map[string]interface{}, error) {
	indexPattern := fmt.Sprintf("%s-*", baseIndexName)

	query, err := buildSearchQuery(queryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	s.logger.Debug("Constructed search query", zap.String("query", query))

	searchRequest := opensearchapi.SearchRequest{
		Index: []string{indexPattern},
		Body:  strings.NewReader(query),
		Size:  opensearchapi.IntPtr(1000), //adjust based on need or parameterise it
	}

	if sortField != "" {
		searchRequest.Sort = []string{fmt.Sprintf("%s:desc", sortField)}
	}

	res, err := searchRequest.Do(s.ctx, s.client)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	defer func() {
		err = res.Body.Close()
	}()

	if res.IsError() {
		var e map[string]interface{}
		if err = json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, fmt.Errorf("error parsing the response body: %w", err)
		}

		s.logger.Error("Search request failed", zap.Any("error_response", e))

		if rootCause, ok := e["error"].(map[string]interface{})["root_cause"].([]interface{}); ok && len(rootCause) > 0 {
			cause := rootCause[0].(map[string]interface{})
			return nil, fmt.Errorf("search failed: type: %v, reason: %v", cause["type"], cause["reason"])
		}

		return nil, fmt.Errorf("search request failed: %s", res.String())
	}

	var result map[string]interface{}
	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	results := make([]map[string]interface{}, len(hits))

	for i, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		index := hit.(map[string]interface{})["_index"].(string)
		source["_index"] = index
		results[i] = source
	}

	return results, nil
}

// checkIndex - Check if an Index exists
func (s *SearchIndex) checkIndex(indexName string) (bool, error) {
	res, err := s.client.Indices.Exists([]string{indexName})
	if err != nil {
		return false, fmt.Errorf("failed to check index existence: %w", err)
	}
	defer func() {
		err = res.Body.Close()
	}()

	return res.StatusCode == 200, nil
}

// buildSearchQuery constructs the OpenSearch query from the provided parameters
func buildSearchQuery(queryParams map[string]interface{}) (string, error) {
	must := []map[string]interface{}{}

	for key, value := range queryParams {
		switch v := value.(type) {
		case string:
			must = append(must, map[string]interface{}{
				"match": map[string]interface{}{
					key: v,
				},
			})
		case []string:
			must = append(must, map[string]interface{}{
				"terms": map[string]interface{}{
					key: v,
				},
			})
		case map[string]interface{}:
			must = append(must, map[string]interface{}{
				"range": map[string]interface{}{
					key: v,
				},
			})
		case float64:
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{
					key: v,
				},
			})
		default:
			return "", fmt.Errorf("unsupported value type for key %s: %T", key, value)
		}
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	return string(queryJSON), nil
}

// BulkIndex performs bulk indexing of documents
//func (s *SearchIndex) BulkIndex(baseIndexName string, records []interface{}) error {
//	if len(records) == 0 {
//		return nil
//	}
//
//	// Ensure the index template exists based on the first record
//	if err := s.ensureIndexTemplate(baseIndexName, records[0]); err != nil {
//		return fmt.Errorf("failed to ensure index template: %w", err)
//	}
//
//	var (
//		wg        sync.WaitGroup
//		batchSize = 1000
//		batches   = (len(records) + batchSize - 1) / batchSize
//		errChan   = make(chan error, batches)
//	)
//
//	for i := 0; i < batches; i++ {
//		wg.Add(1)
//		go func(start int) {
//			defer wg.Done()
//			end := start + batchSize
//			if end > len(records) {
//				end = len(records)
//			}
//
//			bulk := &strings.Builder{}
//			for j := start; j < end; j++ {
//				record := records[j]
//
//				// Determine the timestamp for indexing
//				timestamp := time.Now()
//				if t, ok := getTimeField(record); ok {
//					timestamp = t
//				}
//				indexName := s.getIndexName(baseIndexName, timestamp)
//
//				// Prepare the document
//				doc, err := prepareDocument(record)
//				if err != nil {
//					errChan <- fmt.Errorf("failed to prepare document at index %d: %w", j, err)
//					return
//				}
//
//				// Create the action line (index instruction)
//				action := map[string]interface{}{
//					"index": map[string]interface{}{
//						"_index": indexName,
//						"_id":    fmt.Sprintf("%s-%d", baseIndexName, j), // You might want to use a more meaningful ID
//					},
//				}
//				actionLine, err := json.Marshal(action)
//				if err != nil {
//					errChan <- fmt.Errorf("failed to marshal action at index %d: %w", j, err)
//					return
//				}
//
//				// Create the document line
//				docLine, err := json.Marshal(doc)
//				if err != nil {
//					errChan <- fmt.Errorf("failed to marshal document at index %d: %w", j, err)
//					return
//				}
//
//				// Append action and document lines to the bulk request
//				bulk.Write(actionLine)
//				bulk.WriteString("\n")
//				bulk.Write(docLine)
//				bulk.WriteString("\n")
//			}
//
//			// Perform the bulk index request
//			res, err := s.client.Bulk(strings.NewReader(bulk.String()))
//			if err != nil {
//				errChan <- fmt.Errorf("bulk indexing failed for batch starting at %d: %w", start, err)
//				return
//			}
//			defer res.Body.Close()
//
//			if res.IsError() {
//				errChan <- fmt.Errorf("bulk indexing request failed for batch starting at %d: %s", start, res.String())
//				return
//			}
//
//			// Parse the response to check for individual document errors
//			var bulkResponse map[string]interface{}
//			if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
//				errChan <- fmt.Errorf("failed to parse bulk response for batch starting at %d: %w", start, err)
//				return
//			}
//
//			if bulkResponse["errors"].(bool) {
//				for _, item := range bulkResponse["items"].([]interface{}) {
//					index := item.(map[string]interface{})["index"].(map[string]interface{})
//					if index["error"] != nil {
//						errChan <- fmt.Errorf("error indexing document %s: %v", index["_id"], index["error"])
//					}
//				}
//			}
//		}(i * batchSize)
//	}
//
//	wg.Wait()
//	close(errChan)
//
//	// Collect all errors
//	var errors []string
//	for err := range errChan {
//		errors = append(errors, err.Error())
//	}
//
//	if len(errors) > 0 {
//		return fmt.Errorf("bulk indexing encountered errors: %s", strings.Join(errors, "; "))
//	}
//
//	// Refresh the index to make the documents searchable
//	_, err := s.client.Indices.Refresh(s.client.Indices.Refresh.WithIndex(fmt.Sprintf("%s-*", baseIndexName)))
//	if err != nil {
//		return fmt.Errorf("failed to refresh index: %w", err)
//	}
//
//	s.logger.Info("Bulk indexing completed and index refreshed", zap.String("baseIndexName", baseIndexName), zap.Int("recordCount", len(records)))
//	return nil
//}
