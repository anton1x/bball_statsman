# Backend deploy on VPS (GitHub Actions + Docker Compose)

Ниже минимальная инструкция, чтобы автодеплой бэкенда с GitHub Actions заработал.

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
sudo mkdir -p /opt/bball-statsman
sudo chown -R deploy:deploy /opt/bball-statsman
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
- `VPS_APP_DIR` — директория проекта на сервере, например `/opt/bball-statsman`.

Опционально:

- `VPS_SSH_PORT` — SSH-порт (если не 22).

## 5) Открыть порт приложения

Если используете UFW:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 8080/tcp
sudo ufw enable
sudo ufw status
```

## 6) Первый деплой

1. Запушьте изменения в `main` (или запустите workflow вручную через `workflow_dispatch`).
2. Workflow `Deploy backend to VPS` скопирует `backend/` и `docker-compose.yml` на сервер.
3. На сервере выполнится:

```bash
docker compose up -d --build backend
```

Проверка на сервере:

```bash
cd /opt/bball-statsman
docker compose ps
docker compose logs -f backend
```

## 7) Подключение фронтенда к VPS API

Для GitHub Pages задайте в Secrets/Variables (или в коде) URL вашего API:

- `VITE_API_BASE_URL=https://your-domain-or-ip:8080`

И передавайте это значение в frontend build workflow (если нужно через `env` в шаге `npm run build`).

---

С текущей конфигурацией фронтенд уже деплоится в GitHub Pages через `.github/workflows/deploy-pages.yml`, а новый workflow отвечает только за backend на VPS.
