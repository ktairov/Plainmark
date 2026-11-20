# Plainmark

Настольное приложение для Markdown-заметок на Go + MyGo с нативным интерфейсом. Исходники находятся в `next/`. Node, Rust и WebView для сборки и запуска приложения не нужны. Linux-пакет требует GTK 3, без WebKitGTK.

```sh
cd next
go run ./cmd/plainmark
go test ./...
make build
```

- [Разработка, сборка и проверки](next/README.md)
- [Arch Linux / Arch Linux ARM](docs/LINUX_ARCH.md)
- [План миграции](docs/MIGRATION_PLAN.md)
- [Приёмка](docs/ACCEPTANCE.md)

Оригинальные исходники Tauri/Rust/React удалены из рабочего дерева; контрольная версия сохранена в истории Git, коммит `cb2275f`. Отчёты этапов описывают состояние проекта на момент соответствующих проверок.
