# Plainmark Go

Новая реализация Plainmark. Реализованы workspace, вкладки, файловые операции, watcher и конфликты. Полная функциональная замена исходной версии ещё не готова: расширения просмотра, UI шифрования, настройки и миграция состояния находятся в работе.

Требуется Go 1.27.3 или новее. Node, Rust и cgo для этой реализации не нужны.

```sh
go run ./cmd/plainmark ../testdata/workspace/Math.md
go run ./cmd/plainmark ../testdata/workspace
go test ./...
go vet ./...
go build -o build/plainmark ./cmd/plainmark
```

Проверка настоящего окна (запускает отдельный процесс и работает с временной заметкой):

```sh
go build -tags e2e -o build/plainmark ./cmd/plainmark
python3 scripts/smoke.py
```

Снимки светлого и тёмного просмотра:

```sh
PLAINMARK_SNAPSHOTS=/tmp/plainmark-snapshots go test ./internal/desktop -run Snapshot -count=1
```

Сравнение двух движков формул (артефакты сохраняются в `/tmp/plainmark-mathprobe`):

```sh
go run ./tools/mathprobe
```

Проверка криптографической совместимости с исходным Rust-кодом (требуется Cargo):

```sh
python3 scripts/crypto_compat.py
```

MyGo зафиксирован на v0.7.4, движок формул `go-tex/math` — на v0.50.0. Зависимость Luo-Studio используется только сравнительным инструментом и не импортируется приложением.

Исходная реализация Tauri находится в родительской директории. Функциональный контракт и незавершённые пункты: [план](../docs/MIGRATION_PLAN.md), [приёмка](../docs/ACCEPTANCE.md).
