# Шаблон HTTPS-конфига. install.sh подставляет __SERVER_NAME__.
# Сертификаты лежат в /etc/nginx/tls (bind-mount из ./runtime/tls на хосте).

server {
    listen 80 default_server;
    server_name __SERVER_NAME__;

    # ACME http-01 challenge (на случай webroot-renewals)
    location /.well-known/acme-challenge/ {
        root /var/www/acme;
    }

    # Все остальные запросы — редирект на HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl default_server;
    http2 on;
    server_name __SERVER_NAME__;
    root /usr/share/nginx/html;
    index index.html;

    client_max_body_size 1m;

    ssl_certificate     /etc/nginx/tls/fullchain.pem;
    ssl_certificate_key /etc/nginx/tls/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers         ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;
    add_header Referrer-Policy strict-origin-when-cross-origin always;

    location /api/ {
        proxy_pass http://api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 60s;
    }

    location = /healthz {
        proxy_pass http://api:8080/healthz;
        access_log off;
    }

    # Метрики Prometheus — только из private nets
    location = /metrics {
        allow 127.0.0.1;
        allow 10.0.0.0/8;
        allow 172.16.0.0/12;
        allow 192.168.0.0/16;
        deny all;
        proxy_pass http://api:8080/metrics;
    }

    location / {
        try_files $uri /index.html;
    }
}
