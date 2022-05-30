package cmd

import (
	"context"
	"github.com/jsiebens/ionscale/pkg/gen/api"
	"github.com/muesli/coral"
	"github.com/rodaine/table"
)

func authMethodsCommand() *coral.Command {
	command := &coral.Command{
		Use:   "auth-methods",
		Short: "Manage ionscale auth methods",
	}

	command.AddCommand(listAuthMethods())
	command.AddCommand(createAuthMethodCommand())

	return command
}

func listAuthMethods() *coral.Command {
	command := &coral.Command{
		Use:   "list",
		Short: "List auth methods",
		Long: `List auth methods in this ionscale instance. Example:

      $ ionscale auth-methods list`,
	}

	var target = Target{}
	target.prepareCommand(command)

	command.RunE = func(command *coral.Command, args []string) error {

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		resp, err := client.ListAuthMethods(context.Background(), &api.ListAuthMethodsRequest{})

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME", "TYPE")
		for _, m := range resp.AuthMethods {
			tbl.AddRow(m.Id, m.Name, m.Type)
		}
		tbl.Print()

		return nil
	}

	return command
}

func createAuthMethodCommand() *coral.Command {
	command := &coral.Command{
		Use:   "create",
		Short: "Create a new auth method",
	}

	command.AddCommand(createOIDCAuthMethodCommand())

	return command
}

func createOIDCAuthMethodCommand() *coral.Command {
	command := &coral.Command{
		Use:          "oidc",
		Short:        "Create a new auth method",
		SilenceUsage: true,
	}

	var methodName string

	var clientId string
	var clientSecret string
	var issuer string

	var target = Target{}

	target.prepareCommand(command)
	command.Flags().StringVarP(&methodName, "name", "n", "", "")
	command.Flags().StringVar(&clientId, "client-id", "", "")
	command.Flags().StringVar(&clientSecret, "client-secret", "", "")
	command.Flags().StringVar(&issuer, "issuer", "", "")

	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("client-id")
	_ = command.MarkFlagRequired("client-secret")
	_ = command.MarkFlagRequired("issuer")

	command.RunE = func(command *coral.Command, args []string) error {

		client, c, err := target.createGRPCClient()
		if err != nil {
			return err
		}
		defer safeClose(c)

		req := &api.CreateAuthMethodRequest{
			Type:         "oidc",
			Name:         methodName,
			Issuer:       issuer,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		}

		resp, err := client.CreateAuthMethod(context.Background(), req)

		if err != nil {
			return err
		}

		tbl := table.New("ID", "NAME", "TYPE")
		tbl.AddRow(resp.AuthMethod.Id, resp.AuthMethod.Name, resp.AuthMethod.Type)
		tbl.Print()

		return nil
	}

	return command
}