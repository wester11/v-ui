# void-wg — Enhanced Obfuscated WireGuard Client

Продвинутый WireGuard клиент с обфускацией трафика.
Аналог AmneziaWG — обходит DPI и блокировки.

## Отличия от void-wg-d
| Фича | void-wg-d | void-wg |
|---|---|---|
| Стандартные .conf | ✓ | ✓ (импорт) |
| Формат .vwg | ✗ | ✓ |
| Обфускация (AWG) | ✗ | ✓ |
| Junk packets (Jc/Jmin/Jmax) | ✗ | ✓ |
| Handshake padding (S1/S2) | ✗ | ✓ |
| Magic XOR (H1-H4) | ✗ | ✓ |
| Upgrade .conf → .vwg | ✗ | ✓ |
| Kill switch | ✓ | ✓ |

## Установка
```bash
cd clients/void-wg
go build -o void-wg ./cmd/void-wg
sudo mv void-wg /usr/local/bin/
```

## Использование
```bash
# Поднять с обфускацией
sudo void-wg up myvpn.vwg --kill-switch

# Обновить обычный WG конфиг до .vwg
void-wg upgrade ~/Downloads/vpn.conf
# -> /etc/void-wg/vpn.vwg  (с параметрами Jc=4 Jmin=40 Jmax=70)

# Статус + параметры обфускации
void-wg status myvpn

# Сгенерировать шаблон .vwg
void-wg genconf > /etc/void-wg/template.vwg
```

## Формат .vwg
```ini
[Interface]
PrivateKey = <key>
Address = 10.0.0.2/32
DNS = 1.1.1.1

# VoidWG обфускация (AmneziaWG совместимо)
Obfuscation = on
Jc   = 4      # кол-во junk пакетов перед handshake
Jmin = 40     # мин. размер junk пакета (байт)
Jmax = 70     # макс. размер junk пакета (байт)
S1   = 0      # padding init handshake
S2   = 0      # padding response handshake
H1   = 1      # XOR magic init
H2   = 2      # XOR magic response
H3   = 3      # XOR magic cookie
H4   = 4      # XOR magic transport

[Peer]
PublicKey = <key>
Endpoint = server:51821
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```
