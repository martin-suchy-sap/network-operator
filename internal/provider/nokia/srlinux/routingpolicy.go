// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package srlinux

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ironcore-dev/network-operator/api/core/v1alpha1"
	"github.com/ironcore-dev/network-operator/internal/apistatus"
	"github.com/ironcore-dev/network-operator/internal/provider"
	"github.com/ironcore-dev/network-operator/internal/transport/gnmiext"
)

var _ provider.RoutingPolicyProvider = (*Provider)(nil)

func (p *Provider) EnsureRoutingPolicy(ctx context.Context, req *provider.EnsureRoutingPolicyRequest) error {
	pol := &RoutingPolicy{Name: req.Name}

	for _, stmt := range req.Statements {
		name := strconv.Itoa(int(stmt.Sequence))
		s := &RoutingPolicyStatement{
			Name: name,
			Action: RoutingPolicyAction{
				PolicyResult: toSRLPolicyResult(stmt.Actions.RouteDisposition),
			},
		}

		for _, cond := range stmt.Conditions {
			switch c := cond.(type) {
			case provider.MatchPrefixSetCondition:
				s.Match = &RoutingPolicyMatch{
					Prefix: &RoutingPolicyPrefixMatch{
						PrefixSet:       c.PrefixSet.Spec.Name,
						MatchSetOptions: MatchSetOptionsAny,
					},
				}
			case provider.MatchCommunitySetCondition:
				return apistatus.NewUnsupportedFieldError(apistatus.FieldViolation{
					Field:       "spec.statements[].conditions.matchCommunitySet",
					Description: "Nokia SR Linux native provider does not support community-set match conditions",
				})
			}
		}

		if stmt.Actions.BgpActions != nil {
			if stmt.Actions.BgpActions.SetCommunity != nil {
				return apistatus.NewUnsupportedFieldError(apistatus.FieldViolation{
					Field:       "spec.statements[].actions.bgpActions.setCommunity",
					Description: "Nokia SR Linux native provider does not support setCommunity BGP action",
				})
			}
			if stmt.Actions.BgpActions.SetExtCommunity != nil {
				return apistatus.NewUnsupportedFieldError(apistatus.FieldViolation{
					Field:       "spec.statements[].actions.bgpActions.setExtCommunity",
					Description: "Nokia SR Linux native provider does not support setExtCommunity BGP action",
				})
			}
			if stmt.Actions.BgpActions.SetASPath != nil {
				return apistatus.NewUnsupportedFieldError(apistatus.FieldViolation{
					Field:       "spec.statements[].actions.bgpActions.setASPath",
					Description: "Nokia SR Linux native provider does not support AS-path manipulation",
				})
			}
			if stmt.Actions.BgpActions.SetNextHop != nil {
				return apistatus.NewUnsupportedFieldError(apistatus.FieldViolation{
					Field:       "spec.statements[].actions.bgpActions.setNextHop",
					Description: "Nokia SR Linux native provider does not support setNextHop BGP action",
				})
			}
		}

		pol.Statement.Set(s)
	}

	return p.client.Update(ctx, pol)
}

func (p *Provider) DeleteRoutingPolicy(ctx context.Context, req *provider.DeleteRoutingPolicyRequest) error {
	return p.client.Delete(ctx, &RoutingPolicy{Name: req.Name})
}

func toSRLPolicyResult(rd v1alpha1.RouteDisposition) SRLPolicyResult {
	switch rd {
	case v1alpha1.AcceptRoute:
		return SRLPolicyResultAccept
	case v1alpha1.RejectRoute:
		return SRLPolicyResultReject
	default:
		return SRLPolicyResultReject
	}
}

// SRLPolicyResult represents the SR Linux native routing policy result values.
type SRLPolicyResult string

const (
	SRLPolicyResultAccept        SRLPolicyResult = "accept"
	SRLPolicyResultReject        SRLPolicyResult = "reject"
	SRLPolicyResultNextPolicy    SRLPolicyResult = "next-policy"
	SRLPolicyResultNextStatement SRLPolicyResult = "next-statement"
)

// MatchSetOptions represents the match-set-options for prefix matching.
type MatchSetOptions string

const (
	MatchSetOptionsAny MatchSetOptions = "any"
	MatchSetOptionsAll MatchSetOptions = "all"
)

var _ gnmiext.DataElement = (*RoutingPolicy)(nil)

// RoutingPolicy targets a single policy entry in the SR Linux native routing-policy container.
type RoutingPolicy struct {
	Name      string                                        `json:"name"`
	Statement gnmiext.List[string, *RoutingPolicyStatement] `json:"statement,omitempty"`
}

func (rp *RoutingPolicy) XPath() string {
	return fmt.Sprintf("srl_nokia-routing-policy:routing-policy/policy[name=%s]", rp.Name)
}

// RoutingPolicyStatement represents a single statement in a routing policy.
type RoutingPolicyStatement struct {
	Name   string              `json:"name"`
	Match  *RoutingPolicyMatch `json:"match,omitempty"`
	Action RoutingPolicyAction `json:"action"`
}

func (s *RoutingPolicyStatement) Key() string { return s.Name }

// RoutingPolicyMatch holds the match conditions for a statement.
type RoutingPolicyMatch struct {
	Prefix *RoutingPolicyPrefixMatch `json:"prefix,omitempty"`
}

// RoutingPolicyPrefixMatch holds the prefix-set match condition.
type RoutingPolicyPrefixMatch struct {
	PrefixSet       string          `json:"prefix-set"`
	MatchSetOptions MatchSetOptions `json:"match-set-options"`
}

// RoutingPolicyAction holds the action for a statement.
type RoutingPolicyAction struct {
	PolicyResult SRLPolicyResult `json:"policy-result"`
}
