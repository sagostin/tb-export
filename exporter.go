package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strings"
	"sync"
)

const (
	namespace = "tb" // For Prometheus metrics.
)

func (e *Exporter) BuildDescriptions() {
	// build map for descriptions that is limitless >:) (not really)
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
		//fmt.Println(val.Type().Field(i).Tag.Get("json"))
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

	// ------ ^ that is the general status metric fields
	// below is the ones for the individual naps

	// get nap names, and build metric descriptions for them as well
	// get naps, and load individual statistics
	naps, err := e.client.TBNaps().GetNames(e.config)
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}

	// cycle through naps, and get nap statistics
	for _, nap := range naps {
		/*napStatus, err := e.client.TBNaps().GetNapStatus(e.config, nap)
		if err != nil {
			log.Errorf("Can't query Service API: %v", err)
			return
		}

		var nStatus sbc.NapStatus
		nStatus = *napStatus*/

		tempNapStatusFormat := sbc.NapStatus{}

		valFirst := reflect.ValueOf(tempNapStatusFormat)

		var napFields []string

		for i := 0; i < valFirst.Type().NumField(); i++ {
			//fmt.Println(valFirst.Type().Field(i).Tag.Get("json"))
			nF := strings.Replace(valFirst.Type().Field(i).Tag.Get("json"), ",omitempty", "", -1)
			napFields = append(napFields, nF)
		}

		nFields := napFields
		for _, i := range nFields {
			if nap == "" {
				log.Errorf("Nap is empty, skipping")
				continue
			}

			// todo recursively go through each nap field and go into the individual structs and get those fields as well

			log.Infof("Adding NAP field: %s %s", nap, i)
			newDesc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace+"_"+e.id, "", i),
				fmt.Sprintf("NAP field: %s %s", nap, i),
				nil, nil,
			)

			// for individual nap fields we will want to add a nap label before the field name
			metricDesc[nap+"-"+i] = newDesc
		}

		// some of the fields are structs inside of structs, how to navigate this?
		// todo cycle through naps, get fields, and build according to nap name for later use/metrics calculations

		/*log.Infoln(napStatus.UsagePercent)*/
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

	var wg sync.WaitGroup

	// cycle through naps, and get nap statistics
	for _, nap := range naps {
		if nap == "" {
			log.Infoln("Nap is empty, skipping")
			continue
		}

		go func(cCh chan<- prometheus.Metric, n string, client sbc.Client, config string) {
			wg.Add(1)
			defer wg.Done()
			napStatus, err := client.TBNaps().GetNapStatus(config, n)
			if err != nil {
				log.Errorf("Can't query Service API: %v", err)
				return
			}
			var nStatus sbc.NapStatus
			nStatus = *napStatus

			nVal := reflect.ValueOf(nStatus)

			for i := 0; i < nVal.Type().NumField(); i++ {
				field := nVal.Field(i)

				// remove omitempty from json tag
				fieldName := strings.Replace(nVal.Type().Field(i).Tag.Get("json"), ",omitempty", "", -1)
				if field.Kind() == reflect.Int {
					log.Infoln("NAP field: ", n, fieldName)
					cCh <- prometheus.MustNewConstMetric(e.desc[n+"-"+fieldName], prometheus.GaugeValue, float64(field.Int()), n)
				} else if field.Kind() == reflect.Float64 {
					log.Infoln("NAP field: ", n, fieldName)
					cCh <- prometheus.MustNewConstMetric(e.desc[n+"-"+fieldName], prometheus.GaugeValue, field.Float(), n)
				} else {
					log.Errorf("Unknown field type: %s", fieldName)
				}
			}
		}(ch, nap, e.client, e.config)
		// cycle through the naps, then initialize a go func for each one running in the background,
		// when that func is complete, had it send to channel, and same for the others
	}
	wg.Wait()
}
