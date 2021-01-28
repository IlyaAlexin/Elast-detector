package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

type Rule struct {
	Name      string  `yaml:"name"`
	Index     string  `yaml:"index"`
	Filter    []Query `yaml:"filter"`
	RuleType  string  `yaml:"rule_type"`
	Value     int     `yaml:"rule_value"`
	GroupBy   []Group `yaml:"group_by"`
	TimeRange string  `yaml:"timerange"`
}

type Group struct {
	Field string `yaml:"field"`
	Size  int    `yaml:"size"`
}

type Query struct {
	Type       string `yaml:"type"` // term, wildcard, query
	Field      string `yaml:"field"`
	Match      string `yaml:"match"`
	RawRequest string `yaml:"query"`
}

type Event struct {
	Service    string `json:"-"`
	ConfigItem string `json:"ci"`
	Value      int    `json:"value"`
}

func (rule *Rule) buildESQuery() (bytes.Buffer, error) {
	body := bytes.Buffer{}
	requestBody := make(map[string]interface{})
	query, err := rule.parseFilters()
	if err != nil {
		return body, err
	}
	if len(query) > 0 {
		requestBody["query"] = query
	}
	aggs, err := rule.parseAggs()
	if err != nil {
		return body, err
	}
	if len(aggs) > 0 {
		requestBody["aggs"] = aggs
	}
	if len(requestBody) > 0 {
		requestBody["size"] = 0
		fmt.Printf("Final request body %v\n", requestBody)
		if err := json.NewEncoder(&body).Encode(requestBody); err != nil {
			parsingErr := fmt.Errorf("Error marshaling es query body for rule %v: %v", rule.Name, err.Error())
			return body, parsingErr
		}
		return body, nil
	}
	return body, nil
}

func (rule *Rule) parseFilters() (map[string]interface{}, error) {
	query := make(map[string]interface{})
	filter := rule.Filter[0] // TBD: need to parse multiple filters here
	if filter.Field != "" {
		if filter.Match != "" {
			query["bool"] = map[string]interface{}{
				"must": map[string]interface{}{
					"match": map[string]interface{}{
						filter.Field: filter.Match,
					},
				},
			}
			return query, nil
		}
		err := fmt.Errorf("Can't start job %v. es_query.field defined but es_query.match not found", rule.Name)
		return query, err
	}
	return query, nil
}

func (rule *Rule) parseAggs() (map[string]interface{}, error) {
	aggRequest := make(map[string]interface{})
	aggExpression := rule.GroupBy[0] // TBD: need to parse multiple groups here
	if aggExpression.Field != "" {
		if aggExpression.Size > 0 {
			aggRequest["init_agg"] = map[string]interface{}{
				"terms": map[string]interface{}{
					"field": aggExpression.Field,
					"size":  aggExpression.Size,
				},
			}
			return aggRequest, nil
		}
		errMessage := fmt.Sprintf("Rule %v:aggregation size should be > 0 but received %v\n", rule.Name, aggExpression.Size)
		return aggRequest, errors.New(errMessage)
	}
	return aggRequest, nil
}

func (rule Rule) ParseResults(response map[string]interface{}) ([]gjson.Result, error) {
	jsonResp, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	jsonString := string(jsonResp)
	values := gjson.Get(jsonString, "aggregations.init_agg.buckets")
	return values.Array(), err
}

func (rule Rule) GenerateEvents(searchRes map[string]interface{}) ([]Event, error) {
	var events []Event
	res, err := rule.ParseResults(searchRes)
	if err != nil {
		fmt.Println("Failed to Generate Events", err.Error())
		return events, err
	}
	for _, aggRes := range res {
		mapValue := aggRes.Map()
		ci := mapValue["key"].String()
		hitRes := int(mapValue["doc_count"].Int())
		if rule.triggered(hitRes) {
			newEvent := Event{
				ConfigItem: ci,
				Value:      hitRes,
			}
			events = append(events, newEvent)
		}
	}
	return events, err
}

func (rule Rule) triggered(searchValue int) bool {
	if searchValue > rule.Value {
		return true
	}
	return false
}
