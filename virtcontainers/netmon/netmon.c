/*
 * Copyright (C) 2018 Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <sched.h>
#include <asm/types.h>
#include <sys/socket.h>
#include <net/if.h>
#include <netinet/in.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <arpa/inet.h>
#include <ifaddrs.h>
#include <netdb.h>
#include <sys/ioctl.h>
#include <syslog.h>
#include <getopt.h>

#define PROGRAM_NAME    "netns-watcher"
#define PROGRAM_VERSION "0.0.1"

#ifndef MAX_IFACES
#define MAX_IFACES  50
#endif

#define INIT_IF_IDX -1

bool debug = false;

struct watcher_params {
	char *sandbox_id;
	char *runtime_path;
	char *netns_path;
};

struct ip_addr {
	unsigned char  family;
	char           *addr;
	struct ip_addr *next;
};

struct iface {
	int            idx;
	char           *hw_addr;
	char           *name;
	char           *mtu;
	struct ip_addr *ip_addrs;
};

struct route {
	char *src;
	char *dst;
	char *gw;
	char *dev;
};

/* 
 * Global interface list used to keep track of the interfaces and their
 * parameters.
 */
struct iface iface_list[MAX_IFACES];

/*
 * Print full list of interfaces.
 */
void print_iface_list() {
	int i;

	int max_ifaces = (int)(sizeof(iface_list)/sizeof(iface_list[0]));
	for (i = 0; i < max_ifaces; i++) {
		if (iface_list[i].idx == INIT_IF_IDX) {
			continue;
		}

		printf("IFACE %d\n", i);
		printf("\tidx     = %d\n", iface_list[i].idx);
		printf("\thw_addr = %s\n", iface_list[i].hw_addr);
		printf("\tname    = %s\n", iface_list[i].name);
		printf("\tmtu     = %s\n", iface_list[i].mtu);

		struct ip_addr *ipa = iface_list[i].ip_addrs;
		if (!ipa) {
			printf("\n");
			continue;
		}

		printf("\tip_addrs\n");
		printf("\t    |---- ip_addr = %s, family = %d\n",
				ipa->addr, ipa->family);
		while (ipa->next) {
			printf("\t    |---- ip_addr = %s, family = %d\n",
					ipa->next->addr, ipa->next->family);
			ipa = ipa->next;
		} 

		printf("\n");
	}
}

/*
 * Print version information.
 */
void print_version(void)
{
	printf("%s v%s\n", PROGRAM_NAME, PROGRAM_VERSION);
}

/*
 * Free internal fields of the watcher_params structure.
 */
void free_watcher_params(struct watcher_params *params)
{
	if (!params) {
		return;
	}

	free(params->netns_path);
	params->netns_path = NULL;

	free(params->sandbox_id);
	params->sandbox_id = NULL;

	free(params->runtime_path);
	params->runtime_path = NULL;
}

/*
 * Free internal fields of the ip_addr structure.
 */
void free_ip_addr(struct ip_addr *ipa) {
	struct ip_addr *ipa_next, *ipa_curr;

	if (!ipa) {
		return;
	}

	free(ipa->addr);
	ipa->addr = NULL;

	for (ipa_curr = ipa->next; ipa_curr; ipa_curr = ipa_next) {
		ipa_next = ipa_curr->next;

		free(ipa_curr->addr);
		ipa_curr->addr = NULL;

		free(ipa_curr);
		ipa_curr = NULL;
	}
}

/*
 * Free internal fields of the iface structure.
 */
void free_iface(struct iface *nif) {
	if (!nif) {
		return;
	}

	free(nif->hw_addr);
	nif->hw_addr = NULL;

	free(nif->name);
	nif->name = NULL;

	free(nif->mtu);
	nif->mtu = NULL;

	if (nif->ip_addrs) {
		free_ip_addr(nif->ip_addrs);
	}
	free(nif->ip_addrs);
	nif->ip_addrs = NULL;
}

/*
 * Free internal fields of the route structure.
 */
void free_route(struct route *rt) {
	if (!rt) {
		return;
	}

	free(rt->src);
	rt->src = NULL;

	free(rt->dst);
	rt->dst = NULL;

	free(rt->gw);
	rt->gw = NULL;

	free(rt->dev);
	rt->dev = NULL;
}

