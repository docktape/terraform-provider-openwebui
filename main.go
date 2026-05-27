package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/docktape/terraform-provider-openwebui/internal/provider"
)

func main() {
	providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/docktape/openwebui",
	})
}
