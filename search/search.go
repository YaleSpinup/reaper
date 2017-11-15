package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"git.yale.edu/spinup/reaper/common"
	log "github.com/sirupsen/logrus"
	elastic "gopkg.in/olivere/elastic.v5"
)

// Finder is the connection to elasticsearch
type Finder struct {
	Client *elastic.Client
}

// DateRangeQuery is the properties required for a date range query in the resource finder
type DateRangeQuery struct {
	Field      string
	Gt         string
	Gte        string
	Lt         string
	Lte        string
	Format     string
	To         string
	From       string
	TermFilter []TermQuery
}

// TermQuery is the properties required for a term query in the resource finder
type TermQuery struct {
	Term  string
	Value string
}

// NewTermQueryList generates a new list of term queries from a map of strings to strings
func NewTermQueryList(filters map[string]string) []TermQuery {
	var tqs []TermQuery
	for key, value := range filters {
		tqs = append(tqs, TermQuery{Term: key, Value: value})
	}
	return tqs
}

// NewFinder creates a new elasticsearch finder.  It doesn't currently allow for
// all possible elasticsearch settings, but only the ones we need.
func NewFinder(config *common.Config) (*Finder, error) {
	log.Debugf("Configuring Finder with %+v", *config)

	var finder Finder
	var options []elastic.ClientOptionFunc

	if endpoint, ok := config.SearchEngine["endpoint"]; !ok {
		log.Debug("Setting elasticsearch URL to http://127.0.0.1:9200")
		options = append(options, elastic.SetURL("http://127.0.0.1:9200"))
	} else {
		log.Debugf("Setting elasticsearch URL to %s", endpoint)
		options = append(options, elastic.SetURL(endpoint))
	}

	if sniff, ok := config.SearchEngine["sniff"]; !ok {
		options = append(options, elastic.SetSniff(false))
	} else {
		s, err := strconv.ParseBool(sniff)
		if err != nil {
			log.Errorf("Cannot parse sniff value '%s' as boolean, %s", sniff, err)
			return &finder, err
		}
		options = append(options, elastic.SetSniff(s))
	}

	client, err := elastic.NewClient(options...)
	if err != nil {
		log.Errorln("Couldn't create new elasticsearch client", err)
		return &finder, err
	}
	finder.Client = client

	return &finder, nil
}

// DoGet does the get of an ID and returns a resource
func (f *Finder) DoGet(index, rtype, id string) (*Resource, error) {
	// Do the needful get document
	doc, err := elastic.NewGetService(f.Client).Index(index).Type(rtype).Id(id).Do(context.Background())
	if err != nil {
		log.Errorln("Failed to execute fetch from elasticsearch", err)
		return nil, err
	}

	// Deserialize doc.Source into a Resource
	var r Resource
	err = json.Unmarshal(*doc.Source, &r)
	if err != nil {
		log.Errorln("Couldn't deserialize response from elasticsearch into resource", err)
		return nil, err
	}

	return &r, nil
}

// DoDateRangeQuery searches elasticsearch for a variable number of date range queries
func (f *Finder) DoDateRangeQuery(index string, drqs ...*DateRangeQuery) ([]*Resource, error) {
	var resourceList []*Resource
	log.Debugf("Client status: %s", f.Client.String())

	q, err := constructBoolQuery(drqs)
	if err != nil {
		log.Errorln("Failed to construct date range query", err)
		return nil, err
	}

	// execute search on index
	searchResult, err := f.Client.Search().Index(index).Query(q).Size(1000).Do(context.Background())
	if err != nil {
		log.Errorln("Failed to execute search", err)
		return nil, err
	}

	log.Debugf("Query took %d milliseconds", searchResult.TookInMillis)

	if searchResult.Hits.TotalHits > 0 {
		log.Debugf("Found a total of %d resources", searchResult.Hits.TotalHits)

		// Iterate through results
		for _, hit := range searchResult.Hits.Hits {
			log.Debugf("Hit source: %s", *hit.Source)

			// Deserialize hit.Source into a Resource (could also be just a map[string]interface{}).
			var r Resource
			err := json.Unmarshal(*hit.Source, &r)
			if err != nil {
				log.Errorln("Couldn't deserialize response from elasticsearch into resource", err)
				continue
			}
			r.ID = hit.Id
			resourceList = append(resourceList, &r)
		}
	} else {
		log.Debugf("Found no resources")
	}

	return resourceList, nil
}

// constructRangeQuery puts the query together from the given properties
func contructRangeQuery(drq *DateRangeQuery) elastic.Query {
	// create a new range query
	rangeQuery := elastic.NewRangeQuery(drq.Field).Format(drq.Format)

	// set "to"
	if drq.To != "" {
		rangeQuery.To(drq.To)
	}

	// set "from"
	if drq.From != "" {
		rangeQuery.From(drq.From)
	}

	// set "greater than"
	if drq.Gt != "" {
		rangeQuery.Gt(drq.Gt)
	}

	// set greater than or equal
	if drq.Gte != "" {
		rangeQuery.Gte(drq.Gte)
	}

	// set less than
	if drq.Lt != "" {
		rangeQuery.Lt(drq.Lt)
	}

	// set less than or equal
	if drq.Lte != "" {
		rangeQuery.Lte(drq.Lte)
	}

	return rangeQuery
}

func constructBoolQuery(drqs []*DateRangeQuery) (elastic.Query, error) {
	boolQuery := elastic.NewBoolQuery()

	for _, drq := range drqs {
		rangeQuery := contructRangeQuery(drq)

		log.Debugln("Adding rangeQuery to boolean query")
		boolQuery.Must(rangeQuery)

		for _, tq := range drq.TermFilter {
			log.Debugf("Adding term filter (%s:%s) to boolean query", tq.Term, tq.Value)
			keyword := fmt.Sprintf("%s.keyword", tq.Term)
			boolQuery.Filter(elastic.NewTermQuery(keyword, tq.Value))
		}
	}

	src, _ := boolQuery.Source()
	data, err := json.Marshal(src)
	if err != nil {
		log.Errorln("Unable to marshall boolQuery into JSON", err)
		return nil, err
	}

	log.Debugf("Raw bool query: %s", string(data))

	return boolQuery, nil
}
