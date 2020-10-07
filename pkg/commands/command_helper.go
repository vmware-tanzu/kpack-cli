// Copyright 2020-Present VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/pivotal/kpack/pkg/apis/build"
	"github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pivotal/build-service-cli/pkg/k8s"
)

type CommandHelper struct {
	dryRun bool
	output bool
	wait   bool

	outWriter io.Writer
	errWriter io.Writer

	objPrinter k8s.ObjectPrinter
	strBuilder strings.Builder

	typeToGVK map[reflect.Type]schema.GroupVersionKind
}

func NewCommandHelper(cmd *cobra.Command) (*CommandHelper, error) {
	dryRun, err := getBoolFlag("dry-run", cmd)
	if err != nil {
		return nil, err
	}

	output, err := getStringFlag("output", cmd)
	if err != nil {
		return nil, err
	}

	wait, err := getBoolFlag("wait", cmd)
	if err != nil {
		return nil, err
	}

	var objPrinter k8s.ObjectPrinter

	outputResource := len(output) > 0
	if outputResource {
		objPrinter, err = k8s.NewObjectPrinter(output)
		if err != nil {
			return nil, err
		}
	}

	return &CommandHelper{
		dryRun:     dryRun,
		output:     outputResource,
		wait:       wait,
		outWriter:  cmd.OutOrStdout(),
		errWriter:  cmd.ErrOrStderr(),
		objPrinter: objPrinter,
		strBuilder: strings.Builder{},
		typeToGVK:  getTypeToGVKLookup(),
	}, nil
}

func (ch CommandHelper) IsDryRun() bool {
	return ch.dryRun
}

func (ch CommandHelper) ShouldWait() bool {
	return ch.wait && !ch.dryRun && !ch.output
}

func (ch CommandHelper) PrintObjs(objs []runtime.Object) error {
	for _, obj := range objs {
		if err := ch.PrintObj(obj); err != nil {
			return err
		}
	}
	return nil
}

func (ch CommandHelper) PrintObj(obj runtime.Object) error {
	if !ch.output {
		return nil
	}

	oGVK := obj.GetObjectKind().GroupVersionKind()
	if oGVK.Version == "" || oGVK.Kind == "" {
		nGVK, ok := ch.typeToGVK[reflect.TypeOf(obj)]
		if !ok {
			return errors.Errorf("failed to output. unknown type %q", reflect.TypeOf(obj))
		}
		obj.GetObjectKind().SetGroupVersionKind(nGVK)
	}
	err := ch.objPrinter.PrintObject(obj, ch.outWriter)
	obj.GetObjectKind().SetGroupVersionKind(oGVK)
	return err
}

func (ch CommandHelper) PrintResult(format string, args ...interface{}) error {
	return ch.printDryRun(ch.OutOrDiscardWriter(), format, args...)
}

func (ch CommandHelper) PrintStatus(format string, args ...interface{}) error {
	return ch.printDryRun(ch.OutOrErrWriter(), format, args...)
}

func (ch CommandHelper) Printlnf(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(ch.OutOrErrWriter(), format+"\n", args...)
	return err
}

func (ch CommandHelper) OutOrErrWriter() io.Writer {
	if ch.output {
		return ch.errWriter
	} else {
		return ch.outWriter
	}
}

func (ch CommandHelper) OutOrDiscardWriter() io.Writer {
	if ch.output {
		return ioutil.Discard
	} else {
		return ch.outWriter
	}
}

func (ch CommandHelper) Writer() io.Writer {
	return ch.OutOrErrWriter()
}

func (ch CommandHelper) printDryRun(writer io.Writer, format string, a ...interface{}) error {
	ch.strBuilder.Reset()

	str := fmt.Sprintf(format, a...)
	_, err := ch.strBuilder.WriteString(str)
	if err != nil {
		return err
	}

	if ch.dryRun {
		_, err = ch.strBuilder.WriteString(" (dry run)")
		if err != nil {
			return err
		}
	}
	ch.strBuilder.WriteString("\n")

	_, err = writer.Write([]byte(ch.strBuilder.String()))
	return err
}

func getBoolFlag(name string, cmd *cobra.Command) (bool, error) {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return false, nil
	}

	if !cmd.Flags().Changed(name) {
		return false, nil
	}

	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return value, err
	}
	return value, nil
}

func getStringFlag(name string, cmd *cobra.Command) (string, error) {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return "", nil
	}

	if !cmd.Flags().Changed(name) {
		return "", nil
	}

	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return value, err
	}
	return value, nil
}

func getTypeToGVKLookup() map[reflect.Type]schema.GroupVersionKind {
	v1GV := schema.GroupVersion{Group: v1.GroupName, Version: "v1"}
	buildGV := schema.GroupVersion{Group: build.GroupName, Version: "v1alpha1"}

	return map[reflect.Type]schema.GroupVersionKind{
		reflect.TypeOf(&v1.Secret{}):               v1GV.WithKind("Secret"),
		reflect.TypeOf(&v1.ServiceAccount{}):       v1GV.WithKind("ServiceAccount"),
		reflect.TypeOf(&v1alpha1.Image{}):          buildGV.WithKind("Image"),
		reflect.TypeOf(&v1alpha1.Builder{}):        buildGV.WithKind(v1alpha1.BuilderKind),
		reflect.TypeOf(&v1alpha1.ClusterStack{}):   buildGV.WithKind(v1alpha1.ClusterStackKind),
		reflect.TypeOf(&v1alpha1.ClusterStore{}):   buildGV.WithKind(v1alpha1.ClusterStoreKind),
		reflect.TypeOf(&v1alpha1.ClusterBuilder{}): buildGV.WithKind(v1alpha1.ClusterBuilderKind),
	}
}