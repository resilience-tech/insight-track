# Synthetic Web Page Scanning
## Overview
This system is designed to automate the synthetic scanning of web pages. It utilizes Azure Functions to trigger Chrome browser scans and produces various outputs such as HAR files, screenshots, performance profiles, and memory dumps. The scan results are then sent to a ServiceBus Topic for further processing.

## Architecture

![System Architecture](./is.jpg)

## Synthetic Scan Process

1. **Triggering the Scan**: The scan can be triggered by messages received in the ServiceBus Topic. 
2. **Executing the Scan**: The Azure Function, utilizing a headless Chrome browser, navigates to the specified web page and performs the scan.
3. **Generating Artifacts**: During the scan, the function generates various artifacts:
   - **HAR File**: Captures all network requests and responses.
   - **Screenshots**: Takes screenshots of the web page at different stages.
   - **Performance Profile**: A JSON file containing detailed performance metrics.
   - **Memory Dump**: Provides a snapshot of the memory used by the browser during the scan.
4. **Storing Results**: The results are stored in an azure storage account with blob containers.
5. **Publishing to ServiceBus**: The scan results are sent to a ServiceBus Topic for further processing, notifications, or integration with other systems.

### Components

1. **Scan Trigger**: An initial trigger (such as an email or message from ServiceBus) that initiates the scan process.
2. **Azure Function**: The core component that uses a Chrome browser instance to perform the scan.
3. **HAR (HTTP Archive)**: Captures network waterfall and dependency information of the scanned web page.
4. **Screenshots**: Takes visual snapshots of the web page during the scan.
5. **Performance Profile**: Records the performance metrics of the web page.
6. **Memory Dump**: Captures the memory usage data of the web page.
7. **ServiceBus Topic**: Receives and handles the scan results for further processing, alerts or integrations.
