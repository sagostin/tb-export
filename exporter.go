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

	e.desc = metricDesc
}

type Exporter struct {
	client sbc.Client
	apiUri string
	id     string
	desc   map[string]*prometheus.Desc
}

func NewExporter(c sbc.Client, id string) (*Exporter, error) {

	var e = &Exporter{
		client: c,
		id:     id,
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

	//ch <- prometheus.MustNewConstMetric(e.desc["outgoing_legs"], prometheus.GaugeValue, float64(status.CallLegs.OutgoingLegs))

	/*napStatus, err := e.client.TBNaps().GetNapStatus("Active Config", "pbx_TopsMX")
	if err != nil {
		log.Error(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(napAsrTotalCallCnt, prometheus.GaugeValue, float64(napStatus.AsrStatsIncomingStruct.TotalAnsweredCallCnt))
	*/
}
