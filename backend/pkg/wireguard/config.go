// Package wireguard рендерит .conf для клиентов и серверов.
package wireguard

import (
	"fmt"
	"strings"

	"github.com/voidwg/control/internal/domain"
)

// RenderClientConfig возвращает текст wg-quick конфига клиента.
func RenderClientConfig(p *domain.Peer, srv *domain.Server, privateKey string) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", privateKey)
	fmt.Fprintf(&b, "Address = %s/32\n", p.AssignedIP.String())

	if len(srv.DNS) > 0 {
		dns := make([]string, 0, len(srv.DNS))
		for _, a := range srv.DNS {
			dns = append(dns, a.String())
		}
		fmt.Fprintf(&b, "DNS = %s\n", strings.Join(dns, ", "))
	}
	b.WriteString("\n[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", srv.PublicKey)
	if p.PresharedKey != "" {
		fmt.Fprintf(&b, "PresharedKey = %s\n", p.PresharedKey)
	}
	fmt.Fprintf(&b, "Endpoint = %s\n", srv.Endpoint)
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString("PersistentKeepalive = 25\n")
	return b.String()
}
