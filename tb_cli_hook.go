package main

import (
	"context"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"os/exec"
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
	Gateway     int
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

	args := []string{"-c", "tbstatus -gw " + strconv.Itoa(cli.Gateway) + " " + cli.CommandPath}
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

func GetStatusNAP(cli TbCliStatus) map[string]sbc.NapStatus {
	out, err := cli.runStatusCmd()
	if err != nil {
		log.Errorf(err.Error())
	}

	// check empty data??
	if len(out) <= 0 {
		log.Errorf(err.Error())
	}

	// precompile the regex expressions
	rNapBeginning, err := regexp.Compile(napBeginning)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}

	rNapValueNorm, err := regexp.Compile(napValueNormal)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}

	rNapValueStruct, err := regexp.Compile(napValueStruct)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	rNapStructTitle, err := regexp.Compile(napStructTitle)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}

	napStatuses := make(map[string]sbc.NapStatus)
	var currentStruct string
	var currentNAP string

	// keep track of the previous line processed, ignore if it was blank, as well as keep the line number??
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {

		// if the line contains "struct" we should be able to assume that we are now starting a struct within the status
		// for a nap, we will need to build and reflect onto that nap based on the provided lines

		if strings.Contains(l, "struct") {
			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}
			if rNapStructTitle.MatchString(l) {
				log.Infoln("Found struct, entering struct mode")
				currentStruct = rNapStructTitle.FindAllStringSubmatch(l, -1)[0][1]
				continue
			} else {
				log.Fatal("Found struct, but did not match struct title, exiting")
			}
			// mark it as entered the struct
			// get the map name from the
		} else if currentStruct != "" && rNapValueStruct.MatchString(l) {
			// reflect based on current struct, to parse the next data, and append to built struct
		} else if rNapBeginning.MatchString(l) {

			// increment for each line, if the line contains the beginning of the nap section, then we know that the next line is the nap name
			// it's safe to assume it's the first array inside of array as we're only processing a single line

			// find the nap name, after we've confirmed the line matches
			napName := rNapValueNorm.FindAllStringSubmatch(l, -1)[0][1]

			_, ok := napStatuses[napName]
			// If the key exists
			if !ok {
				newNapStatus := sbc.NapStatus{}
				napStatuses[napName] = newNapStatus
				currentNAP = napName
			}

			// todo

			// once we've confirmed we've entered the statistics of the nap, we can start processing the lines
			// to process the lines, we need to track the nap name, and the line number
			// if the nap name changes and the line number was greater than before, we can update the nap name,
			// as well as build the struct for those values as well as the nap name

			// todo check if the line is empty?? or do we just skip those??
		} else if rNapValueNorm.MatchString(l) {
			// if normal values match, and it *was* in struct mode, remove struct mode and resume.

			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}
			// todo
		}
	}
	// todo grabs all nap statuses /nap
}

func SystemStatus(gw int) {
	// todo grabs all system stats, /system/*
	// todo
}
