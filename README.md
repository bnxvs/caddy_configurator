# Caddy Configurator

Локальный менеджер конфигураций Caddy для типовых сценариев: статический
HTML-сервер, SPA, обратный прокси (Node.js / Python / Go), PHP-FPM.
Сайты описываются как данные в git-friendly YAML-файлах, а готовые
`Caddyfile` и `compose.yaml` (включая бекенды) детерминированно
генерируются — вручную синтаксис Caddy знать не нужно.

Один бинарник: CLI (`init / generate / validate / serve`) + веб-интерфейс
на `http://127.0.0.1:8080` для редактирования через формы с живым preview.

## Возможности (MVP)

- Пресеты сайтов: `static`, `spa`, `proxy`, `php` + общие поля
  `domains[]`, `tls: auto/off`, `log`.
- Бекенды — только готовые Docker-образы (`image` + `env`), без сборки.
- Генерация `generated/Caddyfile` и `generated/compose.yaml` из одних данных.
- Preview без записи на диск; `generate` пишет атомарно.
- `caddy fmt` / `caddy validate` подключаются автоматически, если бинарник
  `caddy` есть в PATH (иначе — пропускаются с пометкой).
- Файлы можно править и в редакторе: UI покажет баннер «Changed on disk»,
  ничего молча не перезапишет.

Подробно про схему: [docs/schema.md](docs/schema.md).
Примеры по сценариям и описание опций: [FAQ.md](FAQ.md).

## Требования

- Для запуска готового бинарника — ничего, кроме ОС (Linux/macOS).
- Для сборки: Go 1.26+, Node.js 18+ (только чтобы один раз собрать фронт).
- Опционально: `caddy` в PATH (форматирование и проверка),
  Docker (чтобы поднимать сгенерированный compose).

## Установка

Собрать из исходников:

```sh
npm --prefix web run build
go build -o caddy-configurator ./cmd/caddy-configurator
```

## Быстрый старт

```sh
# 1. Новый проект
caddy-configurator init ./my-caddy

# 2. Веб-интерфейс: открыть http://127.0.0.1:8080,
#    добавить сайты через формы, посмотреть Preview
caddy-configurator serve ./my-caddy

# 3. Записать артефакты на диск (то же самое делает кнопка Generate в UI)
caddy-configurator generate ./my-caddy

# 4. Проверить и поднять стек
caddy-configurator validate ./my-caddy
docker compose --project-directory ./my-caddy -f ./my-caddy/generated/compose.yaml up
```

Все команды CLI: [docs/cli.md](docs/cli.md).

## Структура проекта

```text
my-caddy/
  caddy.project.yaml      # имя проекта, global.email для Let's Encrypt
  sites/<id>.yaml         # один сайт = один файл (исходники, в git)
  generated/Caddyfile     # артефакт (в git, руками не правим)
  generated/compose.yaml  # артефакт (в git, руками не правим)
  data/<id>/              # файлы сайтов на хосте -> root внутри контейнеров
```

Контракт: `sites/*.yaml` — правда, `generated/*` — чистая функция от них.
Повторная генерация без изменений даёт байт-в-байт тот же результат,
поэтому `git diff` всегда показывает только осмысленные правки.

Соглашения MVP (жёсткие, оверрайды — в следующих версиях схемы):

- proxy: upstream собирается как `<service>:<internalPort>`;
- php: FastCGI всегда `<service>:9000`, `internalPort` запрещён;
- имя сервиса по умолчанию равно id файла, хост-путь данных — `./data/<id>`;
- proxy не монтирует код (он запечён в образе), php делит вольюм между
  `caddy` (только чтение) и php-сервисом.

## Разработка

```sh
npm --prefix web run build   # собрать фронт в web/dist
go build ./...               # собрать всё
go vet ./...                 # проверки
go test ./...                # все тесты
```

Режим разработки UI с hot reload (фронт отдельно, API локально):

```sh
npm --prefix web run dev            # vite на :5173
go run ./cmd/caddy-configurator serve ./my-caddy --dev-frontend http://127.0.0.1:5173
```

## Ограничения MVP

Нет `auth/headers/redir/encode`, нет своих/internal сертов
(только `tls: auto/off`), нет `build`-контекстов и произвольных полей
сервисов, один compose на всех (без dev/prod вариантов), нет git-кнопок
и мультипроекта (один запуск — одна папка), bind только на loopback.
