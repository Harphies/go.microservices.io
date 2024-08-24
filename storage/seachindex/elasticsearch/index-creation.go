package elasticsearch

import (
	"fmt"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"go.uber.org/zap"
	"strings"
)

// CreateIndexWithMapping creates a new index with proper field mappings
func (s *SearchIndex) CreateIndexWithMapping(indexName string) error {
	mapping := `{
		"settings": {
			"number_of_shards": 3,
			"number_of_replicas": 0
		},
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"certified": { "type": "boolean" },
				"certifiedBy": { "type": "keyword" },
				"createdAt": { "type": "date" },
				"updatedAt": { "type": "date" },
				"deletedAt": { "type": "date" },
				"cure": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
				"description": { "type": "text" },
				"image": { "type": "keyword" },
				"name": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
				"price": { "type": "float" },
				"type": { "type": "keyword" }
			}
		}
	}`

	req := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mapping),
	}

	res, err := req.Do(s.ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			fmt.Println("failed to close response body")
		}
	}()

	if res.IsError() {
		return fmt.Errorf("error creating index: %s", res.String())
	}

	s.logger.Info("Index created successfully with mapping", zap.String("index", indexName))
	return nil
}

func (s *SearchIndex) createIndexTemplate(baseIndexName string, record interface{}) error {

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
	}`, baseIndexName, record)

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

/*
Allow the Index Creation method to accept the properties as an argument for dynamic index creation
// for multiple services
Example properties for a treatment service
{
				"id": { "type": "keyword" },
				"certified": { "type": "boolean" },
				"certifiedBy": { "type": "keyword" },
				"createdAt": { "type": "date" },
				"updatedAt": { "type": "date" },
				"deletedAt": { "type": "date" },
				"cure": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
				"description": { "type": "text" },
				"image": { "type": "keyword" },
				"name": { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
				"price": { "type": "float" },
				"type": { "type": "keyword" }
			}
*/
