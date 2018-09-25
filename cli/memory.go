package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/docker/go-units"
	vc "github.com/kata-containers/runtime/virtcontainers"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var kataMemoryCLICommand = cli.Command{
	Name:  "kata-memory",
	Usage: "hotplug memory for container",
	Subcommands: []cli.Command{
		addMemoryCommand,
		delMemoryCommand,
		listMemoryCommand,
	},
	Action: func(context *cli.Context) error {
		return cli.ShowSubcommandHelp(context)
	},
}

var addMemoryCommand = cli.Command{
	Name:      "add",
	Usage:     "add memory to a container",
	ArgsUsage: `add <container-id> <memory size(MiB)>`,
	Flags:     []cli.Flag{},
	Action: func(context *cli.Context) error {
		ctx, err := cliContextToContext(context)
		if err != nil {
			return err
		}

		return memoryHotplugCommand(ctx, context.Args().First(), context.Args().Get(1))
	},
}

var delMemoryCommand = cli.Command{
	Name:      "del",
	Usage:     "delete memory from a container",
	ArgsUsage: `del <container-id> <memory size(MiB)>`,
	Flags:     []cli.Flag{},
	Action: func(context *cli.Context) error {
		ctx, err := cliContextToContext(context)
		if err != nil {
			return err
		}

		return memoryHotUnplugCommand(ctx, context.Args().First(), context.Args().Get(1))
	},
}

var listMemoryCommand = cli.Command{
	Name:      "list",
	Usage:     "list memory in a container",
	ArgsUsage: `list <container-id>`,
	Flags:     []cli.Flag{},
	Action: func(context *cli.Context) error {
		ctx, err := cliContextToContext(context)
		if err != nil {
			return err
		}

		return memoryListCommand(ctx, context.Args().First())
	},
}

type formatMemoryInfo interface {
	Write(memoryList vc.VmMemoryInfo, file *os.File) error
}

type formatMemoryTabular struct{}
type formatMemoryJSON struct{}

func (f formatMemoryTabular) Write(info vc.VmMemoryInfo, file *os.File) error {
	// values used by runc
	flags := uint(0)
	minWidth := 12
	tabWidth := 1
	padding := 3

	w := tabwriter.NewWriter(file, minWidth, tabWidth, padding, ' ', flags)

	fmt.Fprintf(w, "VM NAME: %s\n", info.Name)
	fmt.Fprintf(w, "VM UUID: %s\n", info.UUID)

	fmt.Fprint(w, "MEMORY:\n")
	fmt.Fprint(w, "ID\tTYPE\tSLOT\tSIZE(MiB)\tHOTPLUGGABLE\tHOTPLUGGED\n")
	for _, memory := range info.Mem {
		if memory.ID == "nv0" {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%t\t%t\n",
			memory.ID,
			memory.Type,
			memory.Slot,
			memory.Size/1024/1024,
			memory.Hotpluggable,
			memory.Hotplugged)
	}

	return w.Flush()
}

func (f formatMemoryJSON) Write(info vc.VmMemoryInfo, file *os.File) error {
	return json.NewEncoder(file).Encode(info)
}

func memoryListCommand(ctx context.Context, containerID string) (err error) {
	status, sandboxID, err := getExistingContainerInfo(ctx, containerID)
	if err != nil {
		return err
	}

	containerID = status.ID

	kataLog = kataLog.WithFields(logrus.Fields{
		"container": containerID,
		"sandbox":   sandboxID,
	})

	setExternalLoggers(ctx, kataLog)

	// container MUST be running
	if status.State.State != vc.StateRunning {
		return fmt.Errorf("container %s is not running", containerID)
	}

	mems, err := vci.ListMemory(ctx, sandboxID)
	if err != nil {
		return err
	}

	var fs formatMemoryInfo = formatMemoryTabular{}
	file := defaultOutputFile

	return fs.Write(*mems, file)
}

func memoryHotplugCommand(ctx context.Context, containerID, input string) (err error) {
	status, sandboxID, err := getExistingContainerInfo(ctx, containerID)
	if err != nil {
		return err
	}

	containerID = status.ID

	kataLog = kataLog.WithFields(logrus.Fields{
		"container": containerID,
		"sandbox":   sandboxID,
	})

	setExternalLoggers(ctx, kataLog)

	// container MUST be running
	if status.State.State != vc.StateRunning {
		return fmt.Errorf("container %s is not running", containerID)
	}

	sizeB, err := units.RAMInBytes(input)
	if err != nil {
		return fmt.Errorf("invalid value %s: %s", input, err)
	}
	sizeMiB := int(sizeB >> 20)

	return vci.HotplugMemory(ctx, sandboxID, sizeMiB)
}

func memoryHotUnplugCommand(ctx context.Context, containerID, input string) (err error) {
	status, sandboxID, err := getExistingContainerInfo(ctx, containerID)
	if err != nil {
		return err
	}

	containerID = status.ID

	kataLog = kataLog.WithFields(logrus.Fields{
		"container": containerID,
		"sandbox":   sandboxID,
	})

	setExternalLoggers(ctx, kataLog)

	// container MUST be running
	if status.State.State != vc.StateRunning {
		return fmt.Errorf("container %s is not running", containerID)
	}

	return vci.HotUnplugMemory(ctx, sandboxID, input)
}
