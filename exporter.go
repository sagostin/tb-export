package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	namespace = "tb" // For Prometheus metrics.
)

func (e *Exporter) BuildDescriptions() {
	// build map for descriptions that is limitless >:) (not really)
	var metricDesc = make(map[string]*prometheus.Desc, 0)

	tempNapStatusFormat := sbc.NapStatus{}

	valFirst := reflect.ValueOf(tempNapStatusFormat)

	var napFields []string

	for i := 0; i < valFirst.Type().NumField(); i++ {
		//fmt.Println(valFirst.Type().Field(i).Tag.Get("json"))
		nF1 := strings.Replace(valFirst.Type().Field(i).Tag.Get("json"), ",omitempty", "", -1)
		if valFirst.Type().Field(i).Type.Kind() == reflect.Struct {
			fmt.Println("Struct 1: ", valFirst.Type().Field(i).Type)
			fmt.Println("Struct 2: ", valFirst.Type().Field(i).Type.NumField())
			for j := 0; j < valFirst.Type().Field(i).Type.NumField(); j++ {
				fmt.Println("Struct 3: ", valFirst.Type().Field(i).Type.Field(j).Tag.Get("json"))
				nF2 := strings.Replace(valFirst.Type().Field(i).Type.Field(j).Tag.Get("json"), ",omitempty", "", -1)
				napFields = append(napFields, nF1+"__"+nF2)
			}
			continue
		} else {
			napFields = append(napFields, nF1)
		}
	}

	nFields := napFields
	for _, i := range nFields {
		_, ok := metricDesc[i]
		// If the key exists
		if ok {
			continue
		}

		// todo recursively go through each nap field and go into the individual structs and get those fields as well

		labels := []string{"nap", "id"}

		newDesc := prometheus.NewDesc(
			// subsystem: "_" + nap + "_"
			prometheus.BuildFQName(namespace, "nap", i),
			fmt.Sprintf("NAP field: %s", i),
			labels, nil,
		)

		// for individual nap fields we will want to add a nap label before the field name
		// metricDesc[nap+"-"+i] = newDesc
		metricDesc[i] = newDesc
	}

	// some of the fields are structs inside of  structs, how to navigate this?
	// todo cycle through naps, get fields, and build according to nap name for later use/metrics calculations

	/*log.Infoln(napStatus.UsagePercent)*/

	// update exporter descriptions w/ metrics map
	e.desc = metricDesc
}

type Exporter struct {
	client      sbc.Client
	apiUri      string
	id          string
	config      string
	desc        map[string]*prometheus.Desc
	tbCliStatus TbCliStatus
}

func NewExporter(c sbc.Client, id string, config string, status TbCliStatus) (*Exporter, error) {

	var e = &Exporter{
		client:      c,
		id:          id,
		config:      config,
		tbCliStatus: status,
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
	naps, err := GetStatusNAP(e.tbCliStatus)
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}

	// todo make this more efficient
	// get naps, and load individual statistics for each nap
	var wg sync.WaitGroup
	for n, nap := range naps {
		wg.Add(1)
		go func(cCh chan<- prometheus.Metric, nStatus *NapStatus, nap string, client sbc.Client, config string) {
			defer wg.Done()
			nVal := reflect.ValueOf(nStatus).Elem()

			for i := 0; i < nVal.NumField(); i++ {
				field := nVal.Type().Field(i)

				// remove omitempty from json tag
				fieldName := strings.Replace(nVal.Type().Field(i).Tag.Get("json"), ",omitempty", "", -1)
				if field.Type.Kind() == reflect.Int {
					log.Infoln("NAP field: ", n, fieldName)
					//e.desc[n+"-"+fieldName]
					cCh <- prometheus.NewMetricWithTimestamp(time.Now(),
						prometheus.MustNewConstMetric(e.desc[fieldName],
							prometheus.GaugeValue, float64(nVal.Field(i).Int()), nap, e.id))
				} else if field.Type.Kind() == reflect.Float64 {
					log.Infoln("NAP field: ", n, fieldName)
					cCh <- prometheus.NewMetricWithTimestamp(time.Now(),
						prometheus.MustNewConstMetric(e.desc[fieldName],
							prometheus.GaugeValue, nVal.Field(i).Float(), nap, e.id))
				} else if field.Type.Kind() == reflect.Struct {
					log.Infoln("NAP field: ", n, fieldName)
					log.Warnf("Struct: %s", fieldName)
					vVal := nVal.Field(i)
					for i2 := 0; i2 < vVal.NumField(); i2++ {
						field2 := vVal.Type().Field(i2)
						fieldName2 := field2.Tag.Get("json")
						if field2.Type.Kind() == reflect.Int {
							cCh <- prometheus.NewMetricWithTimestamp(time.Now(),
								prometheus.MustNewConstMetric(e.desc[fieldName+"__"+fieldName2],
									prometheus.GaugeValue, float64(vVal.Field(i2).Int()), nap, e.id))
						} else if field2.Type.Kind() == reflect.Float64 {
							cCh <- prometheus.NewMetricWithTimestamp(time.Now(),
								prometheus.MustNewConstMetric(e.desc[fieldName+"__"+fieldName2],
									prometheus.GaugeValue, vVal.Field(i2).Float(), nap, e.id))
						} else {
							log.Infoln("NAP field: ", n, fieldName, fieldName2)
							continue
						}
					}
				} else {
					log.Errorf("Unknown field type: %s", field.Type.Kind())
					continue
				}
			}
		}(ch, nap, n, e.client, e.config)
	}
	wg.Wait()
	log.Infoln("Done collecting nap metrics")
}
