# Approach for Assignment 2
Sorry for the delay. I was working towards coming up with a working solution for assignment 2. My apologies for not being able to complete it. Nonetheless, I still want to present an approach that I came up with while learning about BPF.



## Kernel Space Implementation

### 1. Header Parsing
- **Extract Ethernet, IP, and TCP headers** from incoming packets to access necessary information such as source/destination IP addresses and TCP ports.

### 2. Process Identification
- **Retrieve the process ID (pid) and current executable name (program_name)** associated with the packet using functions like `bpf_get_current_pid_tgid()` and `bpf_get_current_comm()`.

### 3. Comparison and Filtering
- **Compare the `program_name` with the expected process name ("myprocess")**.
  - If they match:
    - **Check if the destination TCP port (`th->dest`) matches the specified port** (default: 4040).
    - **Allow packets destined for port 4040** to pass through (`BPF_OK`).
    - If the port does not match, **increment a drop counter** (`drop_counter`) and **drop packets that do not match** (`BPF_DROP`).

### 4. Map Usage
- **Utilize eBPF maps** (`port_map` and `process_name_map`) to store and retrieve allowed port and process name configurations between the User Space and Kernel Space.

### 5. Program Attachment
- **Attach the eBPF program to the cgroup ingress hook**. This is achieved by using the `SEC("cgroup_skb/ingress")` section, ensuring the program processes incoming packets on the ingress path.

## User Space Implementation (Go)

### 1. Argument Parsing
- **Parse command-line arguments** to specify the TCP port (`portNum`, defaulting to 4040) and process name (`processName`, defaulting to "myprocess").

- **Use the `github.com/cilium/ebpf` library** to Load eBPF objects (`PortMap`, `ProcessNameMap`, `BlockProcessPorts`).

### 2. Map Updates
- **Update eBPF maps** (`PortMap` and `ProcessNameMap`) with the specified `portNum` and `processName` using the functions provided by the `github.com/cilium/ebpf` library.

### 3. Cgroup Attachment
- **Attach the `BlockProcessPorts` eBPF program to the ingress path** of the cgroup associated with the specified process.
- Maybe utilize `link.AttachCgroup()` from `github.com/cilium/ebpf/link` to achieve this integration.

### 4. Signal Handling
- Implement signal handling for `SIGINT` and `SIGTERM` to shut down the application.
- Ensure cleanup of resources, such as detaching the eBPF program from the cgroup.


