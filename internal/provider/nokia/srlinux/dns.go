// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package srlinux

import (
	"context"
	"fmt"

	"github.com/ironcore-dev/network-operator/api/core/v1alpha1"
	"github.com/ironcore-dev/network-operator/internal/apistatus"
	"github.com/ironcore-dev/network-operator/internal/provider"
	"github.com/ironcore-dev/network-operator/internal/transport/gnmiext"
)

var _ provider.DNSProvider = (*Provider)(nil)

func (p *Provider) EnsureDNS(ctx context.Context, req *provider.EnsureDNSRequest) error {
	spec := req.DNS.Spec
	if err := validateDNSSpec(spec); err != nil {
		return err
	}

	// Group servers by VRF. Each unique VRF maps to one dns-instance on Nokia SRL.
	instances := make(map[string]*DNSInstance)
	for _, s := range spec.Servers {
		inst, ok := instances[s.VrfName]
		if !ok {
			inst = &DNSInstance{
				VrfName:         s.VrfName,
				NetworkInstance: s.VrfName,
				SearchList:      []string{spec.Domain},
			}
			instances[s.VrfName] = inst
		}
		inst.ServerList = append(inst.ServerList, s.Address)
	}

	elements := make([]gnmiext.DataElement, 0, len(instances))
	for _, inst := range instances {
		elements = append(elements, inst)
	}
	return p.client.Update(ctx, elements...)
}

func (p *Provider) DeleteDNS(ctx context.Context) error {
	return p.client.Delete(ctx, &DNSInstanceList{})
}

func validateDNSSpec(spec v1alpha1.DNSSpec) error {
	var violations []apistatus.FieldViolation
	if spec.AdminState == v1alpha1.AdminStateDown {
		violations = append(violations, apistatus.FieldViolation{
			Field:       "spec.adminState",
			Description: "adminState Down is not supported by the Nokia SR Linux DNS model",
		})
	}
	if spec.SourceInterfaceName != "" {
		violations = append(violations, apistatus.FieldViolation{
			Field:       "spec.sourceInterfaceName",
			Description: "sourceInterfaceName is not supported by the Nokia SR Linux DNS model",
		})
	}
	for _, s := range spec.Servers {
		if s.VrfName == "" {
			violations = append(violations, apistatus.FieldViolation{
				Field:       fmt.Sprintf("spec.servers[%s].vrfName", s.Address),
				Description: "vrfName is required by the Nokia SR Linux DNS model — network-instance is mandatory",
			})
		}
	}
	if len(violations) > 0 {
		return apistatus.NewUnsupportedFieldError(violations...)
	}
	return nil
}

// Compile-time assertions.
var (
	_ gnmiext.DataElement = (*DNSInstance)(nil)
	_ gnmiext.DataElement = (*DNSInstanceList)(nil)
)

// DNSInstance represents a single Nokia SRL dns-instance list entry, keyed by VRF name.
type DNSInstance struct {
	VrfName         string   `json:"-"`
	NetworkInstance string   `json:"network-instance"`
	SearchList      []string `json:"search-list,omitempty"`
	ServerList      []string `json:"server-list,omitempty"`
}

func (d *DNSInstance) XPath() string {
	return fmt.Sprintf("srl_nokia-system:system/srl_nokia-dns:dns-instance[name=%s]", d.VrfName)
}

// DNSInstanceList targets the entire dns-instance list for deletion.
type DNSInstanceList struct{}

func (*DNSInstanceList) XPath() string {
	return "srl_nokia-system:system/srl_nokia-dns:dns-instance"
}
