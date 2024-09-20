package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"tailscale.com/tailcfg"
)

type DNSConfig struct {
	HttpsCertsEnabled bool                `json:"http_certs"`
	MagicDNS          bool                `json:"magic_dns"`
	OverrideLocalDNS  bool                `json:"override_local_dns"`
	Nameservers       []string            `json:"nameservers"`
	Routes            map[string][]string `json:"routes"`
	SearchDomains     []string            `json:"search_domains"`
	ExtraRecords      []tailcfg.DNSRecord `json:"extra_records"`
}

func (i *DNSConfig) Equal(x *DNSConfig) bool {
	if i == nil && x == nil {
		return true
	}
	if (i == nil) != (x == nil) {
		return false
	}

	return i.MagicDNS == x.MagicDNS &&
		i.HttpsCertsEnabled == x.HttpsCertsEnabled &&
		i.OverrideLocalDNS == x.OverrideLocalDNS &&
		reflect.DeepEqual(i.Nameservers, x.Nameservers) &&
		reflect.DeepEqual(i.Routes, x.Routes) &&
		reflect.DeepEqual(i.ExtraRecords, x.ExtraRecords) &&
		reflect.DeepEqual(i.SearchDomains, x.SearchDomains)
}

func (i *DNSConfig) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, i)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i DNSConfig) Value() (driver.Value, error) {
	bytes, err := json.Marshal(i)
	return bytes, err
}

// GormDataType gorm common data type
func (DNSConfig) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (DNSConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}
