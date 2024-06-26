package dns

import (
	"context"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/mapping"
	"github.com/libdns/azure"
	"github.com/libdns/cloudflare"
	"github.com/libdns/digitalocean"
	"github.com/libdns/googleclouddns"
	"github.com/libdns/libdns"
	"github.com/libdns/route53"
	"strings"
	"time"
)

type Provider interface {
	SetRecord(ctx context.Context, recordType, recordName, value string) error
}

func NewProvider(config config.DNS) (Provider, error) {
	p := config.Provider
	if len(p.Zone) == 0 {
		return nil, nil
	}

	if !strings.HasSuffix(config.MagicDNSSuffix, p.Zone) {
		return nil, fmt.Errorf("invalid MagicDNS suffix [%s], not part of zone [%s]", config.MagicDNSSuffix, p.Zone)
	}

	switch p.Name {
	case "azure":
		return configureAzureProvider(p.Zone, p.Configuration)
	case "cloudflare":
		return configureCloudflareProvider(p.Zone, p.Configuration)
	case "digitalocean":
		return configureDigitalOceanProvider(p.Zone, p.Configuration)
	case "googleclouddns":
		return configureGoogleCloudDNSProvider(p.Zone, p.Configuration)
	case "route53":
		return configureRoute53Provider(p.Zone, p.Configuration)
	default:
		return nil, fmt.Errorf("unknown dns provider: %s", p.Name)
	}
}

func configureAzureProvider(zone string, values map[string]string) (Provider, error) {
	p := &azure.Provider{}
	if err := mapping.CopyViaJson(values, p); err != nil {
		return nil, err
	}

	e := &azure.Provider{
		TenantId:          config.GetString("IONSCALE_DNS_AZURE_TENANT_ID", ""),
		ClientId:          config.GetString("IONSCALE_DNS_AZURE_CLIENT_ID", ""),
		ClientSecret:      config.GetString("IONSCALE_DNS_AZURE_CLIENT_SECRET", ""),
		SubscriptionId:    config.GetString("IONSCALE_DNS_AZURE_SUBSCRIPTION_ID", ""),
		ResourceGroupName: config.GetString("IONSCALE_DNS_AZURE_RESOURCE_GROUP_NAME", ""),
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(p, e, mergo.WithOverride); err != nil {
		return nil, err
	}

	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func configureCloudflareProvider(zone string, values map[string]string) (Provider, error) {
	p := &cloudflare.Provider{}
	if err := mapping.CopyViaJson(values, p); err != nil {
		return nil, err
	}

	e := &cloudflare.Provider{
		APIToken: config.GetString("IONSCALE_DNS_CLOUDFLARE_API_TOKEN", ""),
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(p, e, mergo.WithOverride); err != nil {
		return nil, err
	}

	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func configureDigitalOceanProvider(zone string, values map[string]string) (Provider, error) {
	p := &digitalocean.Provider{}
	if err := mapping.CopyViaJson(values, p); err != nil {
		return nil, err
	}

	e := &digitalocean.Provider{
		APIToken: config.GetString("IONSCALE_DNS_DIGITALOCEAN_API_TOKEN", ""),
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(p, e, mergo.WithOverride); err != nil {
		return nil, err
	}

	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func configureGoogleCloudDNSProvider(zone string, values map[string]string) (Provider, error) {
	p := &googleclouddns.Provider{}
	if err := mapping.CopyViaJson(values, p); err != nil {
		return nil, err
	}

	e := &googleclouddns.Provider{
		Project:            config.GetString("IONSCALE_DNS_GOOGLECLOUDDNS_PROJECT", ""),
		ServiceAccountJSON: config.GetString("IONSCALE_DNS_GOOGLECLOUDDNS_SERVICE_ACCOUNT_JSON", ""),
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(p, e, mergo.WithOverride); err != nil {
		return nil, err
	}

	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

func configureRoute53Provider(zone string, values map[string]string) (Provider, error) {
	p := &route53.Provider{}
	if err := mapping.CopyViaJson(values, p); err != nil {
		return nil, err
	}

	e := &route53.Provider{
		MaxRetries:         0,
		MaxWaitDur:         0,
		WaitForPropagation: false,
		Region:             config.GetString("IONSCALE_DNS_ROUTE53_REGION", ""),
		AWSProfile:         config.GetString("IONSCALE_DNS_ROUTE53_AWS_PROFILE", ""),
		AccessKeyId:        config.GetString("IONSCALE_DNS_ROUTE53_ACCESS_KEY_ID", ""),
		SecretAccessKey:    config.GetString("IONSCALE_DNS_ROUTE53_SECRET_ACCESS_KEY", ""),
		Token:              config.GetString("IONSCALE_DNS_ROUTE53_TOKEN", ""),
	}

	// merge env configuration on top of the default/file configuration
	if err := mergo.Merge(p, e, mergo.WithOverride); err != nil {
		return nil, err
	}

	return &externalProvider{zone: fqdn(zone), setter: p}, nil
}

type externalProvider struct {
	zone   string
	setter libdns.RecordSetter
}

func (p *externalProvider) SetRecord(ctx context.Context, recordType, recordName, value string) error {
	_, err := p.setter.SetRecords(ctx, p.zone, []libdns.Record{{
		Type:  recordType,
		Name:  libdns.RelativeName(recordName, p.zone),
		Value: value,
		TTL:   1 * time.Minute,
	}})
	return err
}

func fqdn(v string) string {
	if strings.HasSuffix(v, ".") {
		return v
	}
	return fmt.Sprintf("%s.", v)
}
