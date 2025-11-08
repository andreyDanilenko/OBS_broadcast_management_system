Cartoon Scheduler
=================

Этот проект автоматизирует трансляцию мультфильмов через OBS Studio. Он сканирует локальную медиатеку, собирает расписание показов и управляет источником `VideoSource` в OBS, выбирая подходящий эпизод согласно расписанию. В комплект входит три консольных утилиты:

- `cartoon-scheduler` — основной сервис с веб-интерфейсом и интеграцией с OBS.
- `cartoon-scanner` — вспомогательная программа из папки `tools/`, формирует `cartoons.json` с описанием мультфильмов.
- `parse_schedule` — интерактивный генератор расписаний из папки `generate/`, создает `schedule_*.json`.

---

Требования
----------

- Go **1.22+** (в `go.mod` указана версия модуля `go 1.25.3`; используйте последнюю стабильную Go, например 1.22 или новее).
- OBS Studio 29+ c включенным obs-websocket (порт **4455**) и паролем `123456` либо обновите константу `obsPassword` в `main.go`.
- `ffprobe` (часть FFmpeg) должен быть доступен в `PATH` или находиться рядом с исполняемыми файлами.
- Локальная папка `cartoons/` с мультфильмами, структурированная по подпапкам (`cartoons/<cartoon_id>/<video-file>`).

---

Структура проекта
-----------------

- `main.go` — веб-сервис с API /start, /scan и /stop, управляет OBS.
- `tools/generate_meta.go` — исходник утилиты `cartoon-scanner`, извлекает метаданные и строит `cartoons.json`.
- `generate/main.go` — исходник утилиты `parse_schedule`, интерактивно формирует `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json`.
- `cartoons/` — медиатека мультфильмов.
- `cartoons.json`, `schedule_*.json`, `state.json` — данные, которыми обмениваются программы.
- `go.mod`, `go.sum` — зависимые пакеты (`goobs` для OBS и `mpb` для прогресс-баров).

---

Установка Go и зависимостей
---------------------------

1. **Установите Go**
   - macOS: `brew install go`
   - Windows: скачайте MSI-инсталлятор с https://go.dev/dl/

2. **Проверьте окружение**
   - Выполните `go version`, чтобы убедиться, что Go установлен.
   - При необходимости обновите переменные `PATH` и `GOPATH`.

3. **Загрузите зависимости проекта**
   ```sh
   cd /Users/andrei/Documents/develop
   go mod tidy
   ```
   Команда подтянет пакеты из `go.mod` и обновит `go.sum`.

---

Правила работы с рабочей папкой
-------------------------------

