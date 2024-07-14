// Copyright (c) James Toyer
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/paultyng/go-unifi/unifi"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New("test")()),
}

var (
	testClient  *unifiClient
	testContext struct {
		username string
		password string
		url      string
	}
)

type logConsumer struct {
	logger *log.Logger
}

func (c *logConsumer) Accept(l testcontainers.Log) {
	c.logger.Printf(string(l.Content))
}

func testAccPreCheck(t *testing.T) {
	t.Setenv("UNIFI_USERNAME", testContext.username)
	t.Setenv("UNIFI_PASSWORD", testContext.password)
	t.Setenv("UNIFI_INSECURE", "true")
	t.Setenv("UNIFI_URL", testContext.url)
}

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

func runAcceptanceTests(m *testing.M) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	unifiPackageURL := ""
	if packageURL := os.Getenv("UNIFI_PACKAGE_URL"); packageURL != "" {
		unifiPackageURL = packageURL
	}

	unifiStdOut := ""
	var logConsumers []testcontainers.LogConsumer
	if stdOut := os.Getenv("UNIFI_STDOUT"); stdOut != "" {
		unifiStdOut = stdOut
		logConsumers = append(logConsumers, &logConsumer{
			logger: log.New(os.Stderr, "", log.LstdFlags),
		})
	}

	unifiVersion := "latest"
	if version := os.Getenv("UNIFI_VERSION"); version != "" {
		unifiVersion = version
	}

	demoModePath, err := filepath.Abs("../../scripts/init.d/demo-mode")
	if err != nil {
		panic(err)
	}

	r, err := os.Open(demoModePath)
	if err != nil {
		panic(err)
	}

	req := testcontainers.ContainerRequest{
		Name:  "unifi",
		Image: "jacobalberty/unifi:" + unifiVersion,
		Env: map[string]string{
			"PKGURL":       unifiPackageURL,
			"UNIFI_STDOUT": unifiStdOut,
		},
		ExposedPorts: []string{"8443/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            r,
				HostFilePath:      demoModePath, // will be discarded internally
				ContainerFilePath: "/usr/local/unifi/init.d/demo-mode",
				FileMode:          0o700,
			},
		},
		LogConsumerCfg: &testcontainers.LogConsumerConfig{
			Consumers: logConsumers,
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("<launcher> INFO  tomcat - systemd: Startup completed. Ready for watchdog keep-alive checking.").
				WithStartupTimeout(10 * time.Minute).
				WithPollInterval(1 * time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := container.Terminate(ctx); err != nil {
			panic(err)
		}
	}()

	endpoint, err := container.PortEndpoint(ctx, "8443/tcp", "https")
	if err != nil {
		panic(err)
	}

	testContext.username = "admin"
	testContext.password = "admin"
	testContext.url = endpoint

	testClient = &unifiClient{
		Client: &unifi.Client{},
		site:   "default",
	}
	setHTTPClient(testClient, true)
	if err = testClient.SetBaseURL(endpoint); err != nil {
		panic(err)
	}

	if err = testClient.Login(ctx, testContext.username, testContext.password); err != nil {
		panic(err)
	}

	return m.Run()
}
