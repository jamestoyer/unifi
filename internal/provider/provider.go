// Copyright (c) James Toyer.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/paultyng/go-unifi/unifi"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
)

// Ensure UnifiProvider satisfies various provider interfaces.
var _ provider.Provider = &UnifiProvider{}
var _ provider.ProviderWithFunctions = &UnifiProvider{}

// UnifiProvider defines the provider implementation.
type UnifiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type unifiClient struct {
	*unifi.Client
	site string
}

// UnifiProviderModel describes the provider data model.
type UnifiProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	URL      types.String `tfsdk:"url"`
	Site     types.String `tfsdk:"site"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

func (p *UnifiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unifi"
	resp.Version = p.version
}

func (p *UnifiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "Local user name for the Unifi controller API. Can be specified with the `UNIFI_USERNAME` " +
					"environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the user accessing the API. Can be specified with the `UNIFI_PASSWORD` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "URL of the controller. Can be specified with the `UNIFI_URL` environment variable. " +
					"You should **NOT** supply the path (`/api`), the SDK will discover the appropriate paths. This is " +
					"to support UDM Pro style API paths as well as more standard controller paths.",
				Optional: true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site in the Unifi controller this provider will manage. Can be specified with " +
					"the `UNIFI_SITE` environment variable. Default: `default`",
				Optional: true,
			},
			"insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip verification of TLS certificates of API requests. You may need to set this to `true` " +
					"if you are using your local API without setting up a signed certificate. Can be specified with the " +
					"`UNIFI_INSECURE` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *UnifiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data UnifiProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	url := os.Getenv("UNIFI_URL")
	username := os.Getenv("UNIFI_USERNAME")
	password := os.Getenv("UNIFI_PASSWORD")
	site := os.Getenv("UNIFI_SITE")
	var insecure bool

	if !data.URL.IsNull() {
		url = data.URL.ValueString()
	}

	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}

	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	if !data.Site.IsNull() {
		site = data.Password.ValueString()
	}

	if !data.Insecure.IsNull() {
		insecure = data.Insecure.ValueBool()
	} else {
		val, err := strconv.ParseBool(os.Getenv("UNIFI_INSECURE"))
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("insecure"),
				"Invalid insecure value",
				"The provider cannot create the Unifi client as the value for UNIFI_INSECURE is invalid.",
			)
		}

		insecure = val
	}

	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing Controller URL",
			"The provider cannot create the Unifi API client as there is a missing or empty value for the Unifi url. "+
				"Set the url value in the configuration or use the UNIFI_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Controller username",
			"The provider cannot create the Unifi API client as there is a missing or empty value for the Unifi username. "+
				"Set the username value in the configuration or use the UNIFI_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Controller Password",
			"The provider cannot create the Unifi API client as there is a missing or empty value for the Unifi password. "+
				"Set the password value in the configuration or use the UNIFI_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if site == "" {
		site = "default"
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := &unifiClient{
		Client: new(unifi.Client),
		site:   site,
	}
	setHTTPClient(client, insecure)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *UnifiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
	}
}

func (p *UnifiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func (p *UnifiProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UnifiProvider{
			version: version,
		}
	}
}

func setHTTPClient(c *unifiClient, insecure bool) {
	httpClient := &http.Client{}
	httpClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}

	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	_ = c.SetHTTPClient(httpClient)
}
