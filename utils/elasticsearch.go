package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"os"
	"strings"
)

type ElasticsearchClient interface {
	IndexClient(ctx context.Context, index string, id string, document interface{}) error
	SearchClients(ctx context.Context, index string, query map[string]interface{}) ([]map[string]interface{}, error)
	DeleteClient(ctx context.Context, index string, id string) error
	Close() error
}

type elasticsearchClient struct {
	client *elasticsearch.Client
}

func NewElasticsearchClient() (ElasticsearchClient, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Проверка подключения
	res, err := es.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch ping error: %s", res.Status())
	}

	return &elasticsearchClient{client: es}, nil
}

func (e *elasticsearchClient) Close() error {
	// Клиент Elasticsearch не требует явного закрытия
	return nil
}

func (e *elasticsearchClient) IndexClient(ctx context.Context, index string, id string, document interface{}) error {
	jsonDoc, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       strings.NewReader(string(jsonDoc)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}

func (e *elasticsearchClient) SearchClients(ctx context.Context, index string, query map[string]interface{}) ([]map[string]interface{}, error) {
	var buf strings.Builder
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode query: %w", err)
	}

	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(index),
		e.client.Search.WithBody(strings.NewReader(buf.String())),
		e.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
	results := make([]map[string]interface{}, len(hits))

	for i, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		results[i] = source
	}

	return results, nil
}

func (e *elasticsearchClient) DeleteClient(ctx context.Context, index string, id string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: id,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}