/*
 * Modify the global list of interfaces by inserting a new ip_addr structure
 * into the linked list of the specified interface.
 */
int insert_ip_addr(int idx, const char *addr, unsigned char family)
{
	struct ip_addr *ipa;
	struct ip_addr *tmp;

	/* Checking params */
	if (idx < 0) {
		fprintf(stderr, "Wrong index %d provided", idx);
		return EINVAL;
	}
	if (!addr) {
		fprintf(stderr, "IP address is empty");
		return EINVAL;
	}

	ipa = (struct ip_addr *) calloc(1, sizeof(struct ip_addr));
	if (!ipa) {
		return ENOMEM;
	}

	ipa->addr = strdup(addr);
	if (!ipa->addr) {
		return ENOMEM;
	}

	ipa->family = family;
	ipa->next = NULL;

	if (!iface_list[idx].ip_addrs) {
		iface_list[idx].ip_addrs = ipa;
		return 0;
	}

	tmp = iface_list[idx].ip_addrs;
	while (tmp->next) {
		tmp = tmp->next;
	}

	tmp->next = ipa;

	return 0;
}

/*
 * Modify the global list of interfaces by removing an existing ip_addr
 * structure from the linked list of the specified interface.
 */
int delete_ip_addr(int idx, const char *addr)
{
	struct ip_addr *tmp;
	struct ip_addr *prev;
	bool found = false;
	int ret = -1;

	/* Checking params */
	if (idx < 0) {
		fprintf(stderr, "Wrong index %d provided", idx);
		return EINVAL;
	}
	if (!addr) {
		fprintf(stderr, "IP address is empty");
		return EINVAL;
	}

	if (!iface_list[idx].ip_addrs) {
		return ENOENT;
	}

	tmp = iface_list[idx].ip_addrs;
	if (tmp->addr && !strcmp(tmp->addr, addr)) {
		iface_list[idx].ip_addrs = iface_list[idx].ip_addrs->next;
		goto free_node;
	}

	prev = tmp;
	while (tmp->next) {
		prev = tmp;
		tmp = tmp->next;

		if (tmp->addr && !strcmp(tmp->addr, addr)) {
			found = true;
			break;
		}
	}

	if (!found) {
		fprintf(stderr, "No IP address matching %s for interface %s",
			addr, iface_list[idx].name);
		return ENOENT;
	}

	prev->next = tmp->next;

free_node:
	free(tmp->addr);
	free(tmp);

	return 0;
}

/*
 * Print program usage.
 */
void print_usage(void)
{
	printf("\nUsage: %s [options]\n\n", PROGRAM_NAME);
	printf(" -d, --debug        Enable debug output\n");
	printf(" -h, --help         Display usage\n");
	printf(" -n, --netns-path   Network namespace path (required)\n");
	printf(" -p, --sandbox-id   Sandbox ID (required)\n");
	printf(" -r, --runtime-path Runtime path (required)\n");
	printf(" -v, --version      Show version\n");
	printf("\n");
}

/* 
 * Validate the parameters and fill the appropriate structure.
 *
 * sandbox-id: Sandbox identifier related to the network namespace this
 * watcher has to listen to.
 * runtime-path: Runtime path is needed to call into it whenever a
 * change in the network namespace is detected.
 * netns-path: Network namespace path that needs to be monitored.
 *
 * It returns 0 if the parameters are valid, with a watcher_params structure
 * properly filled. Otherwise, it returns -1 with a null watcher_params
 * structure.
 */
