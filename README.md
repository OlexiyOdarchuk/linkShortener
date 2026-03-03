# 🔗 LinkShortener Bot

<div align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/PostgreSQL-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/ClickHouse-FFCC01?style=for-the-badge&logo=clickhouse&logoColor=black" alt="ClickHouse" />
  <img src="https://img.shields.io/badge/Docker-2CA5E0?style=for-the-badge&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/Telegram_Bot-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Bot" />
</div>

<br/>

**LinkShortener** — це потужний, високопродуктивний Telegram-бот для скорочення посилань, генерації QR-кодів та збору детальної аналітики переходів. 

Побудований на сучасній архітектурі, проєкт поєднує в собі швидкість кешування **Redis**, надійність реляційної бази **PostgreSQL** та неперевершену потужність **ClickHouse** для обробки аналітичних даних у реальному часі.

---

## ✨ Ключові можливості

- 🤖 **Зручний Telegram Інтерфейс:** Взаємодія з сервісом прямо через месенджер (на базі `telebot.v4`).
- 🔗 **Скорочення посилань:** Миттєва генерація коротких URL.
- 🎯 **Кастомні посилання:** Можливість задавати власні імена для коротких посилань (`/create_custom`).
- 📱 **Генерація QR-кодів:** Автоматичне створення QR-кодів для ваших посилань.
- 📊 **Глибока Аналітика:** 
  - Відстеження кількості переходів.
  - Геолокація користувачів (завдяки інтеграції MaxMind GeoIP2).
  - Аналітика за країнами, містами та платформами.
- ⚡ **Висока Продуктивність:** Кешування запитів за допомогою Redis.

---

## 🛠 Технологічний стек

- **Мова програмування:** Go (Golang) 1.25+
- **Взаємодія з Telegram:** `gopkg.in/telebot.v4`
- **Бази даних:**
  - **PostgreSQL:** Зберігання користувачів, посилань та налаштувань (`sqlx`, `golang-migrate`).
  - **ClickHouse:** Зберігання та агрегація логів переходів для надшвидкої аналітики (`clickhouse-go`).
- **Кешування:** Redis (`go-redis/v9`).
- **Геолокація:** MaxMind GeoIP2 (`geoip2-golang`).
- **Інфраструктура:** Docker, Docker Compose, Taskfile.

---

## 🏗 Архітектура проєкту

Дані про користувачів та їх посилання надійно зберігаються у **PostgreSQL**. Для забезпечення миттєвого редиректу та зменшення навантаження на БД використовується **Redis**-кеш. Кожен перехід за коротким посиланням асинхронно логується у **ClickHouse**, що дозволяє будувати складні аналітичні звіти за мілісекунди навіть при мільйонах записів. Інформація про IP-адресу переходу аналізується за допомогою локальної бази **GeoIP**.

---

## 📋 Вимоги до середовища

Для запуску проєкту вам знадобляться:
1. **Docker** та **Docker Compose**
2. **Task** ([Taskfile](https://taskfile.dev/))
3. **Go 1.25+** (якщо плануєте розробляти локально без Docker)

---

## 🚀 Швидкий старт

### 1. Клонування репозиторію

```bash
git clone https://github.com/OlexiyOdarchuk/linkShortener.git
cd linkShortener
```

### 2. Налаштування змінних оточення

Створіть файл `.env` на основі шаблону:

```bash
cp .env.example .env
```
*Обов'язково вкажіть ваш `TELEGRAM_BOT_TOKEN` та інші необхідні секрети у `.env`.*

### 3. Запуск проєкту

Проєкт містить зручний `Taskfile.yml` для автоматизації рутини. Щоб підняти всю інфраструктуру (PostgreSQL, Redis, ClickHouse та сам додаток):

```bash
task up
```

### 📦 Доступні команди

| Команда | Опис |
| :--- | :--- |
| `task up` | Підняти всі сервіси через Docker Compose (включаючи збірку) |
| `task down` | Зупинити та видалити всі сервіси |
| `task run` | Запустити додаток локально (потребує запущених баз даних) |
| `task build` | Зібрати бінарний файл додатку |
| `task logs` | Переглянути логи Docker Compose в реальному часі |
| `task db-shell` | Відкрити інтерактивну консоль PostgreSQL |

---

## 📱 Використання бота

Знайдіть вашого бота в Telegram та відправте йому `/start`.

**Доступні команди:**
- `/start` — Запустити бота та відкрити головне меню.
- `/create_custom` — Створити посилання з власним ідентифікатором (наприклад, `mysite`).
- `/my_links` — Переглянути список ваших посилань та детальну статистику по кожному з них.
- `/all_analytics` — Отримати загальну розширену статистику всіх ваших переходів.
- `/cancel` — Скасувати поточну дію (наприклад, під час введення кастомного імені).

Просто відправте боту будь-яке довге посилання (наприклад, `https://github.com/OlexiyOdarchuk/linkShortener.git`), і він миттєво поверне вам його коротку версію разом із згенерованим QR-кодом!

---

<div align="center">
  Зроблено з ❤️ на Go
</div>