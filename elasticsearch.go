package fiberextend

import "github.com/elastic/go-elasticsearch/v8"

func (p *IFiberExConfig) NewES() *elasticsearch.Client {
	es, err := elasticsearch.NewClient(*p.ESConfig)
	if err != nil {
		panic(err)
	}
	return es
}
