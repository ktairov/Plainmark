# Сборки Linux и Windows

Все платформы используют `next/mygo.json`. Отдельные конфиги для AMD64 и ARM64 не нужны: архитектура задаётся флагом `-platform`. В Linux-секции уже отключён WebKitGTK, поскольку Plainmark использует native UI. Windows-версия также использует native UI и не требует WebView2 Runtime.

Из `next/` на машине с Go 1.27.3+:

```sh
make cross
```

Команда собирает четыре платформы, без установщиков Windows:

| Цель | Отдельная команда | Результат |
| --- | --- | --- |
| Linux AMD64 / x86_64 | `make linux-amd64` | `build/release/linux-amd64/` |
| Linux ARM64 / aarch64 | `make linux-arm64` | `build/release/linux-arm64/` |
| Windows AMD64 / x64 | `make windows-amd64` | `build/release/windows-amd64/` |
| Windows ARM64 | `make windows-arm64` | `build/release/windows-arm64/` |

В Linux-папках MyGo создаёт бинарник, `.desktop`, архив с install.sh, `.deb` и PKGBUILD с иконками для Arch. Установка на Arch описана в [LINUX_ARCH.md](LINUX_ARCH.md).

В Windows-папках находятся `Plainmark Go.exe` и ресурсы приложения. Для переноса portable-сборки можно упаковать всю папку. Иконка exe генерируется MyGo из общего PNG; отдельный ICO-конфиг не требуется.

Без Make (например, из PowerShell) выполняется та же команда:

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform linux/amd64,linux/arm64,windows/amd64,windows/arm64 -skip-installer
```

Для установщиков Windows нужен NSIS. На macOS его компилятор устанавливается командой `brew install makensis`; после установки:

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform windows/amd64,windows/arm64
```

MyGo создаёт установщики `plainmark-go-setup-0.2.0-amd64.exe` и `plainmark-go-setup-0.2.0-arm64.exe`. Без NSIS на macOS он пропускает этот шаг. Подпись Windows и её сертификаты — отдельная настройка для распространяемого выпуска.

Кросс-компиляция не подтверждает работу GUI. Проверять запуск, системные меню, диалоги, шрифты, clipboard, watcher, корзину и восстановление окна нужно на целевых системах. Пока GUI проверен на текущем Mac; Linux и Windows требуют такой проверки.

Проверка от 13 ноября 2026 года: `make cross` успешно собрал все четыре цели на macOS. Команда `file` подтвердила ELF x86_64/aarch64 и PE x86-64/Aarch64. Для переноса подготовлены ZIP-архивы обеих Windows-сборок и комплекты PKGBUILD/архив/иконки для обеих архитектур Arch. Установщики NSIS в этой проверке не создавались.
