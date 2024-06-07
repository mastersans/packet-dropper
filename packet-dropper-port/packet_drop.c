#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <bpf/bpf_helpers.h>
#include <arpa/inet.h>

// Per-CPU array to count dropped packets
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __type(key, __u32);
    __type(value, __u16);
    __uint(max_entries, 1000);
} drop_counter SEC(".maps");

// Array to store allowed TCP ports
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u16);
    __uint(max_entries, 1000);
} allowed_ports SEC(".maps");

SEC("xdp") int xdp_packet_filter(struct xdp_md *ctx)
{
    void *data_start = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth_hdr = data_start;

    // Ensure packet contains Ethernet header
    if ((void *)(eth_hdr + 1) > data_end) {
        return XDP_DROP;
    }

    // Filter for IPv4 packets
    if (eth_hdr->h_proto != htons(ETH_P_IP)) {
        return XDP_PASS;
    }

    struct iphdr *ip_hdr = (struct iphdr *)(eth_hdr + 1);

    // Ensure packet contains IP header
    if ((void *)(ip_hdr + 1) > data_end) {
        return XDP_DROP;
    }

    // Filter for TCP packets
    if (ip_hdr->protocol != IPPROTO_TCP) {
        return XDP_PASS;
    }

    struct tcphdr *tcp_hdr = (struct tcphdr *)(ip_hdr + 1);

    // Ensure packet contains TCP header
    if ((void *)(tcp_hdr + 1) > data_end) {
        return XDP_DROP;
    }

    __u32 zero_key = 0;
    __u16 *allowed_port_ptr = bpf_map_lookup_elem(&allowed_ports, &zero_key);

    // Allow all if no port is specified in the map
    if (!allowed_port_ptr) {
        return XDP_PASS;
    }

    __u16 dest_port = ntohs(tcp_hdr->dest);
    __u16 allowed_port = *allowed_port_ptr;

    // Drop packets destined to the specified port
    if (dest_port == allowed_port) {
        __u16 *drop_count_ptr = bpf_map_lookup_elem(&drop_counter, &zero_key);
        if (drop_count_ptr) {
            (*drop_count_ptr)++;
        }
        return XDP_DROP;
    }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
