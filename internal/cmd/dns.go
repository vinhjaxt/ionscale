package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bufbuild/connect-go"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"github.com/spf13/cobra"
)

func getDNSConfigCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "get-dns",
		Short:        "Get DNS configuration",
		SilenceUsage: true,
	})

	command.RunE = func(cmd *cobra.Command, args []string) error {
		req := api.GetDNSConfigRequest{TailnetId: tc.TailnetID()}
		resp, err := tc.Client().GetDNSConfig(cmd.Context(), connect.NewRequest(&req))

		if err != nil {
			return err
		}
		config := resp.Msg.Config

		printDnsConfig(config)

		return nil
	}

	return command
}

func setDNSConfigCommand() *cobra.Command {
	command, tc := prepareCommand(true, &cobra.Command{
		Use:          "set-dns",
		Short:        "Set DNS config",
		SilenceUsage: true,
	})

	var extraRecords []string
	var nameservers []string
	var magicDNS bool
	var httpsCerts bool
	var overrideLocalDNS bool
	var searchDomains []string

	command.Flags().StringSliceVarP(&nameservers, "nameserver", "", []string{}, "Machines on your network will use these nameservers to resolve DNS queries.")
	command.Flags().BoolVarP(&magicDNS, "magic-dns", "", false, "Enable MagicDNS for the specified Tailnet")
	command.Flags().BoolVarP(&httpsCerts, "https-certs", "", false, "Enable HTTPS Certificates for the specified Tailnet")
	command.Flags().BoolVarP(&overrideLocalDNS, "override-local-dns", "", false, "When enabled, connected clients ignore local DNS settings and always use the nameservers specified for this Tailnet")
	command.Flags().StringSliceVarP(&searchDomains, "search-domain", "", []string{}, "Custom DNS search domains.")
	command.Flags().StringSliceVarP(&extraRecords, "extra-records", "", []string{}, "Extra DNS records. Eg: mail.domain.tld::100.123.4.5")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		var globalNameservers []string
		var routes = make(map[string]*api.Routes)

		for _, n := range nameservers {
			if strings.HasPrefix(n, `http://`) || strings.HasPrefix(n, `https://`) { // doh
				globalNameservers = append(globalNameservers, n)
				continue
			}
			split := strings.SplitN(n, ":", 3)
			if len(split) == 2 {
				r, ok := routes[split[0]]
				if ok {
					r.Routes = append(r.Routes, split[1])
				} else {
					routes[split[0]] = &api.Routes{Routes: []string{split[1]}}
				}
				continue
			}
			globalNameservers = append(globalNameservers, n)
		}

		req := api.SetDNSConfigRequest{
			TailnetId: tc.TailnetID(),
			Config: &api.DNSConfig{
				MagicDns:         magicDNS,
				OverrideLocalDns: overrideLocalDNS,
				Nameservers:      globalNameservers,
				Routes:           routes,
				HttpsCerts:       httpsCerts,
				SearchDomains:    searchDomains,
				ExtraRecords:     extraRecords,
			},
		}
		resp, err := tc.Client().SetDNSConfig(cmd.Context(), connect.NewRequest(&req))

		if err != nil {
			return err
		}

		config := resp.Msg.Config

		if resp.Msg.Message != "" {
			fmt.Println(resp.Msg.Message)
			fmt.Println()
		}

		printDnsConfig(config)

		return nil
	}

	return command
}

func printDnsConfig(config *api.DNSConfig) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\t\t%v\n", "MagicDNS", config.MagicDns)
	fmt.Fprintf(w, "%s\t\t%v\n", "HTTPS Certs", config.HttpsCerts)
	fmt.Fprintf(w, "%s\t\t%v\n", "Override Local DNS", config.OverrideLocalDns)

	if config.MagicDns {
		fmt.Fprintf(w, "MagicDNS\t%s\t%s\n", config.MagicDnsSuffix, "100.100.100.100")
	}

	for k, r := range config.Routes {
		for i, t := range r.Routes {
			if i == 0 {
				fmt.Fprintf(w, "SplitDNS\t%s\t%s\n", k, t)
			} else {
				fmt.Fprintf(w, "%s\t%s\n", "", t)
			}
		}
	}

	for i, t := range config.Nameservers {
		if i == 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "Global", "", t)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "", "", t)
		}
	}

	for i, t := range config.SearchDomains {
		if i == 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "Search Domains", t, "")
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "", t, "")
		}
	}

	for i, t := range config.ExtraRecords {
		if i == 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "Extra DNS Records", t, "")
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n", "", t, "")
		}
	}
}
