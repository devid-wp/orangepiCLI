# План разработки OrangeCTL

## 1. Идея проекта

Создать универсальную CLI-программу для Orange Pi под названием:

```bash
orangectl
```

CLI не должен содержать заранее добавленных ботов, проектов или скриптов.

Вместо этого программа предоставляет десять пустых слотов:

```text
slot1
slot2
slot3
slot4
slot5
slot6
slot7
slot8
slot9
slot10
```

Для каждого слота существует отдельный конфигурационный файл.

Пользователь самостоятельно указывает в конфиге:

- отображаемое название;
- рабочую папку;
- команду запуска;
- команду остановки;
- команду перезапуска;
- путь к логам;
- необходимость запуска через `sudo`;
- включён ли слот.

Таким образом, исходный код CLI не нужно менять при добавлении нового скрипта.

---

# 2. Основные команды

CLI должен поддерживать команды:

```bash
orangectl list
orangectl start slot1
orangectl stop slot1
orangectl restart slot1
orangectl status slot1
orangectl logs slot1
```

Статус всех слотов:

```bash
orangectl status
```

Запуск интерактивного меню:

```bash
orangectl
```

или:

```bash
orangectl menu
```

---

# 3. Структура проекта

```text
~/orangectl/
├── orangectl/
│   ├── __init__.py
│   ├── main.py
│   ├── cli.py
│   ├── config_loader.py
│   ├── slot_manager.py
│   ├── process_manager.py
│   ├── log_viewer.py
│   ├── system_info.py
│   ├── validators.py
│   └── ui.py
│
├── configs/
│   ├── slot1.json
│   ├── slot2.json
│   ├── slot3.json
│   ├── slot4.json
│   ├── slot5.json
│   ├── slot6.json
│   ├── slot7.json
│   ├── slot8.json
│   ├── slot9.json
│   └── slot10.json
│
├── state/
│   ├── pids/
│   └── runtime/
│
├── logs/
├── pyproject.toml
├── requirements.txt
└── README.md
```

---

# 4. Десять конфигурационных заготовок

Создать десять одинаковых конфигурационных файлов.

Пример файла:

```text
configs/slot1.json
```

Содержимое:

```json
{
  "slot": "slot1",
  "enabled": false,
  "display_name": "Empty Slot 1",
  "description": "",
  "working_directory": "",
  "start_command": "",
  "stop_command": "",
  "restart_command": "",
  "log_file": "",
  "use_sudo": false,
  "auto_restart": false,
  "environment": {}
}
```

Для второго файла:

```json
{
  "slot": "slot2",
  "enabled": false,
  "display_name": "Empty Slot 2",
  "description": "",
  "working_directory": "",
  "start_command": "",
  "stop_command": "",
  "restart_command": "",
  "log_file": "",
  "use_sudo": false,
  "auto_restart": false,
  "environment": {}
}
```

По такому же принципу создать файлы до:

```text
slot10.json
```

Отличаться должно только значение:

```json
"slot": "slot10"
```

и:

```json
"display_name": "Empty Slot 10"
```

---

# 5. Как пользователь заполняет слот

Допустим, пользователь хочет привязать собственный Python-скрипт к `slot1`.

Он открывает:

```bash
nano ~/orangectl/configs/slot1.json
```

И самостоятельно заполняет конфиг:

```json
{
  "slot": "slot1",
  "enabled": true,
  "display_name": "Мой скрипт",
  "description": "Описание моего скрипта",
  "working_directory": "/home/USER/projects/my-script",
  "start_command": "./venv/bin/python main.py",
  "stop_command": "",
  "restart_command": "",
  "log_file": "/home/USER/projects/my-script/runtime.log",
  "use_sudo": false,
  "auto_restart": false,
  "environment": {}
}
```

После этого доступны команды:

```bash
orangectl start slot1
orangectl stop slot1
orangectl restart slot1
orangectl status slot1
orangectl logs slot1
```

