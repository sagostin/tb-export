package main

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

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
	napValueStruct = "^\\s{5}\\|-\\s(\\w*)\\s*([0-9]*[.]?[0-9]+)\\s*$"
	napStructTitle = "^\\s{3}-\\s(\\w*)\\s*$"
)

func GetStatusNAP(cli TbCliStatus) (map[string]*NapStatus, error) {
	cli.CommandPath = "/nap"

	out, err := cli.runStatusCmd()
	if err != nil {
		return nil, err
	}

	/*out, err := os.ReadFile("./out_test.txt")
	if err != nil {
		return nil, err
	}*/

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
	napStatuses := make(map[string]*NapStatus)

	var currentStruct string
	var currentNAP string
	var insideStats bool

	// keep track of the previous line processed, ignore if it was blank, as well as keep the line number??
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {

		if strings.Contains(l, "local_drop_stats") ||
			strings.Contains(l, "remote_drop_stats") ||
			strings.Contains(l, "system_drop_stats") {
			log.Errorf("Found drop stats, ignoring until handled correctly")
			insideStats = true
			continue
		}

		//log.Warnf(l)

		// if the line contains "struct" we should be able to assume that we are now starting a struct within the status
		// for a nap, we will need to build and reflect onto that nap based on the provided lines

		if strings.Contains(l, "struct") {
			if insideStats {
				log.Info("Previously inside stats, changing to false, and continuing")
				insideStats = false
				continue
			}

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

			if insideStats {
				log.Info("Inside stats, ignoring fields")
				continue
			}

			if currentNAP == "" {
				log.Errorf("Current NAP is empty, cannot process struct")
				continue
			}

			if currentStruct == "" {
				// todo why would this be empty??
				return nil, errors.New("found struct data, but current struct is empty, ignoring")
			}

			fields := rNapValueStruct.FindAllStringSubmatch(l, -1)

			fieldName := fields[0][1]
			fieldValue := fields[0][2]

			vVal := reflect.ValueOf(napStatuses[currentNAP]).Elem()

			for i := 0; i < vVal.NumField(); i++ {
				f := vVal.Type().Field(i)
				if f.Tag.Get("json") == currentStruct {

					if f.Type.Kind() == reflect.Struct {
						// todo handle nested structs
						//log.Warnln("Found nested struct, todo handle")

						nVal := vVal.Field(i)
						for j := 0; j < nVal.NumField(); j++ {
							field := nVal.Type().Field(j)
							if field.Tag.Get("json") == fieldName {
								log.Infof("Found field: %s with value: %s, NAP: %s", fieldName, fieldValue, currentNAP)

								if field.Type.Kind() == reflect.Int {
									fieldValueInt, err := strconv.Atoi(fieldValue)
									if err != nil {
										log.Errorf("Failed to convert string to int: %s", err)
										continue
									}
									nVal.Field(j).SetInt(int64(fieldValueInt))
									break
								} else if field.Type.Kind() == reflect.String {
									nVal.Field(j).SetString(fieldValue)
									break
								} else if field.Type.Kind() == reflect.Bool {
									fieldValueBool, err := strconv.ParseBool(fieldValue)
									if err != nil {
										log.Errorf("Failed to convert string to bool: %s", err)
										continue
									}
									nVal.Field(j).SetBool(fieldValueBool)
									break
								} else if field.Type.Kind() == reflect.Float64 {
									fieldValueFloat, err := strconv.ParseFloat(fieldValue, 64)
									if err != nil {
										log.Errorf("Failed to convert string to float64: %s", err)
										continue
									}
									nVal.Field(j).SetFloat(fieldValueFloat)
									break
								} else if field.Type.Kind() == reflect.Struct {
									log.Errorf("Found unknown type: %s", field.Type.Kind())
									continue
								} else {
									log.Errorf("Found unknown type: %s", field.Type.Kind())
									continue
								}

								break
							}
						}
						break
					} else {
						continue
					}

					log.Infof("12 - Found field: %s with value: %s, NAP: %s", fieldName, fieldValue, currentNAP)
				}
			}

			// todo reflect onto the struct
			continue
		} else if rNapBeginning.MatchString(l) {
			napName := rNapBeginning.FindAllStringSubmatch(l, -1)[0][1]

			_, ok := napStatuses[napName]
			// If the key exists
			if !ok {
				napStatuses[napName] = &NapStatus{}
			} else {
				log.Errorf("NAP already exists, skipping")
			}
			currentNAP = napName
			continue
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

			log.Infof("Found field: %s with value: %s, NAP: %s", fieldName, fieldValue, currentNAP)

			nVal := reflect.ValueOf(napStatuses[currentNAP]).Elem()

			for i := 0; i < nVal.NumField(); i++ {
				field := nVal.Type().Field(i)
				if field.Tag.Get("json") == fieldName {

					if field.Type.Kind() == reflect.Int {
						fieldValueInt, err := strconv.Atoi(fieldValue)
						if err != nil {
							log.Errorf("Failed to convert string to int: %s", err)
							continue
						}
						nVal.Field(i).SetInt(int64(fieldValueInt))
						break
					} else if field.Type.Kind() == reflect.String {
						nVal.Field(i).SetString(fieldValue)
						break
					} else if field.Type.Kind() == reflect.Bool {
						fieldValueBool, err := strconv.ParseBool(fieldValue)
						if err != nil {
							log.Errorf("Failed to convert string to bool: %s", err)
							continue
						}
						nVal.Field(i).SetBool(fieldValueBool)
						break
					} else if field.Type.Kind() == reflect.Float64 {
						fieldValueFloat, err := strconv.ParseFloat(fieldValue, 64)
						if err != nil {
							log.Errorf("Failed to convert string to float64: %s", err)
							continue
						}
						nVal.Field(i).SetFloat(fieldValueFloat)
						break
					} else if field.Type.Kind() == reflect.Struct {
						log.Errorf("Found unknown type: %s", field.Type.Kind())
						continue
					} else {
						log.Errorf("Found unknown type: %s", field.Type.Kind())
						continue
					}

					log.Infof("11 - Found field: %s with value: %s, NAP: %s", fieldName, fieldValue, currentNAP)
					nVal.Field(i).Set(reflect.ValueOf(fieldValue))
					break
				}
			}

			continue
		}
	}
	return napStatuses, nil
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
