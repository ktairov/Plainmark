# Plainmark Go

Новая реализация Plainmark. Реализованы workspace, вкладки, файловые операции, watcher, конфликты, расширения просмотра, шифрование, настройки и восстановление собственной сессии. Папки Markdown открываются напрямую.

## Сборка macOS

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -skip-dmg
open "build/release/darwin-arm64/Plainmark Go.app"
```

Для Linux и Windows (AMD64 и ARM64): `make cross` из этой папки. Все цели используют один `mygo.json`; команды отдельных сборок и установщиков приведены в [BUILDING.md](../docs/BUILDING.md).

Готовая локальная сборка 0.2.0 для Apple Silicon: `build/release/darwin-arm64/Plainmark Go.app`. Для её запуска Go не нужен. Используется локальная ad-hoc подпись; нотариализация Apple не выполнялась. Проверки выполнены на текущем Mac, проверка другого компьютера остаётся частью приёмки.

Перенесены Lucide-иконки, компактная панель инструментов, оформление вкладок и дерева. Панель встроена в заголовок macOS; подробнее в [отчёте оформления](../docs/STAGE_6.md).

Настройки открываются через меню приложения → «Настройки…» или Cmd+,. Кнопка Day/Night в тулбаре временно меняет тему текущего окна; выбранный в настройках режим сохраняет приоритет при следующем запуске. В режиме System смена оформления macOS (в том числе автоматическая на восходе/закате) сбрасывает временное переключение и возвращает следование системе. Явно выбранные Day/Night продолжают действовать независимо от системной темы.

Своя конфигурация: `~/Library/Application Support/Plainmark-Go/state.json`. Она содержит настройки и навигацию, без текста заметок и паролей. Для отдельного профиля можно задать `PLAINMARK_CONFIG_DIR`. При первом запуске нужно выбрать папку; последующие запуски восстанавливают её. Импорта состояния Tauri нет.

## Разработка

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
python3 scripts/lifecycle.py
python3 scripts/settings_ui.py
python3 scripts/navigation.py
python3 scripts/menu_ui.py
python3 scripts/restoration.py
python3 scripts/appearance_ui.py
python3 scripts/titlebar_ui.py
```

Снимки светлого и тёмного просмотра:

```sh
PLAINMARK_SNAPSHOTS=/tmp/plainmark-snapshots go test ./internal/desktop -run Snapshot -count=1
```

Сравнение двух движков формул (артефакты сохраняются в `/tmp/plainmark-mathprobe`):

```sh
go run ./tools/mathprobe
```

Проверка криптографической совместимости с исходным Rust-кодом из Git-коммита `cb2275f` (требуются Cargo и история Git):

```sh
python3 scripts/crypto_compat.py
```

MyGo зафиксирован на v0.7.4, движок формул `go-tex/math` — на v0.50.0. Зависимость Luo-Studio используется только сравнительным инструментом и не импортируется приложением.

Исходная реализация Tauri доступна только в истории Git (`cb2275f`). Палитры, переводы, иконки и fixtures новой версии самостоятельны. Linux-пакет не требует WebKitGTK (`linux.webview: false`); инструкция для [Arch Linux ARM](../docs/LINUX_ARCH.md). Функциональный контракт и проверки: [план](../docs/MIGRATION_PLAN.md), [приёмка](../docs/ACCEPTANCE.md).