В интерфейсе должно отображаться не только имя `slot1`, но и пользовательское название:

```text
slot1    Мой скрипт    stopped
```

---

# 6. Привязка конфигов к именам в коде

В коде должен существовать фиксированный белый список:

```python
ALLOWED_SLOTS = [
    "slot1",
    "slot2",
    "slot3",
    "slot4",
    "slot5",
    "slot6",
    "slot7",
    "slot8",
    "slot9",
    "slot10",
]
```

Когда пользователь вводит:

```bash
orangectl start slot4
```

CLI должен открыть:

```text
configs/slot4.json
```

То есть имя команды напрямую соответствует имени конфигурационного файла:

```text
slot1 → slot1.json
slot2 → slot2.json
slot3 → slot3.json
...
slot10 → slot10.json
```

Другие произвольные имена принимать нельзя.

Например, команда:

```bash
orangectl start something
```

должна вернуть ошибку:

```text
Ошибка: слот something не существует.

Доступные слоты:
slot1, slot2, slot3, slot4, slot5,
slot6, slot7, slot8, slot9, slot10
```

---

# 7. Проверка конфигурации

Перед запуском CLI должен проверить конфиг.

Проверки:

- существует ли конфигурационный файл;
- является ли JSON корректным;
- совпадает ли поле `slot` с именем файла;
- включён ли слот через `enabled`;
- заполнено ли поле `working_directory`;
- существует ли рабочая папка;
- заполнена ли команда запуска;
- не содержит ли путь запрещённые конструкции;
- существует ли файл логов, если он указан.

Команда проверки:

```bash
orangectl validate slot1
```

Проверка всех конфигов:

```bash
orangectl validate
```

Пример результата:

```text
slot1    OK
slot2    disabled
slot3    missing start_command
slot4    invalid working_directory
slot5    disabled
```

---

# 8. Команда списка

Команда:

```bash
orangectl list
```

Должна показать все десять слотов:

```text
SLOT      NAME             ENABLED    STATUS
slot1     Мой скрипт       yes        stopped
slot2     Empty Slot 2     no         disabled
slot3     Empty Slot 3     no         disabled
slot4     Empty Slot 4     no         disabled
slot5     Empty Slot 5     no         disabled
slot6     Empty Slot 6     no         disabled
slot7     Empty Slot 7     no         disabled
slot8     Empty Slot 8     no         disabled
slot9     Empty Slot 9     no         disabled
slot10    Empty Slot 10    no         disabled
```

Выключенные слоты нельзя запускать.

---

# 9. Запуск слота

Команда:

```bash
orangectl start slot1
```

Алгоритм:

1. Проверить, входит ли `slot1` в белый список.
2. Загрузить `configs/slot1.json`.
3. Проверить JSON.
4. Проверить `enabled`.
5. Проверить рабочую папку.
6. Проверить команду запуска.
7. Проверить, не запущен ли процесс.
8. Перейти в рабочую папку.
9. Добавить переменные окружения.
10. Выполнить команду.
11. Перенаправить вывод в лог.
12. Сохранить PID.
13. Показать результат пользователю.

Пример:

```text
✓ Слот slot1 запущен
Название: Мой скрипт
PID: 1842
```

Если процесс уже работает:

```text
Слот slot1 уже запущен.
PID: 1842
```

---

# 10. Остановка слота

Команда:

```bash
orangectl stop slot1
```

Если в конфиге заполнено:

```json
"stop_command": "..."
```

CLI выполняет пользовательскую команду остановки.

Если `stop_command` пустой, CLI использует PID сохранённого процесса:

1. Отправляет `SIGTERM`.
2. Ждёт завершения.
3. Проверяет состояние процесса.
4. При необходимости сообщает, что процесс не завершился.
5. Удаляет PID-файл после успешной остановки.

Нельзя завершать процесс, не проверив, что PID принадлежит нужному слоту.

---

# 11. Перезапуск слота

Команда:

```bash
orangectl restart slot1
```

