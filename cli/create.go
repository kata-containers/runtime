// Copyright (c) 2014,2015,2016 Docker, Inc.
// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kata-containers/runtime/pkg/katautils"
	vc "github.com/kata-containers/runtime/virtcontainers"
	"github.com/kata-containers/runtime/virtcontainers/pkg/oci"
	"github.com/urfave/cli"
)

var createCLICommand = cli.Command{
	Name:  "create",
	Usage: "Create a container",
	ArgsUsage: `<container-id>

   <container-id> is your name for the instance of the container that you
   are starting. The name you provide for the container instance must be unique
   on your host.`,
	Description: `The create command creates an instance of a container for a bundle. The
   bundle is a directory with a specification file named "` + specConfig + `" and a
   root filesystem.
   The specification file includes an args parameter. The args parameter is
   used to specify command(s) that get run when the container is started.
   To change the command(s) that get executed on start, edit the args
   parameter of the spec.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bundle, b",
			Value: "",
			Usage: `path to the root of the bundle directory, defaults to the current directory`,
		},
		cli.StringFlag{
			Name:  "console",
			Value: "",
			Usage: "path to a pseudo terminal",
		},
		cli.StringFlag{
			Name:  "console-socket",
			Value: "",
			Usage: "path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Value: "",
			Usage: "specify the file to write the process id to",
		},
		cli.BoolFlag{
			Name:  "no-pivot",
			Usage: "warning: this flag is meaningless to kata-runtime, just defined in order to be compatible with docker in ramdisk",
		},
	},
	Action: func(context *cli.Context) error {
		ctx, err := cliContextToContext(context)
		if err != nil {
			return err
		}

		runtimeConfig, ok := context.App.Metadata["runtimeConfig"].(oci.RuntimeConfig)
		if !ok {
			return errors.New("invalid runtime config")
		}

		console, err := setupConsole(context.String("console"), context.String("console-socket"))
		if err != nil {
			return err
		}

		return create(ctx, context.Args().First(),
			context.String("bundle"),
			console,
			context.String("pid-file"),
			true,
			context.Bool("systemd-cgroup"),
			runtimeConfig,
		)
	},
}

func create(ctx context.Context, containerID, bundlePath, console, pidFilePath string, detach, systemdCgroup bool,
	runtimeConfig oci.RuntimeConfig) error {
	var err error

	span, ctx := katautils.Trace(ctx, "create")
	defer span.Finish()

	kataLog = kataLog.WithField("container", containerID)
	setExternalLoggers(ctx, kataLog)
	span.SetTag("container", containerID)

	if bundlePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		kataLog.WithField("directory", cwd).Debug("Defaulting bundle path to current directory")

		bundlePath = cwd
	}

	// Checks the MUST and MUST NOT from OCI runtime specification
	if bundlePath, err = validCreateParams(ctx, containerID, bundlePath); err != nil {
		return err
	}

	ociSpec, err := oci.ParseConfigJSON(bundlePath)
	if err != nil {
		return err
	}

	containerType, err := ociSpec.ContainerType()
	if err != nil {
		return err
	}

	katautils.HandleFactory(ctx, vci, &runtimeConfig)

	disableOutput := noNeedForOutput(detach, ociSpec.Process.Terminal)

	var process vc.Process
	switch containerType {
	case vc.PodSandbox:
		process, err = createSandbox(ctx, ociSpec, runtimeConfig, containerID, bundlePath, console, disableOutput, systemdCgroup)
		if err != nil {
			return err
		}
	case vc.PodContainer:
		process, err = createContainer(ctx, ociSpec, containerID, bundlePath, console, disableOutput)
		if err != nil {
			return err
		}
	}

	// Creation of PID file has to be the last thing done in the create
	// because containerd considers the create complete after this file
	// is created.
	return createPIDFile(ctx, pidFilePath, process.Pid)
}

func createSandbox(ctx context.Context, ociSpec oci.CompatOCISpec, runtimeConfig oci.RuntimeConfig,
	containerID, bundlePath, console string, disableOutput, systemdCgroup bool) (vc.Process, error) {
	span, ctx := katautils.Trace(ctx, "createSandbox")
	defer span.Finish()

	err := katautils.SetKernelParams(containerID, &runtimeConfig)
	if err != nil {
		return vc.Process{}, err
	}

	sandboxConfig, err := oci.SandboxConfig(ociSpec, runtimeConfig, bundlePath, containerID, console, disableOutput, systemdCgroup)
	if err != nil {
		return vc.Process{}, err
	}

	// Important to create the network namespace before the sandbox is
	// created, because it is not responsible for the creation of the
	// netns if it does not exist.
	if err := setupNetworkNamespace(&sandboxConfig.NetworkConfig); err != nil {
		return vc.Process{}, err
	}

	// Run pre-start OCI hooks.
	err = katautils.EnterNetNS(sandboxConfig.NetworkConfig.NetNSPath, func() error {
		return katautils.PreStartHooks(ctx, ociSpec, containerID, bundlePath)
	})
	if err != nil {
		return vc.Process{}, err
	}

	sandbox, err := vci.CreateSandbox(ctx, sandboxConfig)
	if err != nil {
		return vc.Process{}, err
	}

	sid := sandbox.ID()
	kataLog = kataLog.WithField("sandbox", sid)
	setExternalLoggers(ctx, kataLog)
	span.SetTag("sandbox", sid)

	containers := sandbox.GetAllContainers()
	if len(containers) != 1 {
		return vc.Process{}, fmt.Errorf("BUG: Container list from sandbox is wrong, expecting only one container, found %d containers", len(containers))
	}

	if err := addContainerIDMapping(ctx, containerID, sandbox.ID()); err != nil {
		return vc.Process{}, err
	}

	return containers[0].Process(), nil
}

func createContainer(ctx context.Context, ociSpec oci.CompatOCISpec, containerID, bundlePath,
	console string, disableOutput bool) (vc.Process, error) {

	span, ctx := katautils.Trace(ctx, "createContainer")
	defer span.Finish()

	ociSpec = katautils.SetEphemeralStorageType(ociSpec)

	contConfig, err := oci.ContainerConfig(ociSpec, bundlePath, containerID, console, disableOutput)
	if err != nil {
		return vc.Process{}, err
	}

	sandboxID, err := ociSpec.SandboxID()
	if err != nil {
		return vc.Process{}, err
	}

	kataLog = kataLog.WithField("sandbox", sandboxID)
	setExternalLoggers(ctx, kataLog)
	span.SetTag("sandbox", sandboxID)

	s, c, err := vci.CreateContainer(ctx, sandboxID, contConfig)
	if err != nil {
		return vc.Process{}, err
	}

	// Run pre-start OCI hooks.
	err = katautils.EnterNetNS(s.GetNetNs(), func() error {
		return katautils.PreStartHooks(ctx, ociSpec, containerID, bundlePath)
	})
	if err != nil {
		return vc.Process{}, err
	}

	if err := addContainerIDMapping(ctx, containerID, sandboxID); err != nil {
		return vc.Process{}, err
	}

	return c.Process(), nil
}

func createPIDFile(ctx context.Context, pidFilePath string, pid int) error {
	span, _ := katautils.Trace(ctx, "createPIDFile")
	defer span.Finish()

	if pidFilePath == "" {
		// runtime should not fail since pid file is optional
		return nil
	}

	if err := os.RemoveAll(pidFilePath); err != nil {
		return err
	}

	f, err := os.Create(pidFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	pidStr := fmt.Sprintf("%d", pid)

	n, err := f.WriteString(pidStr)
	if err != nil {
		return err
	}

	if n < len(pidStr) {
		return fmt.Errorf("Could not write pid to '%s': only %d bytes written out of %d", pidFilePath, n, len(pidStr))
	}

	return nil
}
