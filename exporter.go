package main

import (
	"encoding/json"
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

	/*log.Info("Loading fields for CallLeg status.")

	// todo do i even need to use this old status stuff?
	// can i convert to use just the direct pulling from the tbstatus??

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

	labels := []string{"id"}

	clFields := fields
	for _, i := range clFields {
		log.Infof("Adding CallLeg field: %s", i)
		newDesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "status", i),
			fmt.Sprintf("CallLeg field: %s", i),
			labels, nil,
		)

		// for individual nap fields we will want to add a nap label before the field name
		metricDesc[i] = newDesc
	}*/

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
		nF1 := strings.Replace(valFirst.Type().Field(i).Tag.Get("json"), ",omitempty", "", -1)
		if valFirst.Type().Field(i).Type.Kind() == reflect.Struct {
			//fmt.Println("Struct: ", valFirst.Type().Field(i).Type)
			//fmt.Println("Struct: ", valFirst.Type().Field(i).Type.NumField())
			for j := 0; j < valFirst.Type().Field(i).Type.NumField(); j++ {
				//fmt.Println("Struct: ", valFirst.Type().Field(i).Type.Field(j).Tag.Get("json"))
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
	nap, err := GetStatusNAP(e.tbCliStatus)
	if err != nil {
		return
	}
	var wg sync.WaitGroup

	for k, v := range nap {
		log.Warnf("NAP: %s", k)
		marshal, err := json.Marshal(v)
		if err != nil {
			return
		}
		marshalString := string(marshal)
		log.Warnf(marshalString)
	}

	log.Infoln("Waiting for all goroutines to finish")
	wg.Wait()

	// get status
	/*status, err := e.client.TBStatus().GetStatus()

	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}

	val := reflect.ValueOf(status.CallLegs)

	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Field(i)
		fieldName := val.Type().Field(i).Tag.Get("json")
		if field.Kind() == reflect.Int {
			ch <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, float64(field.Int()), e.id))
		} else if field.Kind() == reflect.Float64 {
			ch <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, field.Float(), e.id))
		}
	}*/

	/*// Create a rate limiter
	limiter := rate.NewLimiter(rate.Limit(maxRequests), burst)

	// todo make this more efficient
	// get naps, and load individual statistics
	naps, err := e.client.TBNaps().GetNames(e.config)
	if err != nil {
		log.Errorf("Can't query Service API: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, nap := range naps {
		if nap == "" {
			log.Infoln("Nap is empty, skipping")
			continue
		}

		// Apply rate limiting before making each API request
		if err := limiter.Wait(context.Background()); err != nil {
			log.Errorf("Rate limit exceeded: %v", err)
			time.Sleep(minDelay) // Wait for the minimum delay before retrying
			continue
		}
		wg.Add(1)
		go func(cCh chan<- prometheus.Metric, n string, client sbc.Client, config string) {
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
					//e.desc[n+"-"+fieldName]
					cCh <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, float64(field.Int()), n, e.id))
				} else if field.Kind() == reflect.Float64 {
					log.Infoln("NAP field: ", n, fieldName)
					cCh <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName], prometheus.GaugeValue, field.Float(), n, e.id))
				} else if field.Kind() == reflect.Struct {
					log.Infoln("NAP field: ", n, fieldName)
					for i2 := 0; i2 < field.NumField(); i++ {
						field2 := field.Field(i2)
						fieldName2 := field.Type().Field(i2).Tag.Get("json")
						if field2.Kind() == reflect.Int {
							cCh <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName+"__"+fieldName2], prometheus.GaugeValue, float64(field2.Int()), n, e.id))
						} else if field2.Kind() == reflect.Float64 {
							cCh <- prometheus.NewMetricWithTimestamp(time.Now(), prometheus.MustNewConstMetric(e.desc[fieldName+"__"+fieldName2], prometheus.GaugeValue, field2.Float(), n, e.id))
						}
					}
				} else {
					log.Errorf("Unknown field type: %s", fieldName)
				}
			}
		}(ch, nap, e.client, e.config)
		// cycle through the naps, then initialize a go func for each one running in the background,
		// when that func is complete, had it send to channel, and same for the others
	}
	wg.Wait()
	log.Infoln("Done collecting metrics")*/
}