int parse_options(int argc, const char **argv, struct watcher_params *params)
{
	struct option prog_opts[] = {
		{"debug", no_argument, 0, 'd'},
		{"help", no_argument, 0, 'h'},
		{"netns-path", required_argument, 0, 'n'},
		{"sandbox-id", required_argument, 0, 'p'},
		{"runtime-path", required_argument, 0, 'r'},
		{"version", no_argument, 0, 'v'},
		{ 0, 0, 0, 0},
	};

	int c;
	while ((c = getopt_long(argc, (char **)argv, "dhn:p:r:v",
					prog_opts, NULL)) != -1) {
		switch (c) {
		case 'd':
			debug = true;
			break;
		case 'h':
			print_usage();
			exit(EXIT_SUCCESS);
		case 'n':
			params->netns_path = strdup(optarg);
			break;
		case 'p':
			params->sandbox_id = strdup(optarg);
			break;
		case 'r':
			params->runtime_path = strdup(optarg);
			break;
		case 'v':
			print_version();
			exit(EXIT_SUCCESS);
		default:
			print_usage();
			exit(EXIT_FAILURE);
		}
	}

	if (!params->netns_path) {
		fprintf(stderr, "Missing network namespace path\n");
		goto err;

	}

	if (!params->sandbox_id) {
		fprintf(stderr, "Missing sandbox ID\n");
		goto err;
	}

	if (!params->runtime_path) {
		fprintf(stderr, "Missing runtime path\n");
		goto err;
	}

	return 0;

err:
	print_usage();
	return EINVAL;
}

/* 
 * Enter the network namespace.
 *
 * Entering the provided network namespace since the point of this binary is
 * to monitor any network change happening inside a specific network namespace.
 */
int enter_netns(const char *netns_path)
{
	int fd;

	if (!netns_path) {
		fprintf(stderr, "Network namespace path is empty");
		return EINVAL;
	}

	fd = open(netns_path, O_RDONLY);
	if (fd == -1) {
		fprintf(stderr, "Failed opening network ns %s: %s\n",
				netns_path, strerror(errno));
		return -1;
	}

	if (setns(fd, 0) == -1) {
		fprintf(stderr, "Failed to join network ns %s: %s\n",
				netns_path, strerror(errno));
		return -1;
	}

	return 0;
}

/*
 * Open the netlink socket with appropriate parameters, and return the file
 * descriptor. It will listen for any message related to a network interface
 * change (add/delete/update), an IP address (IPv4) change (add/delete/update)
 * or a route (IPv4) change (add/delete/update).
 */
int open_netlink()
{
	int sock;
	struct sockaddr_nl sa;

	memset(&sa, 0, sizeof(sa));

	sa.nl_family = AF_NETLINK;
	sa.nl_pid = getpid();
	sa.nl_groups = RTMGRP_LINK |
		RTMGRP_IPV4_IFADDR |
		RTMGRP_IPV4_ROUTE;

	sock = socket(AF_NETLINK, SOCK_RAW, NETLINK_ROUTE);
	if (sock == -1) {
		fprintf(stderr, "Failed creating netlink socket: %s\n",
				strerror(errno));
		return -1;
	}

	if (bind(sock, (struct sockaddr *) &sa, sizeof(sa)) == -1) {
		fprintf(stderr, "Failed binding netlink socket: %s\n",
				strerror(errno));
		return -1;
	}

	return sock;
}

/*
 * Execute the runtime call from a child process.
 */
int fork_runtime_call(char *params[])
{
	int wstatus;
	int exit_code;
	int pid;

	if (!params) {
		fprintf(stderr, "Parameters are NULL\n");
		return EINVAL;
	}

	pid = fork();
	if (pid < 0) {
		fprintf(stderr, "Could not spawn the child: %s\n",
			strerror(errno));
		return -1;
	} else if (!pid) {
		execvp(params[0], params);

		/* This part of the code should not be reached */
		fprintf(stderr, "Failed execvp() runtime\n");
		return -1;
	}

	if (waitpid(pid, &wstatus, 0) == -1) {
		fprintf(stderr, "Failed waitpid(): %s\n", strerror(errno));
		return -1;
	}

	if (WIFEXITED(wstatus)) {
		exit_code = WEXITSTATUS(wstatus);
		printf("Runtime exit code %d\n", exit_code);
		if (!exit_code) {
			return -1;
		}
	}

	return 0;
}

/*
 * Convert a binary IP address.
 * 
 * It's the caller responsibility to free the memory.
 */
char* ip_addr_to_string(unsigned char family, void *attr)
{
	int buf_len;
	char *buf;

	if (family == AF_INET6) {
		buf_len = (INET6_ADDRSTRLEN + 1) * sizeof(char);
	} else {
		buf_len = (INET_ADDRSTRLEN + 1) * sizeof(char);
	}

	buf = (char *) calloc(1, buf_len);

	return (char *) inet_ntop(family, attr, buf, buf_len);
}

