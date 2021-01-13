// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"github.com/kata-containers/runtime/pkg/katautils/katatrace"
	"github.com/urfave/cli"
)

var versionCLICommand = cli.Command{
	Name:  "version",
	Usage: "display version details",
	Action: func(context *cli.Context) error {
		ctx, err := cliContextToContext(context)
		if err != nil {
			return err
		}

		span, _ := katatrace.Trace(ctx, kataLog, "version", cliTags...)
		defer span.Finish()

		cli.VersionPrinter(context)
		return nil
	},
}
