package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"reflect"
)

const (
	namespace = "tb" // For Prometheus metrics.
)

func (e *Exporter) BuildDescriptions() {
	var metricDesc = make(map[string]*prometheus.Desc, 0)

	log.Info("Loading fields for CallLeg status.")

	// general status metric fields
	status, err := e.client.TBStatus().GetStatus()
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}

	val := reflect.ValueOf(status.CallLegs)

	var fields []string

	for i := 0; i < val.Type().NumField(); i++ {
		fmt.Println(val.Type().Field(i).Tag.Get("json"))
		fields = append(fields, val.Type().Field(i).Tag.Get("json"))
	}

	clFields := fields
	for _, i := range clFields {
		log.Infof("Adding CallLeg field: %s", i)
		newDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace+"_"+e.id, "", i),
			fmt.Sprintf("CallLeg field: %s", i),
			nil, nil,
		)

		// for individual nap fields we will want to add a nap label before the field name
		metricDesc[i] = newDesc
	}

	// get nap names, and build metric descriptions for them as well
	// get naps, and load individual statistics
	naps, err := e.client.TBNaps().GetNames(e.config)
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}
	// cycle through naps, and get nap statistics
	for _, nap := range naps {
		napStatus, err := e.client.TBNaps().GetNapStatus(e.config, nap)
		if err != nil {
			return
		}

		// todo cycle through naps, get fields, and build according to nap name for later use/metrics calculations

		log.Infoln(napStatus.UsagePercent)
	}

	// update exporter descriptions w/ metrics map
	e.desc = metricDesc
}

type Exporter struct {
	client sbc.Client
	apiUri string
	id     string
	config string
	desc   map[string]*prometheus.Desc
}

func NewExporter(c sbc.Client, id string, config string) (*Exporter, error) {

	var e = &Exporter{
		client: c,
		id:     id,
		config: config,
	}

	e.BuildDescriptions()

	return e, nil

}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, i := range e.desc {
		ch <- i
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	// get status
	status, err := e.client.TBStatus().GetStatus()

	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}

	val := reflect.ValueOf(status.CallLegs)

	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Field(i)
		fieldName := val.Type().Field(i).Tag.Get("json")
		if field.Kind() == reflect.Int {
			ch <- prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, float64(field.Int()))
		} else if field.Kind() == reflect.Float64 {
			ch <- prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, field.Float())
		}
	}

	// get naps, and load individual statistics
	naps, err := e.client.TBNaps().GetNames(e.config)
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}
	// cycle through naps, and get nap statistics
	for _, nap := range naps {
		napStatus, err := e.client.TBNaps().GetNapStatus(e.config, nap)
		if err != nil {
			return
		}

		log.Infoln(napStatus.UsagePercent)
	}
}
