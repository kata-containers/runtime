package rootless

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var (
	isRootless  bool
	hostUID     int
	rootlessDir string
)

// userMapping reads the map_uid file from proc and returns true if the
// root container ID is mapped to a non root user ID.
func userMapping() (bool, int, error) {
	file, err := os.Open("/proc/self/uid_map")
	if err != nil {
		return false, 0, err
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			if err == io.EOF {
				return false, 0, nil
			}
			return false, 0, err
		}
		if line == nil {
			return false, 0, nil
		}

		// if the container id (id[0]) is 0 (root inside the container)
		// has a mapping to the host id (id[1]) that is not root, then
		// it can be determined that the host user is running rootless
		ids := strings.Fields(string(line))
		if ids[0] == "0" && ids[1] != "0" {
			uid, _ := strconv.Atoi(ids[1])
			return true, uid, nil
		}
	}
}

func setupDir(path string) error {
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0700)
	}
	return err
}

// setRootlessDir prepares the directory for rootless storage
// use homedirPath as the default location for rootless execution, however
// if unable to access or create directory fallback to the userpath.
func setRootlessDir() error {
	// TODO homedirpath doesn't work.
	//homedirPath := "~/.local/kata-rootless"
	//if err := setupDir(homedirPath); err == nil {
	//	rootlessDir = homedirPath
	//	return nil
	//}

	userPath := fmt.Sprintf("/run/user/%d", hostUID)
	if err := setupDir(userPath); err != nil {
		return fmt.Errorf("unable to set rootless dir: %v", err)
	}

	rootlessDir = userPath
	return nil
}

// SetRootless sets the isRootless variable depending on uid_mappings
// This should only be called once at the beginning of kata call
func SetRootless() error {
	mappings, uid, err := userMapping()
	if err != nil {
		return err
	}

	// don't update isRootless or hostUID  until after mapping error handling
	isRootless = mappings
	hostUID = uid

	if !isRootless {
		return nil
	}

	return setRootlessDir()
}

// IsRootless states whether kata is being ran with root or not
func IsRootless() bool {
	return isRootless
}

// GetRootlessDir returns the path to the location for rootless
// container and sandbox storage
func GetRootlessDir() string {
	return rootlessDir
}

// GetRootlessUID returns the UID of the user in the parent userNS
func GetRootlessUID() int {
	return hostUID
}
