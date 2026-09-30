# Proposal

## Why

Написание Caddyfile вручную для типовых сценариев (статика, SPA, reverse proxy, PHP) требует знания синтаксиса Caddy, легко приводит к ошибкам и плохо масштабируется на несколько сайтов. Нужен локальный менеджер, который хранит сайты как данные в git-friendly файлах и детерминированно генерирует готовые к работе `Caddyfile` и `compose.yaml` (включая бекенды), с веб-интерфейсом для тех, кто не хочет учить синтаксис.

## What Changes

- Новое приложение `caddy-configurator`: один Go-бинарник, localhost UI (React SPA) + CLI, работает с одной папкой проекта за запуск (`serve ./my-caddy`).
- Файловое хранение исходников: `caddy.project.yaml` + `sites/<id>.yaml`, генерация артефактов в `generated/Caddyfile` и `generated/compose.yaml`, хост-данные в `data/<id>/`. Исходники и артефакты коммитятся в git.
- MVP-пресеты сайтов: `static`, `spa`, `proxy` (node/python/go — один пресет), `php` + общие поля `domains[]`, `tls: auto/off`, `log: on/off`. Бекенды только на готовых image (`image` + `internalPort`/`env`), без build-контекстов.
- Жёсткие соглашения MVP без оверрайдов: `upstream = <service>:<internalPort>`, `fastcgi = <service>:9000`, хост-путь `root` всегда `./data/<id>`; proxy без вольюмов, php делит вольюм между caddy (ro) и php (rw). Схема `version: 1`, неизвестные поля — варнинг под будущие расширения.
- Preview без записи на диск, `generate` с atomic write + `caddy fmt` при наличии бинарника, детект ручных правок (`changedOnDisk` баннер, без авто-мерджа).
- Вне скоупа MVP: `auth/headers/redir/encode`, custom/internal серты, build-контексты, `command/volumes/ports` произвольно, dev/prod compose-варианты, git-кнопки, мультипроект, bind на `0.0.0.0`.

## Capabilities

### New Capabilities

- `project-workspace`: раскладка проекта, P1-модель (один root за запуск), версионирование схемы, atomic write, детект изменений на диске, соглашение о коммите артефактов.
- `site-config`: пресеты и общие поля сайтов, image-only бекенды, жёсткие соглашения адресации и вольюмов, валидация схемы и формат ошибок.
- `artifact-generation`: детерминированный рендер `Caddyfile` и `compose.yaml`, preview без записи, generate с записью, интеграция `caddy fmt/validate`.
- `local-ui`: localhost React SPA + Go JSON API (CRUD сайтов, формы по пресетам, preview/snippet, generate, status), безопасность (bind 127.0.0.1, санитизация id).
- `cli`: команды `init/generate/validate/serve`, dev/prod раздача фронта, single-binary дистрибуция.

### Modified Capabilities

- Нет (проект пустой, существующих спек нет).

## Impact

- Новый репозиторий/код: Go-ядро (`config/render/caddy/cli/web`), React-фронт (`web/`, vite, embed.FS), два generated-артефакта как контракт.
- Зависимости: Go + Node (только сборка фронта), опционально бинарник `caddy` в PATH для fmt/validate, Docker только для запуска результата.
- Системы: localhost HTTP API на 127.0.0.1, файловая система (watch/atomic write), git-история как журнал изменений.
