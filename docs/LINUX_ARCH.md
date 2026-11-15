# Сборка и установка для Arch Linux / Arch Linux ARM

## ARM64 (aarch64)

Для Arch Linux ARM 64-bit из `next/`:

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform linux/arm64
```

Результат: `next/build/release/linux-arm64/`. Архив для переноса `plainmark-0.2.0-arch-aarch64.tar.gz` содержит `PKGBUILD`, Linux arm64-архив и две иконки. На целевой системе:

```sh
sudo pacman -Syu --needed base-devel
tar -xzf plainmark-0.2.0-arch-aarch64.tar.gz
cd plainmark-arch-aarch64
makepkg -si
plainmark
```

Это сборка для `uname -m` = `aarch64`, не для 32-битного ARM. Установка через makepkg и запуск GUI на ARM Linux требуют проверки на целевом устройстве.

## x86_64

MyGo v0.7.4 собирает Linux-бинарник на macOS без C toolchain. Из `next/`:

```sh
go run github.com/egoist/mygo/cmd/mygo@v0.7.4 build -platform linux/amd64
```

Результат: `next/build/release/linux-amd64/`. Linux-конфигурация отключает зависимость от WebKitGTK, поскольку приложение использует native UI, и добавляет команду `plainmark`. Зависимость пакета Arch — `gtk3`.

Для переноса на Arch нужны четыре файла из этой папки: `PKGBUILD`, `plainmark-0.2.0-linux-amd64.tar.gz`, `plainmark-256x256.png`, `plainmark-512x512.png`. Они должны лежать рядом. Чтобы повторить команды ниже, упаковать эти четыре файла в `plainmark-0.2.0-arch-x86_64.tar.gz` внутри каталога `plainmark-arch-x86_64/`. Готовый архив текущей поставки подготовлен для ARM64 (см. выше).

На Arch:

```sh
sudo pacman -Syu --needed base-devel
tar -xzf plainmark-0.2.0-arch-x86_64.tar.gz
cd plainmark-arch-x86_64
makepkg -si
plainmark
```

`makepkg` запускается обычным пользователем. Он проверит SHA-256 исходных файлов, создаст `.pkg.tar.zst`, установит зависимости и пакет через pacman. Приложение также доступно в меню рабочего стола.

Проверено на macOS: кросс-компиляция, форматы ELF x86_64 и ARM aarch64, синтаксис ARM PKGBUILD, контрольные суммы его исходников. `makepkg`, установка и GUI на Arch пока не проверены. Проверить в целевой системе: меню и сочетания Ctrl, диалоги, clipboard, корзину, системную тему, сохранение состояния окна; отдельно Wayland/X11 при использовании обоих.
