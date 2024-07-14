// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"github.com/paultyng/go-unifi/unifi"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New("test")()),
}

var testClient *unifiClient

func testAccPreCheck(t *testing.T) {
	// const user = "admin"
	// const password = "admin"
	// t.Setenv("UNIFI_USERNAME", user)
	// t.Setenv("UNIFI_PASSWORD", password)
	// t.Setenv("UNIFI_INSECURE", "true")
	// t.Setenv("UNIFI_URL", "https://localhost:8443")
}

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

func runAcceptanceTests(m *testing.M) int {
	dc, err := compose.NewDockerCompose("../../docker-compose.yml")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Workaround for https://github.com/testcontainers/testcontainers-go/issues/2621
	if err = os.Setenv("TESTCONTAINERS_RYUK_RECONNECTION_TIMEOUT", "5m"); err != nil {
		panic(err)
	}

	if err = dc.WithOsEnv().Up(ctx, compose.Wait(true)); err != nil {
		panic(err)
	}

	defer func() {
		if err := dc.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal); err != nil {
			panic(err)
		}
	}()

	container, err := dc.ServiceContainer(ctx, "unifi")
	if err != nil {
		panic(err)
	}

	// Dump the container logs on exit.
	//
	// TODO: Use https://pkg.go.dev/github.com/testcontainers/testcontainers-go#LogConsumer instead.
	defer func() {
		if os.Getenv("UNIFI_STDOUT") == "" {
			return
		}

		stream, err := container.Logs(ctx)
		if err != nil {
			panic(err)
		}

		buffer := new(bytes.Buffer)
		_, _ = buffer.ReadFrom(stream)
		testcontainers.Logger.Printf("%s", buffer)
	}()

	endpoint, err := container.PortEndpoint(ctx, "8443/tcp", "https")
	if err != nil {
		panic(err)
	}

	const user = "admin"
	const password = "admin"

	if err = os.Setenv("UNIFI_USERNAME", user); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_PASSWORD", password); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_INSECURE", "true"); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_URL", endpoint); err != nil {
		panic(err)
	}

	testClient = &unifiClient{
		Client: &unifi.Client{},
		site:   "default",
	}
	setHTTPClient(testClient, true)
	if err = testClient.SetBaseURL(endpoint); err != nil {
		panic(err)
	}
	if err = testClient.Login(ctx, user, password); err != nil {
		panic(err)
	}

	return m.Run()
}
