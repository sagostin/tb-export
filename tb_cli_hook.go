package main

import (
	"context"
	"errors"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

/*

- Create a new file tb_cli_hook.go
- First connect to the system, dump the general statistics
- Then connect to each nap, dump the nap statistics through using the tbstatus commands
- This will significantly reduce time complexity?!??!??!

*/

// todo
// initialize the exporter to build the descriptions for the fields and be able to interpreate
// the fucking ugly format that telcobridges outputs, and seen as i dont wanna parse the damn csv files each time
// and listen to the changes, this seems easier...

type TbCliStatus struct {
	Gateway     string
	CommandPath string
}

func (cli *TbCliStatus) runStatusCmd() ([]byte, error) {
	osDetect := runtime.GOOS

	if osDetect == "linux" {
		log.Info("Running on Linux machine... continuing")
	} else {
		log.Fatal("This exporter is only supported on linux, please run on a linux machine")
	}

	var cmd *exec.Cmd

	args := []string{"-c", "tbstatus -gw " + cli.Gateway + " " + cli.CommandPath}
	cmd = exec.CommandContext(context.TODO(), "/bin/bash", args...)
	out, err := cmd.CombinedOutput()

	return out, err
}

const (
	napBeginning   = "^\\w*:\\/nap:(\\w*)$"
	napValueNormal = "^\\s{3}-\\s(\\w*)\\s*(\\w*)\\s*$"
	napValueStruct = "^\\s{5}\\|-\\s(.*)\\s*(\\w*)\\s*$*/"
	napStructTitle = "^\\s{3}-\\s(\\w*)\\s*"
)

func GetStatusNAP(cli TbCliStatus) (map[string]*sbc.NapStatus, error) {
	cli.CommandPath = "/nap"

	out, err := cli.runStatusCmd()
	if err != nil {
		return nil, err
	}

	// check empty data??
	if len(out) <= 0 {
		return nil, err
	}

	// precompile the regex expressions
	rNapBeginning, err := regexp.Compile(napBeginning)
	if err != nil {
		return nil, err
	}

	rNapValueNorm, err := regexp.Compile(napValueNormal)
	if err != nil {
		return nil, err
	}

	rNapValueStruct, err := regexp.Compile(napValueStruct)
	if err != nil {
		return nil, err
	}
	rNapStructTitle, err := regexp.Compile(napStructTitle)
	if err != nil {
		return nil, err
	}

	// store these values for later
	napStatuses := make(map[string]*sbc.NapStatus)
	var currentStruct string
	var currentNAP string

	// keep track of the previous line processed, ignore if it was blank, as well as keep the line number??
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		//log.Warnf(l)

		// if the line contains "struct" we should be able to assume that we are now starting a struct within the status
		// for a nap, we will need to build and reflect onto that nap based on the provided lines

		if strings.Contains(l, "struct") {
			if currentStruct != "" {
				log.Errorf("Current struct is not empty, assuming it can be overwritten")
			}

			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}
			if rNapStructTitle.MatchString(l) {
				log.Infoln("Found struct, entering struct mode")
				currentStruct = rNapStructTitle.FindAllStringSubmatch(l, -1)[0][1]
				continue
			} else {
				return nil, err
			}
			// mark it as entered the struct
			// get the map name from the
		} else if currentStruct != "" && rNapValueStruct.MatchString(l) {
			// reflect based on current struct, to parse the next data, and append to built struct
			// todo build out the inside struct data

			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}

			if currentStruct == "" {
				return nil, errors.New("found struct data, but current struct is empty, ignoring")
			}

			fieldName := rNapValueNorm.FindAllStringSubmatch(l, -1)[0][1]
			fieldValue := rNapValueNorm.FindAllStringSubmatch(l, -1)[0][2]

			status := napStatuses[currentNAP]
			sVal := reflect.ValueOf(status).Elem()

			nVal := sVal.FieldByName(currentStruct)
			if nVal.Kind() != reflect.Struct {
				return nil, errors.New("field is not a struct, cannot parse")
			}

			tValid := nVal.FieldByName(fieldName).Kind() // check

			if tValid == reflect.String {
				nVal.FieldByName(fieldName).SetString(fieldValue)
			} else if tValid == reflect.Int {
				parseInt, err := strconv.ParseInt(fieldValue, 10, 64)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetInt(parseInt)
			} else if tValid == reflect.Bool {
				parseBool, err := strconv.ParseBool(fieldValue)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetBool(parseBool)
			} else if tValid == reflect.Float64 {
				float, err := strconv.ParseFloat(fieldValue, 64)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetFloat(float)
			}

		} else if rNapBeginning.MatchString(l) {

			// increment for each line, if the line contains the beginning of the nap section, then we know that the next line is the nap name
			// it's safe to assume it's the first array inside of array as we're only processing a single line

			// find the nap name, after we've confirmed the line matches
			napName := rNapBeginning.FindAllStringSubmatch(l, -1)[0][1]

			if currentNAP != "" && currentNAP != napName {
				log.Errorf("Current NAP is not equal to the nap name found, completing nap and moving to next")

				_, ok := napStatuses[napName]
				// If the key exists
				if !ok {
					newNapStatus := &sbc.NapStatus{}
					napStatuses[napName] = newNapStatus
					currentNAP = napName
				}
				continue
			} else {
				_, ok := napStatuses[napName]
				// If the key exists
				if !ok {
					newNapStatus := &sbc.NapStatus{}
					napStatuses[napName] = newNapStatus
					currentNAP = napName
				} else {
					log.Errorf("NAP already exists, skipping wtf??!??")
				}
				continue
			}
			// todo check if the line is empty?? or do we just skip those??
		} else if rNapValueNorm.MatchString(l) {
			// if normal values match, and it *was* in struct mode, remove struct mode and resume.
			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}

			if currentStruct != "" {
				log.Warn("Found normal value, but was in struct mode, exiting struct mode")
				currentStruct = ""
			}

			fieldName := rNapValueNorm.FindAllStringSubmatch(l, -1)[0][1]
			fieldValue := rNapValueNorm.FindAllStringSubmatch(l, -1)[0][2]

			status := napStatuses[currentNAP]
			nVal := reflect.ValueOf(status).Elem()
			tValid := nVal.FieldByName(fieldName).Kind() // check

			if tValid == reflect.String {
				nVal.FieldByName(fieldName).SetString(fieldValue)
			} else if tValid == reflect.Int {
				parseInt, err := strconv.ParseInt(fieldValue, 10, 64)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetInt(parseInt)
			} else if tValid == reflect.Bool {
				parseBool, err := strconv.ParseBool(fieldValue)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetBool(parseBool)
			} else if tValid == reflect.Float64 {
				float, err := strconv.ParseFloat(fieldValue, 64)
				if err != nil {
					return nil, err
				}
				nVal.FieldByName(fieldName).SetFloat(float)
			}
		}

		// todo stupid fucking reflections with updating the struct values

		napStatuses[currentNAP] = st.Interface().(*sbc.NapStatus)
	}
	// todo grabs all nap statuses /nap

	return napStatuses, nil
}

func SystemStatus(gw int) {
	// todo grabs all system stats, /system/*
	// todo
}
