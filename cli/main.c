// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

#define _GNU_SOURCE
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>
#include <errno.h>
#include <stdbool.h>
#include <dirent.h>
#include <syslog.h>
#include <stdarg.h>
#include <libgen.h>

#include <linux/limits.h>
#include <sys/mount.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/stat.h>

#ifdef DEBUG
#define errorf(fmt, ...)  syslog(LOG_ERR,   "cgo: %s:%d [%s]: "fmt, __FILE__, __LINE__, __func__, ##__VA_ARGS__)
#define debugf(fmt, ...)  syslog(LOG_DEBUG, "cgo: %s:%d [%s]: "fmt, __FILE__, __LINE__, __func__, ##__VA_ARGS__)
#else
// FIXME: improve logging system
// Issue: https://github.com/kata-containers/runtime/issues/859
#define errorf(fmt, ...)  syslog(LOG_ERR, "time=\"0000-00-00T00:00:00.000000000-00:00\" level=error msg=\""fmt"\" source=cgo", ##__VA_ARGS__)
#define debugf(fmt, ...)
#endif //DEBUG

#define CHILD_MAX 2

#define close_fd(fd) if(fd > 0){ close(fd); fd=-1; }

typedef char command;
typedef char response;
typedef int (*hookFunc)(const char*);

enum {
	cmd_new_ns = 1,
	cmd_remove_ns,
	cmd_join_ns,
	cmd_persistent_ns,
	cmd_get_fs_info,
	cmd_close_channels,
};

enum {
	res_success = 0,
	res_failure,
};

struct namespace {
	int         type;
	const char* name;
	hookFunc    hook;
};

struct child {
	// child process id.
	pid_t pid;

	// fd to get data from the child.
	int read_fd;

	// fd to send data to the child.
	int write_fd;
};

struct fs_info {
	char device[PATH_MAX];
	char mount_point[PATH_MAX];
	char type[NAME_MAX];
	char data[PATH_MAX];
};

static const char* procMountsPath = "/proc/mounts";

static struct child children[CHILD_MAX] = { 0 };

// number of children
static int children_number = 0;

// Variables used by the children
// child: fd to get data from the parent.
static int child_read_fd = -1;

// child: fd to send data to the parent.
static int child_write_fd = -1;

// child: child's namespaces path
static char child_ns_path[PATH_MAX] = { 0 };

/**
 * Set slave propagation
 *
 * \param namespaces_path path to persistent namespaces
 *
 * \return 0 on success, 1 on failure
 */
static int mnt_slave(const char* namespaces_path) {
	// mount root as slave to not propagate events
	if (mount("none", "/", NULL, MS_REC|MS_SLAVE, NULL) == -1) {
		errorf("Could not mount / as slave: %s", strerror(errno));
		return 1;
	}
	return 0;
}

static const struct namespace supported_namespaces[] = {
	/* { .type = CLONE_NEWUSER,   .name = "user",   .hook = NULL, }, */
	/* { .type = CLONE_NEWCGROUP, .name = "cgroup", .hook = NULL, }, */
	{ .type = CLONE_NEWIPC,    .name = "ipc",    .hook = NULL, },
	{ .type = CLONE_NEWUTS,    .name = "uts",    .hook = NULL, },
	/* { .type = CLONE_NEWNET,    .name = "net",    .hook = NULL, }, */
	/* { .type = CLONE_NEWPID,    .name = "pid",    .hook = NULL, }, */
	{ .type = CLONE_NEWNS,     .name = "mnt",    .hook = mnt_slave, },
	{ .name = NULL }
};

/**
 * Finish current process with its children.
 * \param fmt format of the message to log before exit.
 */
static void die(const char* fmt, ...) {
	va_list args;

	va_start(args, fmt);
	vsyslog(LOG_ERR, fmt, args);
	va_end(args);

	exit(EXIT_FAILURE);
}

/**
 * send SIGKILL signal to all children
 */
static void kill_children(void) {
	int i;

	debugf("Killing children");

	for (i=0; i<CHILD_MAX; i++) {
		if (children[i].pid > 0) {
			kill(children[i].pid, SIGKILL);
		}
	}
}

/**
 * Wait for all children
 *
 * \return 0 on success and if all children finished correctly, otherwise -1 or
 * the exit code of last child that failed.
 */
