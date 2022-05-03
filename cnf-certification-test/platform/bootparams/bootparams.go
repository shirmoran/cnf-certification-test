// Copyright (C) 2020-2022 Red Hat, Inc.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

package bootparams

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/test-network-function/cnf-certification-test/internal/clientsholder"
	"github.com/test-network-function/cnf-certification-test/pkg/arrayhelper"
	"github.com/test-network-function/cnf-certification-test/pkg/loghelper"
	"github.com/test-network-function/cnf-certification-test/pkg/provider"
)

const (
	grubKernelArgsCommand = "cat /host/boot/loader/entries/$(ls /host/boot/loader/entries/ | sort | tail -n 1)"
	kernelArgscommand     = "cat /proc/cmdline"
)

func TestBootParamsHelper(env *provider.TestEnvironment, cut *provider.Container) (claimsLog loghelper.CuratedLogLines, err error) {
	debugPod := env.DebugPods[cut.NodeName]
	if debugPod == nil {
		err = fmt.Errorf("debug pod for container %s not found on node %s", cut, cut.NodeName)
		return claimsLog, err
	}
	mcKernelArgumentsMap, err := GetMcKernelArguments(env, cut.NodeName)
	if err != nil {
		return claimsLog, fmt.Errorf("error getting kernel arguments in node %s, err=%s", cut.NodeName, err)
	}
	currentKernelArgsMap, err := getCurrentKernelCmdlineArgs(cut)
	if err != nil {
		return claimsLog, fmt.Errorf("error getting kernel cli arguments from container: %s, err=%s", cut, err)
	}
	grubKernelConfigMap, err := getGrubKernelArgs(env, cut.NodeName)
	if err != nil {
		return claimsLog, fmt.Errorf("error getting grub  kernel arguments for node: %s, err=%s", cut.NodeName, err)
	}
	for key, mcVal := range mcKernelArgumentsMap {
		if currentVal, ok := currentKernelArgsMap[key]; ok {
			if currentVal != mcVal {
				claimsLog = claimsLog.AddLogLine("%s ContainerKernelArgs!=mcVal %s!=%s", cut, currentVal, mcVal)
			} else {
				logrus.Tracef("%s ContainerKernelArgs==mcVal %s==%s", cut, currentVal, mcVal)
			}
		}
		if grubVal, ok := grubKernelConfigMap[key]; ok {
			if grubVal != mcVal {
				claimsLog = claimsLog.AddLogLine("%s NodeGrubKernelArgs!=mcVal %s!=%s", cut, grubVal, mcVal)
			} else {
				logrus.Tracef("%s NodeGrubKernelArgs==mcVal %s==%s", cut, grubVal, mcVal)
			}
		}
	}
	return claimsLog, nil
}

func GetMcKernelArguments(env *provider.TestEnvironment, nodeName string) (aMap map[string]string, err error) {
	mcKernelArgumentsMap := arrayhelper.ArgListToMap(env.Nodes[nodeName].Mc.Spec.KernelArguments)
	return mcKernelArgumentsMap, nil
}

func getGrubKernelArgs(env *provider.TestEnvironment, nodeName string) (aMap map[string]string, er error) {
	o := clientsholder.GetClientsHolder()
	ctx := clientsholder.Context{Namespace: env.DebugPods[nodeName].Namespace,
		Podname:       env.DebugPods[nodeName].Name,
		Containername: env.DebugPods[nodeName].Spec.Containers[0].Name}
	bootConfig, errStr, err := o.ExecCommandContainer(ctx, grubKernelArgsCommand)
	if err != nil || errStr != "" {
		return aMap, fmt.Errorf("cannot exucute %s on debug pod %s, err=%s, stderr=%s", grubKernelArgsCommand, env.DebugPods[nodeName], err, errStr)
	}

	splitBootConfig := strings.Split(bootConfig, "\n")
	filteredBootConfig := arrayhelper.FilterArray(splitBootConfig, func(line string) bool {
		return strings.HasPrefix(line, "options")
	})
	if len(filteredBootConfig) != 1 {
		return aMap, fmt.Errorf("filteredBootConfig!=1")
	}
	grubKernelConfig := filteredBootConfig[0]
	grubSplitKernelConfig := strings.Split(grubKernelConfig, " ")
	grubSplitKernelConfig = grubSplitKernelConfig[1:]
	return arrayhelper.ArgListToMap(grubSplitKernelConfig), nil
}

func getCurrentKernelCmdlineArgs(cut *provider.Container) (aMap map[string]string, err error) {
	o := clientsholder.GetClientsHolder()
	ctx := clientsholder.Context{Namespace: cut.Namespace,
		Podname:       cut.Podname,
		Containername: cut.Data.Name}
	currnetKernelCmdlineArgs, errStr, err := o.ExecCommandContainer(ctx, kernelArgscommand)
	if err != nil || errStr != "" {
		return aMap, fmt.Errorf("cannot execute %s on container %s, err=%s, stderr=%s", grubKernelArgsCommand, cut, err, errStr)
	}
	currentSplitKernelCmdlineArgs := strings.Split(currnetKernelCmdlineArgs, " ")
	return arrayhelper.ArgListToMap(currentSplitKernelCmdlineArgs), nil
}
