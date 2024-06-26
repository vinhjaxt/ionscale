package domain

import (
	"github.com/jsiebens/ionscale/pkg/client/ionscale"
	"net/netip"
	"strings"
	"tailscale.com/tailcfg"
)

func (a ACLPolicy) BuildSSHPolicy(srcs []Machine, dst *Machine) *tailcfg.SSHPolicy {
	var rules []*tailcfg.SSHRule

	expandSrcAliases := func(aliases []string, action string, u *User) []*tailcfg.SSHPrincipal {
		var allSrcIPsSet = &StringSet{}
		for _, alias := range aliases {
			if strings.HasPrefix(alias, "tag:") && action == "check" {
				continue
			}
			for _, src := range srcs {
				srcIPs := a.expandSSHSrcAlias(&src, alias, u)
				allSrcIPsSet.Add(srcIPs...)
			}
		}

		var result = []*tailcfg.SSHPrincipal{}
		for _, i := range allSrcIPsSet.Items() {
			result = append(result, &tailcfg.SSHPrincipal{NodeIP: i})
		}

		return result
	}

	expandRecorderAliases := func(aliases []string) []netip.AddrPort {
		result := make([]netip.AddrPort, 0)

		for _, alias := range aliases {
			for _, src := range append(srcs, *dst) {
				if src.HasTag(alias) {
					result = append(result, netip.AddrPortFrom(*src.IPv4.Addr, 80))
				}
			}
		}

		return result
	}

	for _, rule := range a.SSH {
		if rule.Action != "accept" && rule.Action != "check" {
			continue
		}

		var action = &tailcfg.SSHAction{
			Accept:                   true,
			AllowAgentForwarding:     true,
			AllowLocalPortForwarding: true,
		}

		if rule.Action == "check" {
			action = &tailcfg.SSHAction{
				HoldAndDelegate: "https://unused/machine/ssh/action/$SRC_NODE_ID/to/$DST_NODE_ID/" + safeCheckPeriod(rule.CheckPeriod),
			}
		}

		if len(rule.Recorder) != 0 {
			action.Recorders = expandRecorderAliases(rule.Recorder)
			action.Message = "# This session is being recorded.\n"
			if rule.EnforceRecorder {
				action.OnRecordingFailure = &tailcfg.SSHRecorderFailureAction{
					RejectSessionWithMessage:    "# Session rejected: failed to start session recording.",
					TerminateSessionWithMessage: "# Session terminated: failed to record session.",
				}
			}
		}

		selfUsers, otherUsers := a.expandSSHDstToSSHUsers(dst, rule)

		if len(selfUsers) != 0 {
			principals := expandSrcAliases(rule.Source, rule.Action, &dst.User)
			if len(principals) != 0 {
				rules = append(rules, &tailcfg.SSHRule{
					Principals: principals,
					SSHUsers:   selfUsers,
					Action:     action,
				})
			}
		}

		if len(otherUsers) != 0 {
			principals := expandSrcAliases(rule.Source, rule.Action, nil)
			if len(principals) != 0 {
				rules = append(rules, &tailcfg.SSHRule{
					Principals: principals,
					SSHUsers:   otherUsers,
					Action:     action,
				})
			}
		}
	}

	return &tailcfg.SSHPolicy{Rules: rules}
}

func (a ACLPolicy) expandSSHSrcAlias(m *Machine, alias string, dstUser *User) []string {
	if dstUser != nil {
		if !m.HasUser(dstUser.Name) || m.HasTags() {
			return []string{}
		}

		if alias == AutoGroupMember || alias == AutoGroupMembers {
			return m.IPs()
		}

		if strings.Contains(alias, "@") && m.HasUser(alias) {
			return m.IPs()
		}

		if strings.HasPrefix(alias, "group:") && a.isGroupMember(alias, m) {
			return m.IPs()
		}

		return []string{}
	}

	if (alias == AutoGroupMember || alias == AutoGroupMembers) && !m.HasTags() {
		return m.IPs()
	}

	if strings.Contains(alias, "@") && !m.HasTags() && m.HasUser(alias) {
		return m.IPs()
	}

	if strings.HasPrefix(alias, "group:") && !m.HasTags() && a.isGroupMember(alias, m) {
		return m.IPs()
	}

	if strings.HasPrefix(alias, "tag:") && m.HasTag(alias) {
		return m.IPs()
	}

	return []string{}
}

func (a ACLPolicy) expandSSHDstToSSHUsers(m *Machine, rule ionscale.ACLSSH) (map[string]string, map[string]string) {
	users := buildSSHUsers(rule.Users)

	var selfUsers map[string]string
	var otherUsers map[string]string

	for _, d := range rule.Destination {
		if strings.HasPrefix(d, "tag:") && m.HasTag(d) {
			otherUsers = users
		}

		if m.HasUser(d) || d == AutoGroupSelf {
			selfUsers = users
		}
	}

	return selfUsers, otherUsers
}

func buildSSHUsers(users []string) map[string]string {
	var autogroupNonRoot = false
	m := make(map[string]string)
	for _, u := range users {
		if u == "autogroup:nonroot" {
			m["*"] = "="
			autogroupNonRoot = true
		} else {
			m[u] = u
		}
	}

	// disable root when autogroup:nonroot is used and root is not explicitly enabled
	if _, exists := m["root"]; !exists && autogroupNonRoot {
		m["root"] = ""
	}

	return m
}

func safeCheckPeriod(period string) string {
	if period == "" {
		return "always"
	}
	return period
}
