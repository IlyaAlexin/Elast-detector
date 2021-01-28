package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	elastic "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

type EsClient struct {
	Escli *elastic.Client
}

func NewEsClient(uri string) (EsClient, error) {
	escfg := elastic.Config{
		Addresses: []string{uri},
	}
	esinstance, err := elastic.NewClient(escfg)
	esclient := EsClient{
		Escli: esinstance,
	}
	if err != nil {
		err = fmt.Errorf("Failed to created es client with uri %v, Error: %v", uri, err.Error())
		return esclient, err
	}
	return esclient, nil
}

func (escli *EsClient) Count(countFuncs []func(*esapi.CountRequest)) (map[string]interface{}, error) {
	res, err := escli.Escli.Count(countFuncs...)
	if err != nil {
		log.Println("Failed to make Count request to ES", err)
	}
	response, err := escli.processBody(res)
	return response, err
}

func (escli *EsClient) Search(searchFuncs []func(*esapi.SearchRequest)) (map[string]interface{}, error) {
	res, err := escli.Escli.Search(searchFuncs...)
	for _, funcSearch := range searchFuncs {
		fmt.Println(funcSearch)
	}
	if err != nil {
		log.Println("Failed to make Search request to ES", err)
	}
	response, err := escli.processBody(res)
	return response, err
}

func (escli *EsClient) processBody(response *esapi.Response) (map[string]interface{}, error) {
	defer response.Body.Close()

	if response.IsError() {
		var errorResp map[string]interface{}
		if err := json.NewDecoder(response.Body).Decode(&errorResp); err != nil {
			fmt.Println("Failed to parse response body", err.Error())
			return errorResp, err
		}
		errMessage := fmt.Sprintf("%v %v: %v\n",
			response.Status(),
			errorResp["error"].(map[string]interface{})["type"],
			errorResp["error"].(map[string]interface{})["reason"],
		)
		return errorResp, errors.New(errMessage)
	}
	var parsedBody map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&parsedBody); err != nil {
		errMessage := fmt.Sprintf("Error parsing the response body: %s", err)
		log.Printf(errMessage)
		return parsedBody, errors.New(errMessage)
	}
	log.Println(parsedBody)
	return parsedBody, nil
}

func (escli *EsClient) ConstructCount(rule Rule) ([]func(*esapi.CountRequest), error) {
	countFuncs := make([]func(*esapi.CountRequest), 0, 5)
	ruleName := rule.Name
	if index := rule.Index; index != "" {
		countFuncs = append(countFuncs, escli.Escli.Count.WithIndex(index))
	} else {
		err := fmt.Sprintf("Rule %v: 'index' field not defined. Can't run elasticsearch query wihtout it", ruleName)
		return countFuncs, errors.New(err)
	}
	body, err := rule.buildESQuery()
	if err != nil {
		fmt.Println(err.Error())
		return countFuncs, err
	}
	fmt.Println(string(body.Bytes()))
	if !reflect.DeepEqual(body, bytes.Buffer{}) {
		countFuncs = append(countFuncs, escli.Escli.Count.WithBody(&body))
	}
	countFuncs = append(countFuncs, escli.Escli.Count.WithContext(context.Background()))
	return countFuncs, nil
}

func (escli *EsClient) ConstructSearch(rule Rule) ([]func(*esapi.SearchRequest), error) {
	fmt.Printf("Get new search rule: %v\n", rule)
	searchFuncs := make([]func(*esapi.SearchRequest), 0, 5)
	ruleName := rule.Name
	if index := rule.Index; index != "" {
		searchFuncs = append(searchFuncs, escli.Escli.Search.WithIndex(index))
	} else {
		err := fmt.Sprintf("Rule %v: 'index' field not defined. Can't run elasticsearch query wihtout it", ruleName)
		return searchFuncs, errors.New(err)
	}
	body, err := rule.buildESQuery()
	if err != nil {
		return searchFuncs, err
	}
	if !reflect.DeepEqual(body, bytes.Buffer{}) {
		searchFuncs = append(searchFuncs, escli.Escli.Search.WithBody(&body))
	}
	searchFuncs = append(searchFuncs, escli.Escli.Search.WithContext(context.Background()))
	return searchFuncs, nil
}