Для корректной работы **все исполняемые файлы и данные должны лежать в одной директории** (например, `C:\cartoon-scheduler\` или `/Users/andrei/Documents/develop`). Минимальный набор:

- `cartoon-scheduler.exe` (Windows) или `cartoon-scheduler` (macOS/Linux)
- `cartoon-scanner.exe` / `cartoon-scanner`
- `schedule_parse.exe` / `parse_schedule`
- `ffprobe.exe` (Windows) либо установленные `ffprobe`/`ffmpeg` в `PATH`
- `cartoons/` (папка с мультфильмами; структура `cartoons/<cartoon_id>/<файлы>` обязательна)
- `cartoons.json` (создаётся `cartoon-scanner`)
- `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json` (создаёт `parse_schedule`)
- `state.json` (создаётся автоматически при запуске основного сервиса)

---

Сборка исполняемых файлов
-------------------------

Перед сборкой перейдите в корень проекта:
```sh
cd /Users/andrei/Documents/develop
```

### 1. Основной сервис (`cartoon-scheduler`)
- **macOS / Linux**
  ```sh
  go build -o cartoon-scheduler .
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o cartoon-scheduler.exe .
  ```

### 2. Сканер медиатеки (`cartoon-scanner`)
- **macOS / Linux**
  ```sh
  go build -o cartoon-scanner ./tools
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o cartoon-scanner.exe ./tools
  ```

### 3. Генератор расписаний (`parse_schedule`)
- **macOS / Linux**
  ```sh
  go build -o parse_schedule ./generate
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o schedule_parse.exe ./generate
  ```

> Подсказка: команды с `GOOS`/`GOARCH` можно выполнять с macOS, чтобы собрать Windows-версии (`.exe`) без отдельной ВМ.

---

Документация по утилитам
------------------------

### cartoon-scanner (`tools/main.go`)
1. Запустите `cartoon-scanner` из директории, где находится папка `cartoons/` с подпапками мультфильмов.
2. Программа сканирует каждую подпапку (`cartoons/<cartoon_id>/…`), измеряет длительность файлов через `ffprobe` и собирает список эпизодов.
3. По завершении формирует `cartoons.json` рядом с исполняемым файлом и выводит статистику найденных мультфильмов.
4. Если `cartoons/` отсутствует или пуста, утилита завершит работу с подсказкой, что нужно добавить структуры папок и видео.

### parse_schedule (`generate/main.go`)
1. Убедитесь, что рядом с исполняемым файлом лежит актуальный `cartoons.json`.
2. Запустите `parse_schedule` (на Windows — `schedule_parse.exe`). При старте программа покажет список доступных `cartoon_id`.
3. В интерактивном режиме вводите пары `cartoon_id количество_серий`. Команда `exit` завершает работу.
4. Скрипт последовательно заполняет расписания утро (06:00–12:00), день (12:00–18:00) и вечер (18:00–00:00). Продолжайте вводить мультфильмы и количество серий, пока не заполните весь день; программа сообщит, когда достигнут конец доступных слотов.
5. По завершении автоматически создаются `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json` рядом с бинарником.

### cartoon-scheduler (`main.go`)
1. Подготовьте OBS Studio:
   - включите obs-websocket server (порт 4455 по умолчанию);
   - установите пароль `123456` или измените значение `obsPassword` в исходниках;
   - создайте источник медиа `VideoSource` в нужной сцене.
2. Убедитесь, что в рабочей папке лежат:
   - бинарник `cartoon-scheduler` (`.exe` для Windows);
   - `cartoon-scanner` (используется кнопкой «Сканировать мультфильмы»);
   - `cartoons.json`, `schedule_*.json`, `state.json` (создаётся автоматически при первом запуске);
   - папка `cartoons/` с контентом.
3. Запустите `cartoon-scheduler`. В консоли отобразится ссылка `http://localhost:8080`.
4. Откройте веб-интерфейс: кнопка «Запустить расписание» стартует трансляцию по текущему периоду, «Сканировать мультфильмы» запускает `cartoon-scanner`, «Остановить» завершает цикл.
5. Программа обновляет `state.json` после каждого эпизода, чтобы помнить, что было показано последний раз.

---

Рабочий цикл
------------

1. **Подготовка медиатеки**  
   Расположите мультфильмы в `cartoons/<cartoon_id>/episode.ext`. Идентификатор папки (`cartoon_id`) используется в расписании.

2. **Генерация списка мультфильмов**  
   Выполните `./cartoon-scanner` → получите `cartoons.json`.

3. **Составление расписания**  
   Запустите `./parse_schedule` → создаются `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json`.

4. **Запуск трансляции**  
   Запустите `./cartoon-scheduler` и управляйте показом через http://localhost:8080.

---

Полезные команды
----------------

- Очистить кэш модулей: `go clean -modcache`
- Проверить зависимости: `go list -m all`
- Обновить модуль OBS: `go get github.com/andreykaipov/goobs@latest`

---

Обратная связь
--------------

Если при сборке или запуске возникают ошибки (например, `ffprobe` не найден или OBS не отвечает), проверяйте сообщения в консоли — программы печатают понятные подсказки о дальнейших действиях.