static int wait_children(void) {
	int i;
	int status;
	int exit_code = 0;

	debugf("Waiting children");

	for (i=0; i<CHILD_MAX; i++) {
		if (children[i].pid <= 0) {
			continue;
		}

		if (waitpid(children[i].pid, &status, 2) == -1) {
			errorf("Could not wait child %d", children[i].pid);
			exit_code = -1;
		}

		close_fd(children[i].read_fd);
		close_fd(children[i].write_fd);

		if (WEXITSTATUS(status) != 0) {
			exit_code = WEXITSTATUS(status);
		}
	}

	return exit_code;
}

/**
 * spawn and save a new child.
 *
 * \return the process id of the new child, -1 on error, 0 in the child.
 */
static int spawn_save_child(void) {
	pid_t pid = 0;
	int rwFds[2];
	int wrFds[2];
	int i;

	debugf("spawning and saving a new child");

	if (children_number >= CHILD_MAX) {
		errorf("BUG: max number of children reached: %d", children_number);
		return -1;
	}

	if (pipe(rwFds) == -1) {
		errorf("Could not create pipe: %s", strerror(errno));
		return -1;
	}

	if (pipe(wrFds) == -1) {
		errorf("Could not create pipe: %s", strerror(errno));
		close_fd(rwFds[0]);
		close_fd(rwFds[1]);
		return -1;
	}

	// create a new child to continue with golang execution
	// parent process listens to the child waiting for instructions.
	pid = fork();
	if (pid == -1) {
		errorf("Could not fork process to continue with golang: %s",
			   strerror(errno));
		close_fd(rwFds[0]);
		close_fd(rwFds[1]);
		close_fd(wrFds[0]);
		close_fd(wrFds[1]);
		return -1;
	}

	if (pid != 0) {
		// parent

		// close child fds
		close_fd(wrFds[0]);
		close_fd(rwFds[1]);

		// save child
		children[children_number].pid = pid;
		children[children_number].read_fd = rwFds[0];
		children[children_number].write_fd = wrFds[1];

		children_number++;

		return pid;
	}

	// child

	// close unneeded fds
	for (i=0; i<CHILD_MAX; i++) {
		close_fd(children[i].read_fd);
		close_fd(children[i].write_fd);
		children[i].pid = 0;
	}

	close_fd(child_read_fd);
	close_fd(child_write_fd);

	// close parent fds
	close_fd(rwFds[0]);
	close_fd(wrFds[1]);

	child_read_fd = wrFds[0];
	child_write_fd = rwFds[1];

	return 0;
}

/**
 * Return the type of namespace. see setns(2) for more information about
 * namespace types.
 *
 * \param namespace name
 *
 * \return namespace type on sucess, -1 if the namespace is not supported.
 */
int get_ns_type(const char* namespace) {
	const struct namespace* ns;
	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		if (strncmp(namespace, ns->name, NAME_MAX) == 0) {
			return ns->type;
		}
	}

	return -1;
}

/**
 * Send a command and the data path to the parent process
 *
 * \param cmd command to send.
 * \param data to send.
 * \param size of data.
 *
 * \return 0 on success. -1 on failure.
 */
static int child_send_cmd(command cmd, const void* data, int size) {
	response res;

	// send command
	debugf("Sending command to the parent: %d", cmd);
	if (write(child_write_fd, &cmd, sizeof(cmd)) <= 0) {
		errorf("Could not send the command %d: %s", cmd, strerror(errno));
		return -1;
	}

	if (data != NULL) {
		// send lenght data
		debugf("Sending data lenght to the parent: %d", size);
		if (write(child_write_fd, &size, sizeof(size)) <= 0) {
			errorf("Could not send data lenght: %s", strerror(errno));
			return -1;
		}

		// send data
		debugf("Sending data to the parent");
		if (write(child_write_fd, data, size) <= 0) {
			errorf("Could not send data: %s", strerror(errno));
			return -1;
		}
	}

	// wait response
	if (read(child_read_fd, &res, sizeof(res)) <= 0) {
		errorf("Could not get response from parent: %s", strerror(errno));
		return -1;
	}

	debugf("Got response from parent: %d", res);

	if (res == res_failure) {
		errorf("Failed to run command %d in parent", cmd);
		return -1;
	}

	return 0;
}

