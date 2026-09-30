# FAQ — сценарии и опции

Все примеры ниже — настоящие: YAML загружается как есть, а фрагменты
`Caddyfile`/`compose.yaml` — дословный вывод `generate`
(проверен `caddy validate`). Полная схема: [docs/schema.md](docs/schema.md).

## 1. Простой HTML-сервер (+ www)

`sites/site.yaml`:

```yaml
version: 1
domains: [example.com, www.example.com]
preset: static
tls: auto
log: true
root: /var/www/html
```

Фрагмент `generated/Caddyfile`:

```caddyfile
example.com, www.example.com {
	root * /var/www/html
	file_server
	log
}
```

Несколько доменов в одном блоке — так и задумано: один сайт, один блок.
Файлы сайта на хосте лежат в `./data/site/`.

## 2. SPA (Vite / React / Vue сборка)

`sites/app.yaml`:

```yaml
version: 1
domains: [app.example.com]
preset: spa
tls: auto
log: true
root: /var/www/app
```

Фрагмент `generated/Caddyfile`:

```caddyfile
app.example.com {
	root * /var/www/app
	try_files {path} {path}/ /index.html
	file_server
	log
}
```

`try_files ... /index.html` — это fallback для клиентского роутинга:
любой несуществующий путь отдаёт `index.html`.

## 3. Node.js API за обратным прокси

`sites/node.yaml`:

```yaml
version: 1
domains: [api.example.com]
preset: proxy
tls: auto
log: true
backend:
  service: api
  image: node:20-alpine
  internalPort: 3000
  env:
    NODE_ENV: production
```

Фрагмент `generated/Caddyfile`:

```caddyfile
api.example.com {
	reverse_proxy api:3000
	log
}
```

Фрагмент `generated/compose.yaml`:

```yaml
  api:
    image: node:20-alpine
    environment:
      NODE_ENV: production
    restart: unless-stopped
```

Поля `upstream` нет — адрес `api:3000` собран автоматически как
`<service>:<internalPort>`. Код бекенда должен быть запечён в образе:
proxy-сервисы не монтируют вольюмы.

## 4. Python-бекенд (имя сервиса по умолчанию)

`sites/python.yaml`:

```yaml
version: 1
domains: [py.example.com]
preset: proxy
tls: auto
log: false
backend:
  image: python:3.12-slim
  internalPort: 8000
  env:
    PYTHONUNBUFFERED: "1"
```

Без поля `service` имя берётся из id файла — upstream получится
`python:8000`:

```caddyfile
py.example.com {
	reverse_proxy python:8000
}
```

`log: false` просто убирает директиву `log` из блока.

## 5. PHP (PHP-FPM)

`sites/blog.yaml`:

```yaml
version: 1
domains: [blog.example.com]
preset: php
tls: auto
log: true
root: /var/www/blog
backend:
  image: php:8.3-fpm
```

Фрагмент `generated/Caddyfile`:

```caddyfile
blog.example.com {
	root * /var/www/blog
	php_fastcgi blog:9000
	file_server
	log
}
```

FastCGI-адрес всегда `<service>:9000`, поле `internalPort` для `php`
запрещено. Код из `./data/blog/` монтируется сразу в два сервиса:
в `caddy` (только чтение, отдавать статику) и в php (исполнять).

## 6. Локальный домен без TLS

`sites/local.yaml`:

```yaml
version: 1
domains: [myapp.test]
preset: static
tls: off
log: false
root: /var/www/local
```

```caddyfile
http://myapp.test {
	root * /var/www/local
	file_server
}
```

`tls: off` превращается в префикс `http://` — так в Caddy v2 штатно
отключается автоматический HTTPS для сайта.

## Опции

| Поле | Где | Значения / пример |
|---|---|---|
| `domains` | каждый сайт | 1+ валидных хостнеймов, блок сортируется |
| `preset` | каждый сайт | `static` / `spa` / `proxy` / `php` |
| `tls` | каждый сайт | `auto` (Let's Encrypt) / `off` (`http://`) |
| `log` | каждый сайт | `true` → директива `log`, `false` → без неё |
| `root` | `static`, `spa`, `php` | путь **внутри** контейнера, абсолютный |
| `backend.service` | `proxy`, `php` | имя сервиса, по умолчанию = id файла |
| `backend.image` | `proxy`, `php` | готовый образ, напр. `node:20-alpine` |
| `backend.internalPort` | только `proxy` | 1–65535, upstream = `service:port` |
| `backend.env` | `proxy`, `php` | `KEY: value` → `environment` в compose |
| `global.email` | `caddy.project.yaml` | email для Let's Encrypt, попадает в `{...}` |

## Вопросы

**Где поле upstream?** Его нет специально: для `proxy` адрес всегда
`<service>:<internalPort>`, для `php` — `<service>:9000`. Одно место
правды вместо двух, которые могут рассинхронизироваться.

**Куда положить файлы сайта?** В `./data/<id>/` рядом с проектом
(`<id>` — имя yaml-файла). В контейнер они попадут по пути `root`.

**Можно ли править `generated/` руками?** Нет — при следующем `generate`
файлы перезапишутся. Правьте `sites/*.yaml` (в UI или в редакторе).

**Обязателен ли бинарник caddy?** Нет. Без него генерация работает,
но пропускает `caddy fmt` и проверку `caddy validate` (с пометкой).
С ним — форматирует вывод и валидирует результат.

**Как поднять стек?** Из корня проекта:

```sh
docker compose --project-directory . -f generated/compose.yaml up
```

`--project-directory .` нужен, чтобы относительные пути вольюмов
(`./data/...`) резолвились от корня, а не от `generated/`.

**UI показывает «Changed on disk».** Файл поменяли мимо интерфейса
(редактор, `git pull`). UI ничего не перезаписывает: нажмите Reload,
чтобы подхватить новое состояние.

**А редирект www → apex / basic auth / свои серты?** Пока нет — это вне
MVP (см. «Ограничения» в README). Мультидоменность закрывается списком
`domains` в одном блоке.

**А собрать бекенд из своего Dockerfile?** Пока нет: бекенды — только
готовые образы (`image` + `env`). `build`-контексты зарезервированы
под следующую версию схемы.

**Как вести второй проект?** Запустить второй `serve` с другой папкой
(и другим `--port`). Один процесс — один проект, таково P1-соглашение.

**Как применить изменения?** Поменяли `sites/*.yaml` → `Preview`
(проверка без записи) → `Generate` (атомарная запись обоих файлов) →
`docker compose up -d` / коммит в git.
