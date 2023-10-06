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
)

type SearchIndex struct {
	client *opensearch.Client
	logger *zap.Logger
	ctx    context.Context
}

func NewSearchIndex(logger *zap.Logger, endpoint string) *SearchIndex {
	ctx := context.Background()
	cfg, _ := config.LoadDefaultConfig(ctx)
	signer, _ := requestsigner.NewSigner(cfg)
	client, _ := opensearch.NewClient(opensearch.Config{
		Addresses: []string{endpoint},
		Signer:    signer,
	})
	info, err := client.Info()
	if err != nil {
		logger.Error("failed to establish connection with AWS OpenSearch Cluster", zap.Error(err))
	}
	var r map[string]interface{}
	_ = json.NewDecoder(info.Body).Decode(&r)
	version := r["version"].(map[string]interface{})
	logger.Info(fmt.Sprintf("Connection Eastalished with AWS OpenSearch Cluster version %v", version["number"]))
	return &SearchIndex{
		client: client,
		logger: logger,
		ctx:    ctx,
	}
}

// IndexRecord - Create an Index does not exist and index the incoming document
func (s *SearchIndex) IndexRecord(indexName, recordId string, item interface{}) {
	// Check if the Index exists before indexing the record
	if indexResp := s.getIndex(indexName); indexResp == nil {
		s.logger.Info(fmt.Sprintf("Index with name %s does not exist in the OpenSearch Cluster. Creating it......", indexName))
		s.createIndex(indexName)
	}

	// Index records
	record, _ := json.Marshal(item)
	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: recordId,
		Body:       strings.NewReader(string(record)),
	}
	_, err := req.Do(s.ctx, s.client)
	if err != nil {
		s.logger.Error(fmt.Sprintf("error occurred: [%s]", err.Error()))
	}
	// refresh the index to make the documents searchable
	_, err = s.client.Indices.Refresh(s.client.Indices.Refresh.WithIndex(indexName))
	if err != nil {
		s.logger.Error(fmt.Sprintf("error occurred: [%s]", err.Error()))
	}
	s.logger.Info(fmt.Sprintf("Record with Id %s successfully Indexed and Index Refreshed", recordId))
}

func (s *SearchIndex) DeleteIndex(indexName string) {
	deleteIndex := opensearchapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}
	deleteIndexResponse, err := deleteIndex.Do(s.ctx, s.client)

	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to delete Index: %v", err))
	}
	s.logger.Info(fmt.Sprintf("Index deleted successfully: %v", deleteIndexResponse))
}

func (s *SearchIndex) Search(indexName string, queryParams map[string]string) {
	basicSearchParam, _ := json.Marshal(queryParams)
	parsedSearchParam := OpenSearchQueryStringFormat(string(basicSearchParam))
	s.logger.Info(fmt.Sprintf("trimmed stringify input:: %v", parsedSearchParam))
	s.basicSearch(indexName, parsedSearchParam)
	/*
		// Returned response format for match hits
			{"took":1487,"timed_out":false,"_shards":{"total":3,"successful":3,"skipped":0,"failed":0},"hits":{"total":{"value":5,"relation":"eq"},"max_score":0.2876821,"hits":[{"_index":"treatments","_id":"mFBIqIoBZVk5hzSKWsdN","_score":0.2876821,"_source":{"name":"stm-cancer-c8","description":"A api-server for colon cancer stage II","image":"image from s3","price":"78","type":"traditional","cure":"cancer","certifiedBy":["NVD"]}},{"_index":"treatments","_id":"ev1AqIoBjkovibuuqhSr","_score":0.18232156,"_source":{"name":"stm-cancer-c5","description":"A api-server for colon cancer stage II","image":"image from s3","price":"67","type":"medical","cure":"cancer","certified":true,"certifiedBy":["LBS"]}},{"_index":"treatments","_id":"ff1KqIoBjkovibuuNxQl","_score":0.18232156,"_source":{"name":"stm-diabetics-d6","description":"A api-server for dibatetics type II","image":"image from s3","price":"88","type":"traditional","cure":"cancer","certified":true,"certifiedBy":["LLM"]}},{"_index":"treatments","_id":"e_1JqIoBjkovibuuHBQR","_score":0.18232156,"_source":{"name":"stm-cancer-k7","description":"A api-server for blood cancer stage II","image":"image from s3","price":"59","type":"medical","cure":"cancer","certified":true,"certifiedBy":["NHS"]}},{"_index":"treatments","_id":"fP1JqIoBjkovibuulRSB","_score":0.18232156,"_source":{"name":"stm-cancer-d6","description":"A api-server for malaria","image":"image from s3","price":"109","type":"medical","cure":"cancer","certifiedBy":["NHS"]}}]}}]
	*/
}

// SearchAll searched for all document in an index.
func (s *SearchIndex) SearchAll(indexName string) {
	res, err := s.client.Search(
		s.client.Search.WithIndex(indexName))
	if err != nil {
		s.logger.Error(fmt.Sprintf("error occured: [%s]", err.Error()))
	}
	s.logger.Info(fmt.Sprintf("response: [%+v]", res))
}

func (s *SearchIndex) createIndex(indexName string) {
	mapping := strings.NewReader(`{
	 "settings": {
	   "index": {
	        "number_of_shards": 3,
			"number_of_replicas": 0
	        }
	      }
	 }`)
	createIndex := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  mapping,
	}
	createIndexResponse, err := createIndex.Do(s.ctx, s.client)
	if err != nil {
		s.logger.Error(fmt.Sprintf("failed to create Index: %v", err.Error()))
	}
	s.logger.Info(fmt.Sprintf("Index created successfully: %v", createIndexResponse))
}

// getIndex - Check if an Index exists
func (s *SearchIndex) getIndex(indexName string) interface{} {
	res, _ := s.client.Indices.Get([]string{indexName})
	var r map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&r)
	if r[indexName] != nil {
		return r
	}
	return nil
}

func (s *SearchIndex) basicSearch(indexName, query string) {
	part, err := s.client.Search(
		s.client.Search.WithIndex(indexName),
		s.client.Search.WithQuery(fmt.Sprintf(`%s`, query)))

	if err != nil {
		s.logger.Error(fmt.Sprintf("search request failed"))
	}
	var r map[string]interface{}
	_ = json.NewDecoder(part.Body).Decode(&r)
	hits := r["hits"].(map[string]interface{})
	resp := hits["hits"]
	s.logger.Info(fmt.Sprintf("search response hits: [%+v]", resp))
}

func (s *SearchIndex) complexQuery(indexName string, query interface{}) {

}

// Formatted String for OpenSearch Query String Format

func OpenSearchQueryStringFormat(input string) string {
	trimmedString := strings.Trim(input, "{}")
	split := strings.Split(trimmedString, ":")
	return split[0][1:len(split[0])-1] + ":" + split[1]
}