/**
 * This function is used by the children.
 * Join the supported namespaces in namespaces_path.
 *
 * \param namespaces_path namespaces path.
 *
 * \return a bit mask of the namespaces joined, -1 on failure.
 */
static int child_join_namespaces(const char* namespaces_path) {
	const struct namespace* ns;
	struct stat st;
	char ns_path[PATH_MAX] = { 0 };
	int fd = 0;
	int ns_joined = 0;

	debugf("Moving child %d to the namespaces in %s", getpid(), namespaces_path);

	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		snprintf(ns_path, sizeof(ns_path), "%s/%s", namespaces_path, ns->name);

		if (stat(ns_path, &st) == -1) {
			debugf("Namespace %s not found", ns_path);
			continue;
		}

		fd = open(ns_path, O_RDONLY);
		if (fd == -1) {
			errorf("Could not open namespace file %s: %s", ns_path, strerror(errno));
			return -1;
		}

		debugf("Moving child %d to the namespace %s", getpid(), ns_path);

		if (setns(fd, ns->type) == -1) {
			errorf("Could not join namespace %s: %s", ns_path, strerror(errno));
			close_fd(fd);
			return -1;
		}

		ns_joined |= ns->type;

		close_fd(fd);
	}

	// save joined namespaces path
	snprintf(child_ns_path, sizeof(child_ns_path), "%s", namespaces_path);

	return ns_joined;
}

/**
 * This function is used by the children.
 * Create new persistent namespaces in namespaces_path.
 *
 * \param namespaces_path namespaces path.
 * \param len lenght of namespaces_path
 *
 * \return 0 on success, -1 on failure.
 */
static int child_new_namespaces(const char* namespaces_path, int len) {
	const struct namespace* ns = NULL;
	int ns_joined = 0;
	int unshare_flags = 0;
	int ret;

	// Filesystem where persistent namespaces are created MUST BE slave or private,
	// next tricks are to achieve that.
	if (mount(namespaces_path, namespaces_path, NULL, MS_BIND, NULL) != 0) {
		errorf("Could not bind mount namespaces directory: %s", strerror(errno));
		return -1;
	}

	// must be slave to see all new point points in old namespaces.
	if (mount("none", namespaces_path, NULL, MS_REC|MS_SLAVE, NULL) != 0) {
		errorf("Could not make namespaces directory slave: %s", strerror(errno));
		return -1;
	}

	// join existing namespaces
	ns_joined = child_join_namespaces(namespaces_path);
	if (ns_joined == -1) {
		errorf("Could not join namespaces in %s", namespaces_path);
		return -1;
	}

	// check supported namespaces and omit already joined namespaces
	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		if (ns_joined & ns->type) {
			continue;
		}
		debugf("Add namespace %s to unshare flags", ns->name);
		unshare_flags |= ns->type;
	}

	if (unshare_flags == 0) {
		debugf("No unshare flags");
		return 0;
	}

	debugf("Unsharing namespaces");
	if (unshare(unshare_flags) == -1) {
		errorf("Could not unshare namespaces %d: %s",
			   unshare_flags, strerror(errno));
		return -1;
	}

	// parent is the only one able to make persistent namespace
	ret = child_send_cmd(cmd_persistent_ns, namespaces_path, len);
	if (ret == -1) {
		return -1;
	}

	// run namespace hooks
	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		if (ns->hook != NULL) {
			debugf("Running %s hook", ns->name);
			ns->hook(namespaces_path);
		}
	}

	return 0;
}

/**
 * Get data from read_fd
 *
 * \param read_fd file descriptor.
 * \param[out] data where data read is copied.
 * \param data_size size of data
 *
 * \return size of data read, -1 on failure.
 */
static int parent_get_data(int read_fd, void* data, int data_size) {
	int size;
	int ret;

	// Read data lenght
	if (read(read_fd, &size, sizeof(size)) == -1) {
		errorf("Could not get data size from child: %s", strerror(errno));
		return -1;
	}

	if (size > data_size) {
		errorf("There is no enough space to save child's data: %d > %d", size, data_size);
		return -1;
	}

	data_size = size;

	while(data_size > 0) {
		ret = read(read_fd, data, size);
		if (ret == -1) {
			errorf("Could not get data from child: %s", strerror(errno));
			return -1;
		}
		data_size -= ret;
	}

	return size;
}