/*
 * Get network interface name from index.
 * 
 * It's the caller responsibility to free the memory.
 */
char* iface_idx_to_name(int idx)
{
	char *buf = (char *) calloc(1, IF_NAMESIZE);

	return if_indextoname((unsigned int)idx, buf);
}

/*
 * Convert an unsigned mac address into an hexadecimal string.
 * 
 * It's the caller responsibility to free the memory.
 */
char* hw_addr_to_string(unsigned char *c)
{
	char *buf;

	if (asprintf(&buf, "%02x:%02x:%02x:%02x:%02x:%02x",
				c[0], c[1], c[2], c[3], c[4], c[5]) == -1) {
		printf("Failed allocating memory: %s\n", strerror(errno));
		return NULL;
	}

	return buf;
}

/*
 * Get information from ifinfomsg.
 */
struct iface parse_ifinfomsg(const struct nlmsghdr *nh)
{
	struct ifinfomsg *ifi = (struct ifinfomsg *) NLMSG_DATA(nh);
	struct rtattr *attr;
	int ret;

	if (ifi->ifi_change & IFF_UP) {
		printf("IFI CHANGE IFF_UP\n");
	}
	if (ifi->ifi_change & IFF_RUNNING) {
		printf("IFI CHANGE IFF_RUNNING\n");
	}
	if (ifi->ifi_flags & IFF_UP) {
		printf("IFI FLAGS IFF_UP\n");
	}
	if (ifi->ifi_flags & IFF_RUNNING) {
		printf("IFI FLAGS IFF_RUNNING\n");
	}
	

	struct iface nif = {
		.idx      = ifi->ifi_index,
		.hw_addr  = NULL,
		.name     = NULL,
		.mtu      = NULL,
		.ip_addrs = NULL,
	};

	int len = nh->nlmsg_len - NLMSG_LENGTH (sizeof(*ifi));

	for (attr = IFLA_RTA (ifi); RTA_OK (attr, len);
			attr = RTA_NEXT (attr, len)) {
		switch(attr->rta_type){
		case IFLA_UNSPEC:
			break;
		case IFLA_ADDRESS:
			nif.hw_addr = hw_addr_to_string(
					(unsigned char *) RTA_DATA (attr));
			break;
		case IFLA_BROADCAST:
			break;
		case IFLA_IFNAME:
			nif.name = strdup((char *) RTA_DATA (attr));
			break;
		case IFLA_MTU:
			ret = asprintf(&(nif.mtu), "%d",
					*((unsigned int *) RTA_DATA (attr)));
			if (ret == -1) {
				fprintf(stderr,
					"Failed allocating memory: %s\n",
					strerror(errno));
			}
			break;
		case IFLA_LINK:
			break;
		case IFLA_QDISC:
			break;
		case IFLA_STATS:
			break;
		default:
			break;
		}
	}

	return nif;
}

/*
 * Get information from ifaddrmsg.
 */
struct iface parse_ifaddrmsg(const struct nlmsghdr *nh)
{
	struct ifaddrmsg *ifa = (struct ifaddrmsg *) NLMSG_DATA(nh);
	struct rtattr *attr;

	struct iface nif = {
		.idx      = ifa->ifa_index,
		.hw_addr  = NULL,
		.name     = NULL,
		.mtu      = NULL,
		.ip_addrs = NULL,
	};

	struct ip_addr *ipa = (struct ip_addr *)
		calloc(1, sizeof(struct ip_addr));

	ipa->family = ifa->ifa_family;
	ipa->next = NULL;

	int len = nh->nlmsg_len - NLMSG_LENGTH (sizeof(*ifa));

	for (attr = IFA_RTA (ifa); RTA_OK (attr, len);
			attr = RTA_NEXT (attr, len)) {
		switch(attr->rta_type){
		case IFA_UNSPEC:
			break;
		case IFA_ADDRESS:
			break;
		case IFA_LOCAL:
			ipa->addr = ip_addr_to_string(ifa->ifa_family,
					RTA_DATA (attr));
			break;
		case IFA_LABEL:
			nif.name = strdup((char *) RTA_DATA (attr));
			break;
		case IFA_BROADCAST:
			break;
		case IFA_ANYCAST:
			break;
		case IFA_CACHEINFO:
			break;
		default:
			break;
		}
	}

