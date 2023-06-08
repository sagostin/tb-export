package main

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"os"
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
	napValueStruct = "^\\s{5}\\|-\\s(\\w*)\\s*(\\w*)\\s*$"
	napStructTitle = "^\\s{3}-\\s(\\w*)\\s*$"
)

func GetStatusNAP(cli TbCliStatus) (map[string]*NapStatus, error) {
	cli.CommandPath = "/nap"

	/*out, err := cli.runStatusCmd()
	if err != nil {
		return nil, err
	}*/

	out, err := os.ReadFile("./out_test.txt")
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

	var temporaryNapStatus NapStatus // used to store the current nap status

	// store these values for later
	napStatuses := make(map[string]*NapStatus)
	var currentStruct string
	var currentNAP string

	// keep track of the previous line processed, ignore if it was blank, as well as keep the line number??
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {

		if strings.Contains(l, "local_drop_stats") ||
			strings.Contains(l, "remote_drop_stats") ||
			strings.Contains(l, "local_drop_stats") {
			log.Errorf("Found drop stats, ignoring until handled correctly")
			currentStruct = ""

			napStatuses[currentNAP] = &temporaryNapStatus
			log.Infoln("Added nap status to map")

			return napStatuses, nil
		}

		log.Warnf(l)

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
				log.Warnln("Found struct, entering struct mode")
				currentStruct = rNapStructTitle.FindAllStringSubmatch(l, -1)[0][1]
				log.Infof("Current struct is: %s", currentStruct)
				// todo reflect onto the struct
				continue
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
				// todo why would this be empty??
				return nil, errors.New("found struct data, but current struct is empty, ignoring")
			}

			fields := rNapValueStruct.FindAllStringSubmatch(l, -1)

			log.Infoln(fields)

			fieldName := fields[0][1]
			fieldValue := fields[0][2]

			log.Infof("Found field: %s with value: %s", fieldName, fieldValue)

			// todo reflect onto the struct

			/*nVal := reflect.ValueOf(temporaryNapStatus)
			err := updateField(fieldName, fieldValue, nVal)
			if err != nil {
				return nil, err
			}*/
			continue
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
					temporaryNapStatus = NapStatus{}
					currentNAP = napName
				}
				continue
			} else {
				_, ok := napStatuses[napName]
				// If the key exists
				if !ok {
					temporaryNapStatus = NapStatus{}
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

			log.Infof("Found field: %s with value: %s", fieldName, fieldValue)

			nVal := reflect.ValueOf(temporaryNapStatus)

			err := updateField(fieldName, fieldValue, nVal)
			if err != nil {
				return nil, err
			}

			temporaryNapStatus = nVal.Interface().(NapStatus)
			continue
		}

		napStatuses[currentNAP] = &temporaryNapStatus
	}

	return napStatuses, nil
}

func updateField(fieldName string, fieldValue string, nVal reflect.Value) error {
	tValid := nVal.Kind()
	if tValid == reflect.String {
		nVal.FieldByName(fieldName).SetString(fieldValue)
	} else if tValid == reflect.Int {
		parseInt, err := strconv.ParseInt(fieldValue, 10, 64)
		if err != nil {
			return err
		}
		nVal.FieldByName(fieldName).SetInt(parseInt)
	} else if tValid == reflect.Bool {
		parseBool, err := strconv.ParseBool(fieldValue)
		if err != nil {
			return err
		}
		nVal.FieldByName(fieldName).SetBool(parseBool)
	} else if tValid == reflect.Float64 {
		float, err := strconv.ParseFloat(fieldValue, 64)
		if err != nil {
			return err
		}
		nVal.FieldByName(fieldName).SetFloat(float)
	}
	return nil
}

/*

there's also this section that will need to be handled accordingly...

   - local_drop_stats
     |- TOTAL                              8
     |- TOOLPACK_NORMAL                    4
     |- TOOLPACK_SIGNALING_ERROR           4
   - remote_drop_stats
     |- TOTAL                              8
     |- NORMAL_CALL_CLEARING (16)          2
     |- 404_NOT_FOUND                      6
   - system_drop_stats
     |- TOTAL                              42
     |- TOOLPACK_SIGNALING_ERROR           41
     |- 488_NOT_ACCEPTBLE_HERE             1

*/