/**
 * Read from child the information needed to remove the persistent namespaces
 *
 * \param child process.
 *
 * \return 0 on success, otherwise -1.
 */
static int parent_remove_namespaces(const struct child* child) {
	char namespaces_path[PATH_MAX] = { 0 };
	char persistent_ns_path[PATH_MAX] = { 0 };
	struct stat st;
	const struct namespace* ns;

	debugf("Removing persistent namespaces");

	if (parent_get_data(child->read_fd, namespaces_path, sizeof(namespaces_path)) == -1) {
		return -1;
	}

	debugf("Got namespace path from child: %s", namespaces_path);

	if (stat(namespaces_path, &st) == -1) {
		errorf("Could not stat persistent namespaces path %s: %s",
			   persistent_ns_path, strerror(errno));
		return -1;
	}

	if ((st.st_mode & S_IFMT) != S_IFDIR) {
		errorf("Namespaces path %s is not a directory",
			   persistent_ns_path);
		return -1;
	}

	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		snprintf(persistent_ns_path, sizeof(persistent_ns_path), "%s/%s",
				 namespaces_path, ns->name);

		if (stat(persistent_ns_path, &st) == -1) {
			debugf("Persistent namespace not found %s: %s",
				   persistent_ns_path, strerror(errno));
			continue;
		}

		// not a regular file, not a persistent namespace
		if ((st.st_mode & S_IFMT) != S_IFREG) {
			debugf("File %s is not a regular file", persistent_ns_path);
			continue;
		}

		debugf("Unmounting persistent namespace %s", persistent_ns_path);
		if (umount(persistent_ns_path) == -1) {
			errorf("Could not unmount persistent namespace %s: %s",
				   persistent_ns_path, strerror(errno));
		}

		debugf("Removing persistent namespace %s", persistent_ns_path);
		if (remove(persistent_ns_path) == -1) {
			errorf("Could not remove persistent namespace %s: %s",
				   persistent_ns_path, strerror(errno));
		}
	}

	debugf("Unmounting namespaces directory: %s", namespaces_path);
	if (umount(namespaces_path) == -1) {
		errorf("Could not unmount persistent namespace %s: %s",
			   namespaces_path, strerror(errno));
		return -1;
	}

	return 0;
}

/**
 * Read from child the information needed to make child's namespaces persistent.
 * If namespace already exist in namespaces path, then it is ommited.
 *
 * \param child process.
 *
 * \return 0 on success, -1 on failure.
 */
static int parent_persistent_namespaces(const struct child* child) {
	const struct namespace* ns = NULL;
	int fd = 0;
	int ret = 0;
	char persistent_ns_path[PATH_MAX] = { 0 };
	char pid_ns_path[PATH_MAX] = { 0 };
	char namespaces_path[PATH_MAX] = { 0 };
	struct stat st;

	debugf("Making persistent namespaces");

	if (parent_get_data(child->read_fd, namespaces_path, sizeof(namespaces_path)) == -1) {
		return -1;
	}

	debugf("Got namespace path from child: %s", namespaces_path);

	for (ns = supported_namespaces; ns->name != NULL; ns++) {
		// source
		snprintf(pid_ns_path, sizeof(pid_ns_path), "/proc/%d/ns/%s",
				 child->pid, ns->name);

		// target
		snprintf(persistent_ns_path, sizeof(persistent_ns_path), "%s/%s",
				 namespaces_path, ns->name);

		if (stat(persistent_ns_path, &st) == 0) {
			debugf("Namespace already exist: %s", persistent_ns_path);
			continue;
		}

		// create namespace target
		fd = open(persistent_ns_path, O_WRONLY|O_CREAT|O_TRUNC, S_IRWXU);
		if (fd == -1) {
			errorf("Could not create persistent namespace %s: %s",
				   pid_ns_path, strerror(errno));
			return -1;
		}
		close_fd(fd);

		// bind mount namespace to create a persistent namespace
		ret = mount(pid_ns_path, persistent_ns_path, NULL, MS_BIND, NULL);
		if (ret == -1) {
			errorf("Could not bind mount %s in %s: %s",
				   pid_ns_path, persistent_ns_path, strerror(errno));
			return -1;
		}
		debugf("Created persistent namespace %s", persistent_ns_path);
	}

	return 0;
}

/**
 * Read from child the information needed to Spawn a new child in new namespaces.
 *
 * \param child process.
 *
 * \return 0 on success, 1 in the child process, otherwise -1.
 */
