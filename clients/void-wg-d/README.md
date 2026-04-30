# void-wg-d — Standard WireGuard Client

Лёгкий WireGuard-совместимый клиент с дополнениями.
Полная обратная совместимость со стандартными `.conf` файлами.

## Отличия от wg-quick
| Фича | wg-quick | void-wg-d |
|---|---|---|
| Стандартные .conf файлы | ✓ | ✓ |
| Kill switch | ручной PostUp/Down | `--kill-switch` флаг |
| Bypass subnets | ручной | `--bypass 192.168.1.0/24` |
| Peer статистика | `wg show` | `void-wg-d status <name>` |
| Авто-реконнект | ✗ | roadmap |

## Установка (сборка)
```bash
cd clients/void-wg-d
go build -o void-wg-d ./cmd/void-wg-d
sudo mv void-wg-d /usr/local/bin/
```

## Использование
```bash
# Поднять туннель (читает стандартный .conf)
sudo void-wg-d up myvpn.conf --kill-switch --bypass 192.168.1.0/24

# Статус и статистика пиров
sudo void-wg-d status myvpn

# Опустить туннель
sudo void-wg-d down myvpn

# Импортировать конфиг
void-wg-d import ~/Downloads/vpn.conf
```

## Формат конфига
Стандартный WireGuard .conf — без изменений.
```ini
[Interface]
PrivateKey = ...
Address = 10.0.0.2/32
DNS = 1.1.1.1

[Peer]
PublicKey = ...
Endpoint = server:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```
