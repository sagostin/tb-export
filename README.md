https://docs.telcobridges.com/Tbstatus_monitoring
https://docs.telcobridges.com/Toolpack_Application:tbstatus

# TelcoBridges Prometheus Exporter (tb-export)

## Overview

The TelcoBridges Prometheus Exporter, or `tb-export`, is a monitoring tool designed for collecting live metrics from TelcoBridges session border controllers (SBCs). It's essential for retrieving real-time statistics associated with Network Access Points (NAPs) and other vital operational metrics. Running `tb-export` directly on the SBC is recommended to avoid latency issues in data collection and to ensure the most accurate monitoring experience.

## Prerequisites

- Access to a TelcoBridges session border controller.
- Prometheus server for metrics collection and storage.
- Network access configuration to allow `tb-export` communications through the SBC's firewall.

## Installation and Setup

1. **Obtain `tb-export`**: Download the exporter from the official source or repository.

2. **Deployment**: Transfer `tb-export` to your TelcoBridges SBC. This can be done via SCP, FTP, or any method suitable for your environment.

3. **Set Permissions**: Ensure `tb-export` is executable.

    ```bash
    chmod +x tb-export
    ```

4. **Firewall Configuration**: Adjust the SBC's firewall settings to allow `tb-export` to communicate with your Prometheus server. This typically involves allowing outbound connections on the port Prometheus uses to scrape metrics (default is 9090).

## Running `tb-export`

To run `tb-export`, simply execute it from the command line. No special configuration file is needed, making it straightforward to deploy and run.

```bash
./tb-export
```
Ensure that tb-export is running in an environment with sufficient permissions to access all necessary data on the SBC.