type NapStatus struct {
	AvailabilityDetectionStruct          AvailabilityDetectionStruct `json:"availability_detection_struct"`
	PortRangeSharedUsagePercent          int                         `json:"port_range_shared_usage_percent"`
	AvailableCnt                         int                         `json:"available_cnt"`
	InstIncomingCallCntTerminating       int                         `json:"inst_incoming_call_cnt_terminating"`
	InstIncomingCallCntAnswered          int                         `json:"inst_incoming_call_cnt_answered"`
	SignalingType                        string                      `json:"signaling_type"`
	TotalIncomingFilePlaybacks           int                         `json:"total_incoming_file_playbacks"`
	InstOutgoingCallCnt                  int                         `json:"inst_outgoing_call_cnt"`
	InstIncomingEmergencyCallCnt         int                         `json:"inst_incoming_emergency_call_cnt"`
	ResetAsrStats                        string                      `json:"reset_asr_stats"`
	InstOutgoingCallRate                 int                         `json:"inst_outgoing_call_rate"`
	InstIncomingCallRateAnswered         int                         `json:"inst_incoming_call_rate_answered"`
	InstIncomingCallRateAccepted         int                         `json:"inst_incoming_call_rate_accepted"`
	FirewallBlockedCnt                   int                         `json:"firewall_blocked_cnt"`
	ResetStats                           string                      `json:"reset_stats"`
	ResetNapDropStats                    string                      `json:"reset_nap_drop_stats"`
	AsrStatsIncomingStruct               AsrStatsIncomingStruct      `json:"asr_stats_incoming_struct"`
	UsagePercent                         int                         `json:"usage_percent"`
	TotalIncomingInterceptions           int                         `json:"total_incoming_interceptions"`
	InstIncomingFilePlaybacks            int                         `json:"inst_incoming_file_playbacks"`
	InstOutgoingCallCntAnswered          int                         `json:"inst_outgoing_call_cnt_answered"`
	InstIncomingEmergencyCallRateHighest int                         `json:"inst_incoming_emergency_call_rate_highest"`
	UniqueId                             int                         `json:"unique_id"`
	SystemDropStats                      struct {
	} `json:"system_drop_stats"`
	LocalDropStats struct {
	} `json:"local_drop_stats"`
	RemoteDropStats struct {
	} `json:"remote_drop_stats"`
	MosStruct                            MosStruct              `json:"mos_struct"`
	SipSharedUsagePercent                int                    `json:"sip_shared_usage_percent"`
	InstIncomingCallRateAnsweredHighest  int                    `json:"inst_incoming_call_rate_answered_highest"`
	InstIncomingCallCnt                  int                    `json:"inst_incoming_call_cnt"`
	TotalOutgoingFileRecordings          int                    `json:"total_outgoing_file_recordings"`
	InstOutgoingCallRateAnsweredHighest  int                    `json:"inst_outgoing_call_rate_answered_highest"`
	InstIncomingCallRate                 int                    `json:"inst_incoming_call_rate"`
	InstIncomingCallCntInProgress        int                    `json:"inst_incoming_call_cnt_in_progress"`
	AvailabilityPercent                  int                    `json:"availability_percent"`
	InstIncomingFileRecordings           int                    `json:"inst_incoming_file_recordings"`
	InstOutgoingCallRateAccepted         int                    `json:"inst_outgoing_call_rate_accepted"`
	FirewallBlocked                      bool                   `json:"firewall_blocked"`
	CallCongestionPeriodDroppedCalls     int                    `json:"call_congestion_period_dropped_calls"`
	RegistrationStruct                   RegistrationStruct     `json:"registration_struct"`
	NetworkQualityStruct                 NetworkQualityStruct   `json:"network_quality_struct"`
	AsrStatsOutgoingStruct               AsrStatsOutgoingStruct `json:"asr_stats_outgoing_struct"`
	InstOutgoingCallRateHighest          int                    `json:"inst_outgoing_call_rate_highest"`
	InstIncomingEmergencyCallRate        int                    `json:"inst_incoming_emergency_call_rate"`
	LowDelayRelaySharedUsagePercent      int                    `json:"low_delay_relay_shared_usage_percent"`
	TotalOutgoingInterceptions           int                    `json:"total_outgoing_interceptions"`
	InstOutgoingFilePlaybacks            int                    `json:"inst_outgoing_file_playbacks"`
	InstIncomingInterceptions            int                    `json:"inst_incoming_interceptions"`
	CallCongestion                       bool                   `json:"call_congestion"`
	MipsSharedUsagePercent               int                    `json:"mips_shared_usage_percent"`
	SharedUsagePercent                   int                    `json:"shared_usage_percent"`
	UnavailableCnt                       int                    `json:"unavailable_cnt"`
	InstOutgoingFileRecordings           int                    `json:"inst_outgoing_file_recordings"`
	InstOutgoingCallRateAnswered         int                    `json:"inst_outgoing_call_rate_answered"`
	InstOutgoingCallCntTerminating       int                    `json:"inst_outgoing_call_cnt_terminating"`
	InstIncomingEmergencyCallCntAnswered int                    `json:"inst_incoming_emergency_call_cnt_answered"`
	RtpStatisticsStruct                  RtpStatisticsStruct    `json:"rtp_statistics_struct"`
	ResetRtpStats                        string                 `json:"reset_rtp_stats"`
	TotalOutgoingFilePlaybacks           int                    `json:"total_outgoing_file_playbacks"`
	InstOutgoingInterceptions            int                    `json:"inst_outgoing_interceptions"`
	TotalIncomingFileRecordings          int                    `json:"total_incoming_file_recordings"`
	InstOutgoingCallRateAcceptedHighest  int                    `json:"inst_outgoing_call_rate_accepted_highest"`
	InstIncomingCallRateAcceptedHighest  int                    `json:"inst_incoming_call_rate_accepted_highest"`
	InstIncomingCallRateHighest          int                    `json:"inst_incoming_call_rate_highest"`
}

