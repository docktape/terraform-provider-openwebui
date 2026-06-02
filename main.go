package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/docktape/terraform-provider-openwebui/internal/provider"
)

func main() {
	if err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/docktape/openwebui",
	}); err != nil {
		log.Fatal(err)
	}
}
