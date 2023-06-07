https://docs.telcobridges.com/Tbstatus_monitoring
https://docs.telcobridges.com/Toolpack_Application:tbstatus

We'll need to run "tbstatus -gw 12358 /nap" to get the status of the gateway 12358.
Then from that we'll need to parse the data in the weird format of:


Regex to match the begining for the NAP stats
^\w*:\/nap:(\w*)$\n

Regex to Match normal value data
^\s{3}-\s(\w*)\s*(\w*)\s*$\n

Regex to match nap struct
^\s{5}\|-\s(.*)\s*(\w*)\s*$\n

regex values are in Multiline mode

```
[root@telcobridgespro ~]# tbstatus -gw 12358 /nap

The requested path is '/nap'.


0:/nap:BRIDGE_SBC2
BRIDGE_SBC2 -
- unique_id                             146
- signaling_type                        SIP
- reset_stats                           : No             (No,Yes)
- call_congestion                       false
- call_congestion_period_dropped_calls  0
- firewall_blocked                      false
- firewall_blocked_cnt                  0
- inst_incoming_call_cnt                0
- inst_incoming_call_cnt_in_progress    0
- inst_incoming_call_cnt_answered       0
- inst_incoming_call_cnt_terminating    0
- inst_incoming_call_rate               0
- inst_incoming_call_rate_highest       0
- inst_incoming_call_rate_accepted      0
- inst_incoming_call_rate_accepted_highest 0
- inst_incoming_call_rate_answered      0
- inst_incoming_call_rate_answered_highest 0
- inst_incoming_emergency_call_cnt      0
- inst_incoming_emergency_call_cnt_answered 0
- inst_incoming_emergency_call_rate     0
- inst_incoming_emergency_call_rate_highest 0
- inst_outgoing_call_cnt                0
- inst_outgoing_call_cnt_answered       0
- inst_outgoing_call_cnt_terminating    0
- inst_outgoing_call_rate               0
- inst_outgoing_call_rate_highest       0
- inst_outgoing_call_rate_answered      0
- inst_outgoing_call_rate_answered_highest 0
- inst_outgoing_call_rate_accepted      0
- inst_outgoing_call_rate_accepted_highest 0
- inst_incoming_interceptions           0
- inst_incoming_file_playbacks          0
- inst_incoming_file_recordings         0
- total_incoming_interceptions          0
- total_incoming_file_playbacks         0
- total_incoming_file_recordings        0
- inst_outgoing_interceptions           0
- inst_outgoing_file_playbacks          0
- inst_outgoing_file_recordings         0
- total_outgoing_interceptions          0
- total_outgoing_file_playbacks         0
- total_outgoing_file_recordings        0
- available_cnt                         1000
- unavailable_cnt                       0
- availability_percent                  100
- usage_percent                         0
- shared_usage_percent                  0
- low_delay_relay_shared_usage_percent  0
- mips_shared_usage_percent             0
- port_range_shared_usage_percent       0
- sip_shared_usage_percent              0
- reset_asr_stats                       : No             (No,Yes)
- asr_stats_incoming_struct                             
  |- global_asr_percent                 0               
  |- total_call_cnt                     0               
  |- total_accepted_call_cnt            0               
  |- total_answered_call_cnt            0               
  |- last_24h_asr_percent               0               
  |- last_24h_call_cnt                  0               
  |- current_hour_asr_percent           0               
  |- current_hour_call_cnt              0               
  |- last_hour_asr_percent              0               
  |- last_hour_call_cnt                 0
- asr_stats_outgoing_struct                             
  |- global_asr_percent                 83              
  |- total_call_cnt                     6               
  |- total_accepted_call_cnt            6               
  |- total_answered_call_cnt            5               
  |- last_24h_asr_percent               0               
  |- last_24h_call_cnt                  0               
  |- current_hour_asr_percent           0               
  |- current_hour_call_cnt              0               
  |- last_hour_asr_percent              0               
  |- last_hour_call_cnt                 0
- mos_struct                                            
  |- last_24h_ingress                   0.000           
  |- last_24h_egress                    0.000           
  |- current_hour_ingress               0.000           
  |- current_hour_egress                0.000           
  |- last_hour_ingress                  0.000           
  |- last_hour_egress                   0.000
- network_quality_struct                                
  |- last_24h_ingress                   0               
  |- last_24h_egress                    0               
  |- current_hour_ingress               0               
  |- current_hour_egress                0               
  |- last_hour_ingress                  0               
  |- last_hour_egress                   0
- availability_detection_struct                         
  |- poll_remote_proxy                  Yes             
  |- is_available                       Yes
- registration_struct                                   
  |- register_to_proxy                  No              
  |- registered                         No
- reset_rtp_stats                       : No             (No,Yes)
- rtp_statistics_struct                                 
  |- from_net_nb_packets                2468            
  |- from_net_nb_out_of_seq_packets     0               
  |- from_net_nb_lost_packets           0               
  |- from_net_nb_duplicate_packets      0               
  |- from_net_nb_early_late_packets     0               
  |- from_net_nb_bad_protocol_headers   0               
  |- from_net_nb_buffer_overflows       0               
  |- from_net_nb_other_errors           0               
  |- to_net_nb_packets                  2138            
  |- to_net_nb_arp_failures             0               
  |- t38_nb_pages_to_tdm                0               
  |- t38_nb_pages_from_tdm              0
- reset_nap_drop_stats                  : No             (No,Yes)
- local_drop_stats                                      
  |- TOTAL                              6               
  |- NORMAL_UNSPECIFIED (31)            1               
  |- TOOLPACK_NORMAL                    5
- remote_drop_stats
- system_drop_stats                                     


Application done.
```