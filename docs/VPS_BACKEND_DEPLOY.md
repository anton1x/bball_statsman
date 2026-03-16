# Deploy frontend + backend on VPS (GitHub Actions + Docker Compose)

Ниже инструкция для автодеплоя всего приложения (frontend + backend) на один сервер в `/opt/bball_statsman`.

## 1) Подготовить сервер

Установите Docker + Docker Compose plugin:

```bash
sudo apt update
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo $VERSION_CODENAME) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

Проверьте:

```bash
docker --version
docker compose version
```

## 2) Создать пользователя для деплоя

```bash
sudo adduser --disabled-password --gecos "" deploy
sudo usermod -aG docker deploy
sudo mkdir -p /opt/bball_statsman
sudo chown -R deploy:deploy /opt/bball_statsman
```

## 3) Настроить SSH-ключ для GitHub Actions

На локальной машине создайте ключ без passphrase:

```bash
ssh-keygen -t ed25519 -C "gha-deploy" -f ./gha_deploy_key
```

Добавьте публичный ключ на сервер:

```bash
sudo -u deploy mkdir -p /home/deploy/.ssh
sudo -u deploy chmod 700 /home/deploy/.ssh
cat ./gha_deploy_key.pub | sudo -u deploy tee -a /home/deploy/.ssh/authorized_keys
sudo -u deploy chmod 600 /home/deploy/.ssh/authorized_keys
```

## 4) Добавить GitHub Secrets (Repository → Settings → Secrets and variables → Actions)

Обязательные секреты:

- `VPS_HOST` — IP/домен сервера.
- `VPS_USER` — пользователь, например `deploy`.
- `VPS_SSH_KEY` — приватный ключ из `gha_deploy_key` (целиком, включая BEGIN/END).
- `VPS_APP_DIR` — директория проекта на сервере: `/opt/bball_statsman`.

Опционально:

- `VPS_SSH_PORT` — SSH-порт (если не 22).
- `VPS_SSL_CERT` — TLS сертификат (PEM), например fullchain от GlobalSign.
- `VPS_SSL_KEY` — приватный ключ TLS (PEM).

> Если `VPS_SSL_CERT` и `VPS_SSL_KEY` заполнены, workflow создаст файлы `ssl/tls.crt` и `ssl/tls.key` на VPS, и nginx автоматически включит HTTPS (443) + редирект с HTTP на HTTPS.
> Если SSL-секреты пустые — контейнер запустится только на HTTP (порт 80).

## 5) Открыть порты

Если используете UFW:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

## 6) Первый деплой

1. Запушьте изменения в `main` (или запустите workflow вручную через `workflow_dispatch`).
2. Workflow `Deploy app to VPS (frontend + backend)`:
   - скопирует frontend+backend файлы и `docker-compose.yml` на сервер;
   - (опционально) запишет SSL-сертификаты из секретов в `${VPS_APP_DIR}/ssl`;
   - выполнит `docker compose up -d --build`.

Проверка на сервере:

```bash
cd /opt/bball_statsman
docker compose ps
docker compose logs -f frontend
docker compose logs -f backend
```

Проверить сертификат:

```bash
openssl s_client -connect your-domain:443 -servername your-domain </dev/null 2>/dev/null | openssl x509 -noout -issuer -subject -dates
```

## 7) Как это работает теперь

- Frontend не деплоится в GitHub Pages.
- Frontend собирается в Docker-образ и отдается через `nginx` в контейнере `frontend`.
- Запросы `/api/*` из браузера проксируются контейнером `frontend` в `backend:8080` внутри docker-сети.
- Backend доступен только внутри docker-сети (наружу публикуются только 80 и 443 порты фронтенда).
- HTTPS включается автоматически при наличии файлов `ssl/tls.crt` и `ssl/tls.key`.
