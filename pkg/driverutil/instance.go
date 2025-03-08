// SPDX-FileCopyrightText: Copyright The Lima Authors
// SPDX-License-Identifier: Apache-2.0

package driverutil

import (
	"github.com/lima-vm/lima/pkg/driver"
	"github.com/lima-vm/lima/pkg/rpc"
)

func CreateTargetDriverInstance(base *driver.BaseDriver) driver.Driver {
	return rpc.New(base)
	// limaDriver := base.Instance.Config.VMType
	// if *limaDriver == limayaml.VZ {
	// 	return vz.New(base)
	// }
	// if *limaDriver == limayaml.WSL2 {
	// 	return wsl2.New(base)
	// }
	// if *limaDriver == limayaml.RPC {
	// 	return rpc.New(base)
	// }
	// return qemu.New(base)
}
