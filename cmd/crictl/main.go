/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

const (
	defaultTimeout = 10 * time.Second
)

var (
	// RuntimeEndpoint is CRI server runtime endpoint (default: "/var/run/dockershim.sock")
	RuntimeEndpoint string
	// ImageEndpoint is CRI server image endpoint, default same as runtime endpoint
	ImageEndpoint string
	// Timeout  of connecting to server (default: 10s)
	Timeout time.Duration
	// Debug enable debug output
	Debug bool
)

func getRuntimeClientConnection(context *cli.Context) (*grpc.ClientConn, error) {
	if RuntimeEndpoint == "" {
		return nil, fmt.Errorf("--runtime-endpoint is not set")
	}
	conn, err := grpc.Dial(RuntimeEndpoint, grpc.WithInsecure(), grpc.WithTimeout(Timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return conn, nil
}
func getImageClientConnection(context *cli.Context) (*grpc.ClientConn, error) {
	if ImageEndpoint == "" {
		if RuntimeEndpoint == "" {
			return nil, fmt.Errorf("--image-endpoint is not set")
		}
		ImageEndpoint = RuntimeEndpoint
	}
	conn, err := grpc.Dial(ImageEndpoint, grpc.WithInsecure(), grpc.WithTimeout(Timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return conn, nil
}

func main() {
	app := cli.NewApp()
	app.Name = "crictl"
	app.Usage = "client for CRI"
	app.Version = "0.1.0"

	app.Commands = []cli.Command{
		runtimeVersionCommand,
		runtimePodSandboxCommand,
		runtimeContainerCommand,
		runtimeStatusCommand,
		runtimeAttachCommand,
		imageCommand,
		runtimeExecCommand,
		runtimePortForwardCommand,
		logsCommand,
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config-file, c",
			EnvVar: "CRI_CONFIG_FILE",
			Value:  "",
			Usage:  "config file for crictl",
		},
		cli.StringFlag{
			Name:   "runtime-endpoint, r",
			EnvVar: "CRI_RUNTIME_ENDPOINT",
			Value:  "/var/run/dockershim.sock",
			Usage:  "endpoint for CRI container runtime",
		},
		cli.StringFlag{
			Name:   "image-endpoint, i",
			EnvVar: "CRI_IMAGE_ENDPOINT",
			Usage:  "endpoint for CRI image service",
		},
		cli.DurationFlag{
			Name:  "timeout",
			Value: defaultTimeout,
			Usage: "Timeout of connecting to server",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug output",
		},
	}

	app.Before = func(context *cli.Context) error {
		configFile := context.GlobalString("config-file")
		if configFile == "" {
			RuntimeEndpoint = context.GlobalString("runtime-endpoint")
			ImageEndpoint = context.GlobalString("image-endpoint")
			Timeout = context.GlobalDuration("timeout")
			Debug = context.GlobalBool("debug")
		} else {
			// Get config from file.
			config, err := ReadConfig(configFile)
			if err != nil {
				logrus.Fatalf("Falied to load config file: %v", err)
			}

			// Command line flags overrides config file.
			if context.IsSet("runtime-endpoint") {
				RuntimeEndpoint = context.String("runtime-endpoint")
			} else if config.RuntimeEndpoint != "" {
				RuntimeEndpoint = config.RuntimeEndpoint
			} else {
				RuntimeEndpoint = context.GlobalString("runtime-endpoint")
			}
			if context.IsSet("image-endpoint") {
				ImageEndpoint = context.String("image-endpoint")
			} else if config.ImageEndpoint != "" {
				ImageEndpoint = config.ImageEndpoint
			} else {
				ImageEndpoint = context.GlobalString("image-endpoint")
			}
			if context.IsSet("timeout") {
				Timeout = context.Duration("timeout")
			} else if config.Timeout != 0 {
				Timeout = time.Duration(config.Timeout) * time.Second
			} else {
				Timeout = context.GlobalDuration("timeout")
			}
			if context.IsSet("debug") {
				Debug = context.GlobalBool("debug")
			} else {
				Debug = config.Debug
			}
		}

		if Debug {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