Если в конфиге заполнено:

```json
"restart_command": "..."
```

CLI использует эту команду.

Если поле пустое, выполняется стандартная последовательность:

```text
stop → проверка остановки → start
```

---

# 12. Статус процессов

Команда одного слота:

```bash
orangectl status slot1
```

Пример:

```text
Slot: slot1
Name: Мой скрипт
Enabled: yes
Status: running
PID: 1842
Working directory: /home/USER/projects/my-script
```

Команда всех слотов:

```bash
orangectl status
```

Пример:

```text
slot1     running
slot2     disabled
slot3     stopped
slot4     config error
slot5     disabled
```

---

# 13. PID-файлы

PID-файлы хранить отдельно:

```text
~/.local/state/orangectl/pids/
├── slot1.pid
├── slot2.pid
├── slot3.pid
└── ...
```

CLI не должен слепо доверять PID-файлу.

Необходимо проверять:

- существует ли процесс;
- совпадает ли его команда запуска;
- относится ли процесс к нужной рабочей папке;
- не был ли PID переиспользован другим процессом.

Если PID-файл устарел, его нужно удалить.

---

# 14. Просмотр логов

Команда:

```bash
orangectl logs slot1
```

По умолчанию показывает последние 50 строк.

Дополнительные варианты:

```bash
orangectl logs slot1 --lines 100
orangectl logs slot1 --follow
```

Если поле `log_file` пустое:

```text
Для slot1 не настроен файл логов.
Укажите log_file в configs/slot1.json.
```

CLI не должен автоматически читать случайные файлы вне пути, указанного в конфиге.

---

# 15. Редактирование конфигов через CLI

Добавить удобную команду:

```bash
orangectl edit slot1
```

Она открывает соответствующий конфиг в редакторе.

Редактор определяется по переменной:

```bash
$EDITOR
```

Если переменная не задана, использовать:

```bash
nano
```

Команда:

```bash
orangectl edit slot3
```

открывает:

```text
configs/slot3.json
```

После закрытия редактора CLI должен предложить или автоматически выполнить проверку конфига.

---

# 16. Сброс слота

Добавить команду:

```bash
orangectl reset slot1
```

Она возвращает конфиг к пустой заготовке:

```json
{
  "slot": "slot1",
  "enabled": false,
  "display_name": "Empty Slot 1",
  "description": "",
  "working_directory": "",
  "start_command": "",
  "stop_command": "",
  "restart_command": "",
  "log_file": "",
  "use_sudo": false,
  "auto_restart": false,
  "environment": {}
}
```

Перед сбросом обязательно запросить подтверждение:

```text
Конфигурация slot1 будет полностью очищена.
Продолжить? [y/N]
```

Перед очисткой желательно создать резервную копию:

```text
configs/backups/slot1-2026-07-31-212200.json
```

---

# 17. Переменные окружения

Поле:

```json
"environment": {}
```

позволяет указать переменные окружения:

```json
"environment": {
  "APP_MODE": "production",
  "PORT": "8080"
}
```

CLI передаёт их процессу при запуске.

Секретные данные и токены лучше не хранить прямо в JSON.

Для секретов можно добавить поле:

```json
"env_file": "/home/USER/projects/my-script/.env"
```

Но CLI не должен показывать содержимое `.env` в терминале или логах.

---

# 18. Использование sudo

По умолчанию:

```json
"use_sudo": false
```

Если пользователь устанавливает:

```json
"use_sudo": true
```

CLI может запускать команду через `sudo`.

Но перед этим он должен вывести предупреждение:

```text
Слот slot1 настроен для запуска с правами sudo.
```

Сам CLI не должен всегда запускаться от root.

---

# 19. Интерактивное меню

Команда:

```bash
orangectl
```

открывает меню:

