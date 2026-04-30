// Package wireguard рендерит wg-quick / AmneziaWG конфиги.
package wireguard

import (
	"fmt"
	"strings"

	"github.com/voidwg/control/internal/domain"
)

// RenderClientConfigStub возвращает wg-quick конфиг с пометкой
// `PrivateKey = <PASTE_YOUR_PRIVATE_KEY>` — клиент сам подставляет приватник.
//
// В секцию [Interface] добавляются AmneziaWG-параметры (Jc/Jmin/Jmax/S1/S2/H1-H4),
// если на сервере включена обфускация. amneziawg-go и совместимые клиенты
// (AmneziaVPN) их понимают; чистый wg-quick — игнорирует и работает как обычный WG.
func RenderClientConfigStub(p *domain.Peer, srv *domain.Server) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = <PASTE_YOUR_PRIVATE_KEY>\n")
	fmt.Fprintf(&b, "Address = %s/32\n", p.AssignedIP.String())

	if len(srv.DNS) > 0 {
		dns := make([]string, 0, len(srv.DNS))
		for _, a := range srv.DNS {
			dns = append(dns, a.String())
		}
		fmt.Fprintf(&b, "DNS = %s\n", strings.Join(dns, ", "))
	}
	if srv.ObfsEnabled {
		writeAWGParams(&b, srv.AWG)
	}

	b.WriteString("\n[Peer]\n")
	fmt.Fprintf(&b, "PublicKey = %s\n", srv.PublicKey)
	fmt.Fprintf(&b, "Endpoint = %s\n", srv.Endpoint)
	b.WriteString("AllowedIPs = 0.0.0.0/0, ::/0\n")
	b.WriteString("PersistentKeepalive = 25\n")
	return b.String()
}

// writeAWGParams — пишет параметры в формате amneziawg-go (regex-friendly).
func writeAWGParams(b *strings.Builder, a domain.AWGParams) {
	fmt.Fprintf(b, "Jc = %d\n",   a.Jc)
	fmt.Fprintf(b, "Jmin = %d\n", a.Jmin)
	fmt.Fprintf(b, "Jmax = %d\n", a.Jmax)
	fmt.Fprintf(b, "S1 = %d\n",   a.S1)
	fmt.Fprintf(b, "S2 = %d\n",   a.S2)
	fmt.Fprintf(b, "H1 = %d\n",   a.H1)
	fmt.Fprintf(b, "H2 = %d\n",   a.H2)
	fmt.Fprintf(b, "H3 = %d\n",   a.H3)
	fmt.Fprintf(b, "H4 = %d\n",   a.H4)
}
