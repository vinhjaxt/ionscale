package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	api "github.com/jsiebens/ionscale/pkg/gen/ionscale/v1"
	"tailscale.com/tailcfg"
)

func (s *Service) GetDNSConfig(ctx context.Context, req *connect.Request[api.GetDNSConfigRequest]) (*connect.Response[api.GetDNSConfigResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	resp := &api.GetDNSConfigResponse{
		Config: domainDNSConfigToApiDNSConfig(tailnet),
	}

	return connect.NewResponse(resp), nil
}

func (s *Service) SetDNSConfig(ctx context.Context, req *connect.Request[api.SetDNSConfigRequest]) (*connect.Response[api.SetDNSConfigResponse], error) {
	principal := CurrentPrincipal(ctx)
	if !principal.IsSystemAdmin() && !principal.IsTailnetAdmin(req.Msg.TailnetId) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied"))
	}

	dnsConfig := req.Msg.Config

	if dnsConfig.HttpsCerts && !dnsConfig.MagicDns {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("MagicDNS must be enabled when enabling HTTPS Certs"))
	}

	if dnsConfig.HttpsCerts && s.dnsProvider == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("A DNS provider must be configured when enabling HTTPS Certs"))
	}

	tailnet, err := s.repository.GetTailnet(ctx, req.Msg.TailnetId)
	if err != nil {
		return nil, logError(err)
	}
	if tailnet == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tailnet not found"))
	}

	oldConfig := tailnet.DNSConfig
	newConfig := apiDNSConfigToDomainDNSConfig(req.Msg.Config)

	if oldConfig.Equal(&newConfig) {
		return connect.NewResponse(&api.SetDNSConfigResponse{Config: domainDNSConfigToApiDNSConfig(tailnet)}), nil
	}

	tailnet.DNSConfig = newConfig
	if err := s.repository.SaveTailnet(ctx, tailnet); err != nil {
		return nil, logError(err)
	}

	s.sessionManager.NotifyAll(tailnet.ID)

	return connect.NewResponse(&api.SetDNSConfigResponse{Config: domainDNSConfigToApiDNSConfig(tailnet)}), nil
}

func domainRoutesToApiRoutes(routes map[string][]string) map[string]*api.Routes {
	var result = map[string]*api.Routes{}
	for k, v := range routes {
		result[k] = &api.Routes{Routes: v}
	}
	return result
}

func apiRoutesToDomainRoutes(routes map[string]*api.Routes) map[string][]string {
	var result = map[string][]string{}
	for k, v := range routes {
		result[k] = v.Routes
	}
	return result
}

func domainExtraRecordsToApiExtraRecords(cfgExtraRecords []tailcfg.DNSRecord) (extraRecords []string) {
	for _, r := range cfgExtraRecords {
		extraRecords = append(extraRecords, r.Name+":"+r.Type+":"+r.Value)
	}
	return
}

func apiExtraRecordsToDomainExtraRecords(extraRecords []string) (cfgExtraRecords []tailcfg.DNSRecord) {
	for _, r := range extraRecords {
		split := strings.SplitN(r, ":", 3) // domain:type:value
		if len(split) != 3 {
			continue
		}
		cfgExtraRecords = append(cfgExtraRecords, tailcfg.DNSRecord{
			Name:  split[0],
			Type:  split[1],
			Value: split[2],
		})
	}
	return
}

func apiDNSConfigToDomainDNSConfig(dnsConfig *api.DNSConfig) domain.DNSConfig {
	if dnsConfig == nil {
		return domain.DNSConfig{}
	}

	return domain.DNSConfig{
		MagicDNS:          dnsConfig.MagicDns,
		HttpsCertsEnabled: dnsConfig.HttpsCerts,
		OverrideLocalDNS:  dnsConfig.OverrideLocalDns,
		Nameservers:       dnsConfig.Nameservers,
		Routes:            apiRoutesToDomainRoutes(dnsConfig.Routes),
		SearchDomains:     dnsConfig.SearchDomains,
		ExtraRecords:      apiExtraRecordsToDomainExtraRecords(dnsConfig.ExtraRecords),
	}
}

func domainDNSConfigToApiDNSConfig(tailnet *domain.Tailnet) *api.DNSConfig {
	tailnetDomain := domain.SanitizeTailnetName(tailnet.Name)
	dnsConfig := tailnet.DNSConfig
	return &api.DNSConfig{
		MagicDns:         dnsConfig.MagicDNS,
		HttpsCerts:       dnsConfig.HttpsCertsEnabled,
		MagicDnsSuffix:   fmt.Sprintf("%s.%s", tailnetDomain, config.MagicDNSSuffix()),
		OverrideLocalDns: dnsConfig.OverrideLocalDNS,
		Nameservers:      dnsConfig.Nameservers,
		Routes:           domainRoutesToApiRoutes(dnsConfig.Routes),
		SearchDomains:    dnsConfig.SearchDomains,
		ExtraRecords:     domainExtraRecordsToApiExtraRecords(dnsConfig.ExtraRecords),
	}
}
