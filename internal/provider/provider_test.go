// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	// dc, err := compose.NewDockerCompose("../../docker-compose.yml")
	// if err != nil {
	// 	panic(err)
	// }

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// if err = dc.WithOsEnv().Up(ctx, compose.Wait(true)); err != nil {
	// 	panic(err)
	// }

	// defer func() {
	// 	if err := dc.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal); err != nil {
	// 		panic(err)
	// 	}
	// }()
	//
	// container, err := dc.ServiceContainer(ctx, "unifi")
	// if err != nil {
	// 	panic(err)
	// }

	// Dump the container logs on exit.
	//
	// TODO: Use https://pkg.go.dev/github.com/testcontainers/testcontainers-go#LogConsumer instead.
	// defer func() {
	// 	// if os.Getenv("UNIFI_STDOUT") == "" {
	// 	// 	return
	// 	// }
	//
	// 	stream, err := container.Logs(ctx)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	buffer := new(bytes.Buffer)
	// 	buffer.ReadFrom(stream)
	// 	testcontainers.Logger.Printf("%s", buffer)
	// }()

	// endpoint, err := container.PortEndpoint(ctx, "8443/tcp", "https")
	// if err != nil {
	// 	panic(err)
	// }

	const user = "admin"
	const password = "admin"

	if err := os.Setenv("UNIFI_USERNAME", user); err != nil {
		panic(err)
	}

	if err := os.Setenv("UNIFI_PASSWORD", password); err != nil {
		panic(err)
	}

	if err := os.Setenv("UNIFI_INSECURE", "true"); err != nil {
		panic(err)
	}

	if err := os.Setenv("UNIFI_URL", "localhost:8443"); err != nil {
		panic(err)
	}
}