```text
╭─────────────────────────────╮
│         ORANGECTL           │
├─────────────────────────────┤
│ 1. Список слотов            │
│ 2. Запустить слот           │
│ 3. Остановить слот          │
│ 4. Перезапустить слот       │
│ 5. Статус                   │
│ 6. Логи                     │
│ 7. Редактировать конфиг     │
│ 8. Проверить конфиги        │
│ 9. Информация о системе     │
│ 0. Выход                    │
╰─────────────────────────────╯
```

При выборе запуска показывать только включённые слоты.

При редактировании показывать все десять слотов.

---

# 20. Системная информация

Команда:

```bash
orangectl system
```

Показывает:

```text
Hostname
Uptime
CPU usage
RAM usage
Disk usage
Temperature
Local IP
Operating system
Kernel
```

Эта функция не должна быть связана с конкретными пользовательскими скриптами.

---

# 21. Зависимости

Использовать:

```text
typer
rich
psutil
```

Файл:

```text
requirements.txt
```

```txt
typer
rich
psutil
```

Для запуска процессов использовать стандартные библиотеки Python:

```python
subprocess
pathlib
json
os
signal
shlex
```

---

# 22. Глобальная команда

В `pyproject.toml` добавить:

```toml
[project.scripts]
orangectl = "orangectl.main:app"
```

Установка:

```bash
cd ~/orangectl
python3 -m venv venv
source venv/bin/activate
pip install -e .
```

Для глобального использования без ручной активации окружения можно установить через `pipx`:

```bash
pipx install -e ~/orangectl
```

После этого команда доступна из любой папки:

```bash
orangectl
```

---

# 23. Что запрещено добавлять

В проект нельзя заранее добавлять:

- Telegram-ботов;
- Discord-ботов;
- загрузчики;
- парсеры;
- серверы;
- пользовательские проекты;
- конкретные пути к существующим скриптам;
- токены;
- API-ключи;
- `.env` файлы;
- реальные команды пользователя.

Все десять слотов должны быть пустыми.

CLI предоставляет только универсальную систему управления, а пользователь самостоятельно заполняет конфигурации.

---

# 24. Этапы разработки

## Этап 1 — каркас CLI

Реализовать:

```bash
orangectl list
orangectl validate
orangectl validate slot1
```

Создать десять пустых JSON-конфигов.

## Этап 2 — управление процессами

Добавить:

```bash
orangectl start slot1
orangectl stop slot1
orangectl restart slot1
orangectl status slot1
```

## Этап 3 — PID и состояние

Добавить безопасное хранение PID и проверку процессов.

## Этап 4 — логи

Добавить:

```bash
orangectl logs slot1
orangectl logs slot1 --follow
```

## Этап 5 — управление конфигами

Добавить:

```bash
orangectl edit slot1
orangectl reset slot1
```

## Этап 6 — меню

Добавить интерактивный интерфейс через Rich.

## Этап 7 — системная информация

Добавить:

```bash
orangectl system
```

## Этап 8 — установка

Оформить проект через `pyproject.toml` и установить команду через `pipx`.

---

# 25. Ожидаемый результат

После установки пользователь получает десять пустых слотов:

```bash
orangectl list
```

```text
slot1     Empty Slot 1     disabled
slot2     Empty Slot 2     disabled
slot3     Empty Slot 3     disabled
slot4     Empty Slot 4     disabled
slot5     Empty Slot 5     disabled
slot6     Empty Slot 6     disabled
slot7     Empty Slot 7     disabled
slot8     Empty Slot 8     disabled
slot9     Empty Slot 9     disabled
slot10    Empty Slot 10    disabled
```

Пользователь открывает нужный конфиг:

```bash
orangectl edit slot1
```

Заполняет собственные параметры и включает слот:

```json
"enabled": true
```

После этого управляет своим скриптом:

```bash
orangectl start slot1
orangectl status slot1
orangectl logs slot1
orangectl restart slot1
orangectl stop slot1
```

Главный принцип проекта:

> CLI ничего не знает о конкретных программах пользователя. Он работает только с десятью универсальными слотами и их конфигурационными файлами.