	nif.ip_addrs = ipa;

	return nif;
}

/*
 * Get information from rtmsg.
 */
struct route parse_rtmsg(const struct nlmsghdr *nh)
{
	struct rtmsg *rtm = (struct rtmsg *) NLMSG_DATA(nh);
	struct rtattr *attr;
	int ret;
	char *dst_ip = NULL;

	struct route rt = {
		.src = NULL,
		.dst = NULL,
		.gw  = NULL,
		.dev = NULL,
	};

	int len = RTM_PAYLOAD (nh);

	for (attr = RTM_RTA (rtm); RTA_OK (attr, len);
			attr = RTA_NEXT (attr, len)) {
		switch(attr->rta_type){
		case RTA_UNSPEC:
			break;
		case RTA_DST:
			dst_ip = ip_addr_to_string(rtm->rtm_family,
					RTA_DATA (attr));
			ret = asprintf(&(rt.dst), "%s/%d",
					dst_ip, rtm->rtm_dst_len);
			if (ret == -1) {
				fprintf(stderr,
					"Failed allocating memory: %s\n",
					strerror(errno));
			}
			free(dst_ip);
			break;
		case RTA_SRC:
			rt.src = ip_addr_to_string(rtm->rtm_family,
					RTA_DATA (attr));
			break;
		case RTA_IIF:
			break;
		case RTA_OIF:
			rt.dev = iface_idx_to_name(*((int *) RTA_DATA (attr)));
			break;
		case RTA_GATEWAY:
			rt.gw = ip_addr_to_string(rtm->rtm_family,
					RTA_DATA (attr));
			break;
		case RTA_PRIORITY:
			break;
		case RTA_PREFSRC:
			break;
		case RTA_METRICS:
			break;
		case RTA_MULTIPATH:
			break;
		case RTA_PROTOINFO:
			break;
		case RTA_FLOW:
			break;
		case RTA_CACHEINFO:
			break;
		default:
			break;
		}
	}

	return rt;
}

/*
 * Check if the interface parameters have changed compared to the global list
 * of interfaces.
 */
bool iface_changed(struct iface new, struct iface old)
{
	if (new.hw_addr != old.hw_addr ||
			new.name != old.name ||
			new.mtu != old.mtu) {
		return true;
	}

	return false;
}

/*
 * Add a new interface to the list of interfaces.
 */
void add_iface_to_list(struct iface nif)
{
	iface_list[nif.idx].idx = nif.idx;

	/* Name */
	if (nif.name) {
		iface_list[nif.idx].name = strdup(nif.name);
	}

	/* HW Address */
	if (nif.hw_addr) {
		iface_list[nif.idx].hw_addr = strdup(nif.hw_addr);
	}

	/* MTU */
	if (nif.mtu) {
		iface_list[nif.idx].mtu = strdup(nif.mtu);
	}
}

/*
 * Clean an existing interface from the list of interfaces.
 */
void delete_iface_from_list(unsigned int idx)
{
	free_iface(&(iface_list[idx]));

	iface_list[idx].idx = INIT_IF_IDX;
}

/*
 * Update list of interfaces by modifying only name/hw_addr/mtu.
 */
void update_iface_list(struct iface nif)
{
	/* Name */
	if (nif.name) {
		free(iface_list[nif.idx].name);
		iface_list[nif.idx].name = strdup(nif.name);
	}

	/* HW Address */
	if (nif.hw_addr) {
		free(iface_list[nif.idx].hw_addr);
		iface_list[nif.idx].hw_addr = strdup(nif.hw_addr);
	}

	/* MTU */
	if (nif.mtu) {
		free(iface_list[nif.idx].mtu);
		iface_list[nif.idx].mtu = strdup(nif.mtu);
	}
}

/*
 * Update interface subroutine that can be used both from add_interface()
 * and update_interface().
 */