static int parent_new_namespaces(const struct child* child) {
	char namespaces_path[PATH_MAX] = { 0 };
	pid_t pid = 0;
	int len = 0;

	debugf("New persistent namespaces");

	len = parent_get_data(child->read_fd, namespaces_path, sizeof(namespaces_path));
	if (len == -1) {
		return -1;
	}

	debugf("Got namespace path from child: %s", namespaces_path);

	pid = spawn_save_child();
	if (pid == -1) {
		errorf("Could not fork and save a new child");
		return -1;
	}

	if (pid != 0) {
		// parent
		return 0;
	}

	// child
	if (child_new_namespaces(namespaces_path, len) == -1) {
		die("Could not create persistent namespaces");
	}

	return 1;
}

/**
 * Read from child the information needed to fork a new child and move it
 * to a new namespace.
 *
 * \param child process.
 *
 * \return 0 on success, 1 in the child process, -1 on failure.
 */
static int parent_join_namespaces(const struct child* child) {
	char namespaces_path[PATH_MAX] = { 0 };
	pid_t pid;

	debugf("Joining namespaces");

	if (parent_get_data(child->read_fd, namespaces_path, sizeof(namespaces_path)) == -1) {
		return -1;
	}

	pid = spawn_save_child();
	if (pid == -1) {
		errorf("Could not fork and save a new child");
		return -1;
	}

	if (pid != 0) {
		// parent
		return 0;
	}

	// child
	if (child_join_namespaces(namespaces_path) == -1) {
		die("Could not join namespaces in %s", namespaces_path);
	}

	return 1;
}

/**
 * Parse and set filesystem information
 *
 * \param[out] info contains filesystem information
 * \param data is a line read from /proc/mounts
 *
 * \return 0 on success, -1 on failure.
 */
static int parent_set_fs_info(struct fs_info* info, char* data) {
	char *tok = NULL;
	const char* del = " \t\n";

	// device
	tok = strtok(data, del);
	if (tok == NULL) {
		return -1;
	}
	snprintf(info->device, PATH_MAX, "%s", tok);

	// mount point
	tok = strtok(NULL, del);
	if (tok == NULL) {
		return -1;
	}
	snprintf(info->mount_point, PATH_MAX, "%s", tok);

	// type
	tok = strtok(NULL, del);
	if (tok == NULL) {
		return -1;
	}
	snprintf(info->type, NAME_MAX, "%s", tok);

	//data
	tok = strtok(NULL, del);
	if (tok == NULL) {
		return -1;
	}
	snprintf(info->data, PATH_MAX, "%s", tok);

	return 0;
}

/**
 * Read filesystem information of given path
 *
 * \param path filesystem path
 * \param[out] info contains filesystem information
 *
 * \return 0 on success, -1 on failure.
 */
static int parent_read_fs_info(const char* path, struct fs_info* info) {
	FILE *mountsFile = NULL;
	char* line = NULL;
	size_t len = 0;
	ssize_t read;
	int ret = -1;

	mountsFile = fopen(procMountsPath, "r");
	if (mountsFile == NULL) {
		errorf("Could not read file %s: %s", procMountsPath, strerror(errno));
		return -1;
	}

	while ((read = getline(&line, &len, mountsFile)) != -1) {
		debugf("Checking mount point: %s", line);
		if (parent_set_fs_info(info, line) == -1) {
			continue;
		}

		if (strncmp(path, info->mount_point, PATH_MAX) == 0) {
			ret = 0;
			break;
		}
	}

	fclose(mountsFile);
	if (line) {
		free(line);
	}

	return ret;
}

/**
 * Get the mount point of path
 *
 * \param path filesystem path
 * \param[out] mountPoint where path is mounted
 *
 * \return 0 on success, -1 on failure
 */