type MosStruct struct {
	CurrentHourEgress  float64 `json:"current_hour_egress"`
	LastHourEgress     float64 `json:"last_hour_egress"`
	CurrentHourIngress float64 `json:"current_hour_ingress"`
	LastHourIngress    float64 `json:"last_hour_ingress"`
	Last24HIngress     float64 `json:"last_24h_ingress"`
	Last24HEgress      float64 `json:"last_24h_egress"`
}

type RtpStatisticsStruct struct {
	FromNetNbOtherErrors        int `json:"from_net_nb_other_errors"`
	FromNetNbLostPackets        int `json:"from_net_nb_lost_packets"`
	T38NbPagesFromTdm           int `json:"t38_nb_pages_from_tdm"`
	FromNetNbBadProtocolHeaders int `json:"from_net_nb_bad_protocol_headers"`
	FromNetNbPackets            int `json:"from_net_nb_packets"`
	ToNetNbPackets              int `json:"to_net_nb_packets"`
	T38NbPagesToTdm             int `json:"t38_nb_pages_to_tdm"`
	ToNetNbArpFailures          int `json:"to_net_nb_arp_failures"`
	FromNetNbBufferOverflows    int `json:"from_net_nb_buffer_overflows"`
	FromNetNbOutOfSeqPackets    int `json:"from_net_nb_out_of_seq_packets"`
	FromNetNbEarlyLatePackets   int `json:"from_net_nb_early_late_packets"`
	FromNetNbDuplicatePackets   int `json:"from_net_nb_duplicate_packets"`
}

type AvailabilityDetectionStruct struct {
	PollRemoteProxy string `json:"poll_remote_proxy"`
	IsAvailable     string `json:"is_available"`
}

type AsrStatsOutgoingStruct struct {
	Last24HCallCnt        int `json:"last_24h_call_cnt"`
	Last24HAsrPercent     int `json:"last_24h_asr_percent"`
	TotalCallCnt          int `json:"total_call_cnt"`
	GlobalAsrPercent      int `json:"global_asr_percent"`
	LastHourCallCnt       int `json:"last_hour_call_cnt"`
	CurrentHourCallCnt    int `json:"current_hour_call_cnt"`
	TotalAnsweredCallCnt  int `json:"total_answered_call_cnt"`
	TotalAcceptedCallCnt  int `json:"total_accepted_call_cnt"`
	LastHourAsrPercent    int `json:"last_hour_asr_percent"`
	CurrentHourAsrPercent int `json:"current_hour_asr_percent"`
}

type AsrStatsIncomingStruct struct {
	Last24HCallCnt        int `json:"last_24h_call_cnt"`
	Last24HAsrPercent     int `json:"last_24h_asr_percent"`
	TotalCallCnt          int `json:"total_call_cnt"`
	GlobalAsrPercent      int `json:"global_asr_percent"`
	LastHourCallCnt       int `json:"last_hour_call_cnt"`
	CurrentHourCallCnt    int `json:"current_hour_call_cnt"`
	TotalAnsweredCallCnt  int `json:"total_answered_call_cnt"`
	TotalAcceptedCallCnt  int `json:"total_accepted_call_cnt"`
	LastHourAsrPercent    int `json:"last_hour_asr_percent"`
	CurrentHourAsrPercent int `json:"current_hour_asr_percent"`
}

type RegistrationStruct struct {
	Registered      string `json:"registered"`
	RegisterToProxy string `json:"register_to_proxy"`
}

type NetworkQualityStruct struct {
	CurrentHourEgress  int `json:"current_hour_egress"`
	LastHourEgress     int `json:"last_hour_egress"`
	CurrentHourIngress int `json:"current_hour_ingress"`
	LastHourIngress    int `json:"last_hour_ingress"`
	Last24HIngress     int `json:"last_24h_ingress"`
	Last24HEgress      int `json:"last_24h_egress"`
}