int _update_interface(const struct nlmsghdr *nh, struct iface nif,
		const char *sandbox_id, const char* runtime_path) {
	int ret;

	/* 
	 * Call into the runtime with the proper command:
	 * # kata-runtime upd-net-if --name ...
	 */
	printf("# %s upd-net-if --name %s --hw-addr %s --mtu %s\n",
		runtime_path, iface_list[nif.idx].name,
		iface_list[nif.idx].hw_addr, iface_list[nif.idx].mtu);

	/* If the exec terminated properly, let's update iface_list */
	update_iface_list(nif);

	if (nh->nlmsg_type == RTM_NEWADDR) {
		ret = insert_ip_addr(nif.idx,
				     (const char*)nif.ip_addrs->addr,
				     nif.ip_addrs->family);
	} else if (nh->nlmsg_type == RTM_DELADDR) {
		ret = delete_ip_addr(nif.idx,
				     (const char*)nif.ip_addrs->addr);
	}

	free_iface(&nif);

	return ret;
}

/*
 * Add interface.
 */
int add_interface(const struct nlmsghdr *nh, const char *sandbox_id,
		const char* runtime_path)
{
	char *params[3];
	struct iface nif = parse_ifinfomsg(nh);

	if (iface_list[nif.idx].idx != INIT_IF_IDX) {
		if (!iface_changed(nif, iface_list[nif.idx])) {
			printf("Interface %s didn't change\n", nif.name);
			return 0;
		}

		return _update_interface(nh, nif, sandbox_id, runtime_path);
	}

	/* 
	 * Call into the runtime with the proper command:
	 * # kata-runtime add-net-if --name ...
	 */
	printf("# %s add-net-if --name %s --hw-addr %s --mtu %s\n",
			runtime_path, nif.name, nif.hw_addr, nif.mtu);

	params[0] = (char *) runtime_path;
	asprintf(&(params[1]), "add-net-if --name %s --hw-addr %s --mtu %s",
			nif.name, nif.hw_addr, nif.mtu);
	params[2] = NULL;

	//fork_runtime_call(params);

	/* If the exec terminated properly, let's add it to iface_list */
	add_iface_to_list(nif);

	free(params[1]);
	free_iface(&nif);

	return 0;
}

/*
 * Update interface.
 */
int update_interface(const struct nlmsghdr *nh, const char *sandbox_id,
		const char* runtime_path)
{
	struct iface nif = parse_ifaddrmsg(nh);

	return _update_interface(nh, nif, sandbox_id, runtime_path);
}

/*
 * Delete interface.
 */
int delete_interface(const struct nlmsghdr *nh, const char *sandbox_id,
		const char* runtime_path)
{
	struct iface nif = parse_ifinfomsg(nh);

	/* 
	 * Call into the runtime with the proper command:
	 * # kata-runtime del-net-if --name ...
	 */
	printf("# %s del-net-if --name %s\n", runtime_path, nif.name);

	/* If the exec terminated properly, let's delete it from iface_list */
	delete_iface_from_list(nif.idx);

	free_iface(&nif);

	return 0;
}

/*
 * Add route.
 */
int add_route(const struct nlmsghdr *nh, const char *sandbox_id,
		const char* runtime_path)
{
	struct route rt = parse_rtmsg(nh);

	/* 
	 * Call into the runtime with the proper command:
	 * # kata-runtime add-net-route --name ...
	 */
	printf("# %s add-net-route --src %s --dst %s --gw %s --dev %s\n",
			runtime_path, rt.src, rt.dst, rt.gw, rt.dev);

	free_route(&rt);

	return 0;
}

/*
 * Delete route.
 */
int delete_route(const struct nlmsghdr *nh, const char *sandbox_id,
		const char* runtime_path)
{
	struct route rt = parse_rtmsg(nh);

	/* 
	 * Call into the runtime with the proper command:
	 * # kata-runtime del-net-route --name ...
	 */
	printf("# %s del-net-route --src %s --dst %s --gw %s --dev %s\n",
			runtime_path, rt.src, rt.dst, rt.gw, rt.dev);

	free_route(&rt);

	return 0;
}

/*
 * Listen to the netlink socket and handle messages.
 */