static int parent_get_mount_point(const char* path, char* mountPoint) {
	char fspath[PATH_MAX];
	struct stat st;
	dev_t dev_id_path = 0;

	debugf("Getting mount point of path: %s", path);

	strncpy(fspath, path, PATH_MAX);

	if (stat(fspath, &st) != 0) {
		errorf("Could not stat file %s: %s", fspath, strerror(errno));
		return -1;
	}

	dev_id_path = st.st_dev;
	strncpy(mountPoint, fspath, PATH_MAX);

	while (strncmp(dirname(fspath), "/", sizeof(fspath)) != 0) {
		debugf("Check %s device id", fspath);

		if (stat(fspath, &st) != 0) {
			errorf("Could not stat file %s: %s", fspath, strerror(errno));
			return -1;
		}

		if (dev_id_path != st.st_dev) {
			break;
		}

		snprintf(mountPoint, PATH_MAX, "%s", fspath);
	}

	if (strncmp(fspath, "/", sizeof(fspath)) == 0) {
		snprintf(mountPoint, PATH_MAX, "/");
	}

	return 0;
}

/**
 * Read from child the information needed to get a filesystem information
 *
 * \param child process.
 */
static void parent_get_fs_info(const struct child* child) {
	char fs_path[PATH_MAX] = { 0 };
	char mount_point[PATH_MAX] = { 0 };
	int ret;
	response res = res_failure;
	struct fs_info fsinfo;

	debugf("Get filesystem information");

	if (parent_get_data(child->read_fd, fs_path, sizeof(fs_path)) == -1) {
		goto end;
	}

	if (fs_path[0] != '/') {
		errorf("Filesystem path must be absolute: %s", fs_path);
		goto end;
	}

	if (parent_get_mount_point(fs_path, mount_point) != 0) {
		errorf("Could not get %s mount point", fs_path);
		goto end;
	}

	debugf("Got mount point: %s", mount_point);

	if (parent_read_fs_info(mount_point, &fsinfo) != 0) {
		errorf("Could not read %s filesystem information", mount_point);
		goto end;
	}

	res = res_success;

end:
	// send
	ret = write(child->write_fd, &res, sizeof(res));
	if (ret == -1) {
		errorf("Could not send response to the child: %s", strerror(errno));
	}

	if (res == res_success) {
		debugf("Sending filesystem to child: %s on %s", fsinfo.device, fsinfo.mount_point);

		// send fsinfo
		ret = write(child->write_fd, &fsinfo, sizeof(fsinfo));
		if (ret == -1) {
			errorf("Could not send filesystem information to the child: %s", strerror(errno));
		}
	}
}

/**
 * Listen a specific child.
 *
 * \param child to listen.
 *
 * \return 0 on success; -1 on failure.
 * If a new process is spawned 0 is returned in the parent and 1 in the child,
 * allowing child process continue with its execution.
 */
static int parent_listen_child(const struct child* child) {
	command cmd;
	response res;
	int ret;

	while ((ret = read(child->read_fd, &cmd, sizeof(cmd))) > 0) {
		debugf("Got command from child: %d", cmd);

		switch(cmd) {
		case cmd_new_ns:
			ret = parent_new_namespaces(child);
			break;

		case cmd_join_ns:
			ret = parent_join_namespaces(child);
			break;

		case cmd_remove_ns:
			ret = parent_remove_namespaces(child);
			break;

		case cmd_persistent_ns:
			ret = parent_persistent_namespaces(child);
			break;

		case cmd_get_fs_info:
			parent_get_fs_info(child);
			goto listen_child;

		case cmd_close_channels:
			close(child->read_fd);
			close((child->write_fd));
			return 0;

		default:
			errorf("Unsupported command: %d", cmd);
			ret = -1;
		}

		if (ret == 1) {
			// This is a new child
			return 1;
		}

		res = res_failure;
		if (ret == 0) {
			res = res_success;
		}

		// send response to the child
		ret = write(child->write_fd, &res, sizeof(res));
		if (ret == -1) {
			errorf("Could not send response to the child: %s", strerror(errno));
			return -1;
		}

listen_child:
		debugf("Listening child %d", child->pid);
	}

	if (ret == -1) {
		errorf("Could not get command from child: %s", strerror(errno));
		return -1;
	}

	debugf("Writing end of child %d was closed", child->pid);
	return 0;
}

/**
 * This function is called from Golang.
 * remove the persistent namespaces in namespaces_path.
 *
 * \param namespaces_path directory where persistent namespaces are.
 * This directory MUST exist.
 * \param len lenght of namespaces_path
 *
 * \return 0 on success, -1 on failure.
 */
int remove_namespaces(const char* namespaces_path, unsigned int len) {
	debugf("Removing persistent namespaces in %s", namespaces_path);

	return child_send_cmd(cmd_remove_ns, namespaces_path, len);
}