int listen_netlink(int fd, const char *sandbox_id, const char* runtime_path)
{
	int ret;
	int len;
	char buf[8192];
	struct iovec iov = { buf, sizeof(buf) };
	struct sockaddr_nl sa;
	struct msghdr msg = { &sa, sizeof(sa), &iov, 1, NULL, 0, 0 };
	struct nlmsghdr *nh;
	bool if_list_changed;

	len = recvmsg(fd, &msg, 0);

	if (len < 0) {
		/* Non blocking socket, we should not error */
		if (errno == EWOULDBLOCK || errno == EAGAIN) {
			return 0;
		}

		fprintf(stderr, "Failed reading netlink socket: %s\n",
				strerror(errno));

		return len;
	}

	for (nh = (struct nlmsghdr *) buf; NLMSG_OK (nh, len);
			nh = NLMSG_NEXT (nh, len)) {

		if_list_changed = false;

		switch (nh->nlmsg_type) {
		case NLMSG_DONE:
			return 0;
		case NLMSG_ERROR:
			fprintf(stderr,
				"Error while listening on netlink socket\n");
			return -1;
		case RTM_NEWADDR:
			printf("netlink msg type: RTM_NEWADDR\n");
			ret = update_interface((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			if_list_changed = true;
			break;
		case RTM_DELADDR:
			printf("handle_netlink_message: RTM_DELADDR\n");
			ret = update_interface((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			if_list_changed = true;
			break;
		case RTM_NEWROUTE:
			printf("handle_netlink_message: RTM_NEWROUTE\n");
			ret = add_route((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			break;
		case RTM_DELROUTE:
			printf("handle_netlink_message: RTM_DELROUTE\n");
			ret = delete_route((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			break;
		case RTM_NEWLINK:
			printf("handle_netlink_message: RTM_NEWLINK\n");
			ret = add_interface((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			if_list_changed = true;
			break;
		case RTM_DELLINK:
			printf("handle_netlink_message: RTM_DELLINK\n");
			ret = delete_interface((const struct nlmsghdr *) nh,
					sandbox_id, runtime_path);
			if_list_changed = true;
			break;
		default:
			printf("handle_netlink_message: Unknown msg type %d\n",
					nh->nlmsg_type);
			break;
		}

		if (debug && if_list_changed) {
			print_iface_list();
		}

		if (ret) {
			fprintf(stderr, "Failed handling netlink message\n");
			return ret;
		}
	}

	return 0;
}

/* 
 * Monitor the network and call into the runtime to update the network of
 * the sandbox.
 *
 * The netlink socket is going to be listened to detect any change that could
 * happen to the network of the current network namespace.
 * As soon as a change gets detected, the runtime binary will be called with
 * the appropriate options to reflect the network change.
 */
int monitor_netns(const char *sandbox_id, const char* runtime_path)
{
	int ret;
	int fd;

	/* Open netlink socket */
	fd = open_netlink();
	if (fd <= 0) {
		return -1;
	}

	while (1) {
		/* Listen and handle netlink messages */
		ret = listen_netlink(fd, sandbox_id, runtime_path);
		if (ret) {
			return ret;
		}
	}

	return 0;
}

/*
 * Initialize global list of interfaces.
 */
void init_iface_list()
{
	int i;

	for (i = 0; i < MAX_IFACES; i++) {
		struct iface nif = {
			.idx      = INIT_IF_IDX,
			.hw_addr  = NULL,
			.name     = NULL,
			.mtu      = NULL,
			.ip_addrs = NULL,
		};

		iface_list[i] = nif;
	}
}

/*
 * Fill a new interface from the information given by ifaddrs.
 */
int create_iface_from_ifaddrs(struct ifaddrs *ifa, struct iface *nif)
{
	int sock;
	int ret = 0;
	struct ifreq ifr;

	if (!strncpy(ifr.ifr_name , nif->name, IFNAMSIZ-1)) {
		fprintf(stderr, "Failed to copy interface name %s", nif->name);
		return -1;
	}

	sock = socket(AF_INET, SOCK_DGRAM, 0);
	if (sock == -1) {
		fprintf(stderr, "Failed to open socket: %s", strerror(errno));
		return -1;
	}

	if ((ret = ioctl(sock, SIOCGIFHWADDR, &ifr)) == -1) {
		fprintf(stderr, "Failed to get HW ADDR: %s", strerror(errno));
		goto exit;
	}

	nif->hw_addr = hw_addr_to_string((unsigned char *)
			ifr.ifr_hwaddr.sa_data);

	if ((ret = ioctl(sock, SIOCGIFMTU, &ifr)) == -1) {
		fprintf(stderr, "Failed to get MTU: %s", strerror(errno));
		goto exit;
	}

	if (asprintf(&(nif->mtu), "%d", ifr.ifr_mtu) == -1) {
		fprintf(stderr, "Failed allocating memory: %s\n",
			strerror(errno));
		ret = ENOMEM;
		goto exit;
	}

exit:
	close(sock);
	return ret;
}

/*
 * Fill a new interface from the information given by ifaddrs.
 */
int add_ip_addr_to_iface_list(struct ifaddrs *ifa, unsigned int idx)
{
	unsigned char family;
	int s;
	char host[NI_MAXHOST];

	family = ifa->ifa_addr->sa_family;
	if (family != AF_INET && family != AF_INET6) {
		return 0;
	}

	s = getnameinfo(ifa->ifa_addr,
			(family == AF_INET) ? sizeof(struct sockaddr_in) :
			sizeof(struct sockaddr_in6),
			host, NI_MAXHOST, NULL, 0, NI_NUMERICHOST);
	if (s) {
		printf("Failed getnameinfo(): %s\n", gai_strerror(s));
		return -1;
	}

	return insert_ip_addr(idx, (const char*)host, family);
}

/* 
 * Scan the network and store it into a global list.
 *
 * A precise description of the network is needed to make sure events received
 * through the netlink socket will be properly interpreted.
 */
int scan_netns()
{
	struct ifaddrs *ifaddr, *ifa;
	int n;
	int ret = 0;

	if (getifaddrs(&ifaddr) == -1) {
		printf("Failed getifaddrs(): %s\n", strerror(errno));
		return -1;
	}

	for (ifa = ifaddr, n = 0; ifa != NULL; ifa = ifa->ifa_next, n++) {
		struct iface nif;
		unsigned int if_idx;

		if (!ifa->ifa_name) {
			continue;
		}

		if_idx = if_nametoindex(ifa->ifa_name);
		if (!if_idx) {
			printf("Failed if_nametoindex() for %s: %s\n",
					ifa->ifa_name, strerror(errno));
			goto exit;
		}

		if (iface_list[if_idx].idx == INIT_IF_IDX) {
			nif.name = strdup(ifa->ifa_name);

			ret = create_iface_from_ifaddrs(ifa, &nif);
			if (ret) {
				free_iface(&nif);
				goto exit;
			}

			nif.idx = if_idx;

			if (if_idx >= MAX_IFACES) {
				printf("Interface idx %d over the limit %d\n",
						if_idx, MAX_IFACES);
				free_iface(&nif);
				goto exit;
			}

			/* Update the global list of interfaces */
			iface_list[if_idx] = nif;
		}

		ret = add_ip_addr_to_iface_list(ifa, if_idx);
		if (ret) {
			goto exit;
		}
	}

exit:
	freeifaddrs(ifaddr);
	return ret;
}

int main(int argc, char **argv)
{
	int ret = 0;

	struct watcher_params params = {
		.netns_path   = NULL,
		.sandbox_id   = NULL,
		.runtime_path = NULL,
	};

	/* Initialize list of interfaces */
	init_iface_list();

	/* Validate parameters */
	ret = parse_options(argc, (const char**) argv, &params);
	if (ret) {
		goto exit;
	}

	/* Enter network namespace */
	ret = enter_netns((const char*) params.netns_path);
	if (ret) {
		goto exit;
	}

	/* Scan network namespace */
	ret = scan_netns();
	if (ret) {
		goto exit;
	}

	/* Print content of iface_list */
	if (debug) {
		print_iface_list();
	}

	/* Monitor the network */
	ret = monitor_netns((const char*) params.sandbox_id,
			(const char*) params.runtime_path);

exit:
	free_watcher_params(&params);
	return ret;
}