/**
 * This function is called from Golang.
 * Spawn a new child in new namespaces. If the namespace already exist in
 * namespaces_path, a new namespaces is not created, the child is just moved and
 * new namespaces are made persistent in namespaces_path.
 * On success, calling process should exit as soon as possible
 * to allow new child continue with its execution.
 *
 * \param namespaces_path directory where persistent namespaces are.
 * This directory MUST exist.
 *
 * \param len lenght of namespaces_path.
 *
 * \return 0 on success, 1 if persistent namespaces already exist, -1 on failure.
 */
int new_namespaces(const char* namespaces_path, unsigned int len) {
	if (strncmp(namespaces_path, child_ns_path, len) == 0) {
		return 1;
	}

	debugf("New persistent namespaces in %s", namespaces_path);

	return child_send_cmd(cmd_new_ns, namespaces_path, len);
}

/**
 * This function is called from Golang.
 * Spawn a new process and move it to the namespaces in namespaces_path
 * directory. On success, calling process should exit as soon as possible
 * to allow new child continue with its execution.
 *
 * \param namespaces_path persistent namespaces directory.
 * This directory MUST exist.
 * \param len lenght of namespaces_path
 *
 * \return 0 on success, 1 if the process already joined the namespaces,
 * -1 on failure.
 */
int join_namespaces(const char* namespaces_path, unsigned int len) {
	if (strncmp(namespaces_path, child_ns_path, len) == 0) {
		return 1;
	}

	debugf("Joining persistent namespaces in %s", namespaces_path);

	return child_send_cmd(cmd_join_ns, namespaces_path, len);
}


/**
 * This function is called from Golang.
 * Close the communication channel with the parent.
 *
 * \return 0 on success, -1 on failure.
 */
int close_channels(void) {
	int ret;
	command cmd;

	debugf("Closing communication channels");

	cmd = cmd_close_channels;
	ret = write(child_write_fd, &cmd, sizeof(cmd));

	close_fd(child_write_fd);
	close_fd(child_read_fd);

	return ret;
}

/**
 * This function is called from Golang.
 * Read directory's filesystem information
 *
 * \param fs path to get filesystem information
 * \param[out] device mounted, buffer capacity MUST BE >= PATH_MAX
 * \param[out] mount_point where device is mounted, buffer capacity MUST BE >= PATH_MAX
 * \param[out] type of the filesystem, buffer capacity MUST BE >= NAME_MAX
 * \param[out] data contains mount options associated with the filesystem, buffer capacity MUST BE >= PATH_MAX
 *
 * \return 0 on success, -1 on failure.
 */
int get_fs_info(const char* fs, char* device, char* mount_point, char* type, char* data) {
	struct fs_info info;

	debugf("Getting filesystem information %s", fs);

	if (child_send_cmd(cmd_get_fs_info, fs, strlen(fs)) != 0) {
		return -1;
	}

	// read filesystem information
	if (read(child_read_fd, &info, sizeof(info)) == -1) {
		errorf("Could not get filesystem information from parent: %s", strerror(errno));
		return -1;
	}

	strncpy(device, info.device, sizeof(info.device));
	strncpy(mount_point, info.mount_point, sizeof(info.mount_point));
	strncpy(type, info.type, sizeof(info.type));
	strncpy(data, info.data, sizeof(info.data));

	return 0;
}

/**
 * This function is called from Golang.
 * cgo entry point.
 * MUST BE called in the C constructor.
 */
void init(void) {
	pid_t pid = 0;
	int ret = 0;
	int i;

	pid = spawn_save_child();
	if (pid == -1) {
		die("Could not spawn and save a new child");
	}

	if (pid == 0) {
		//child
		// continue with golang code
		return;
	}

	// listen to the children
	for (i=0; i<CHILD_MAX; i++) {
		if (children[i].pid <= 0) {
			continue;
		}

		ret = parent_listen_child(&children[i]);
		close_fd(children[i].read_fd);
		close_fd(children[i].write_fd);

		if (ret == 1) {
			// new child
			// continue with golang code
			return;
		}

		if (ret == -1) {
			close_fd(child_read_fd);
			close_fd(child_write_fd);
			kill_children();
			wait_children();
			exit(EXIT_FAILURE);
		}
	}

	// wait for all children
	exit(wait_children());
}
