# Wellness Center Management System

Backend система для учета клиентов оздоровительного центра с интеграцией Kafka, Redis, Elasticsearch, Sentry, Prometheus, Grafana.

## 🔍 Основной функционал
- Регистрация клиентов с детальной анкетой
- Полнотекстовый поиск (Elasticsearch)
- Асинхронная обработка событий (Kafka Producer/Consumer)
- Кеширование данных (Redis)
- Мониторинг в реальном времени (Prometheus + Grafana)
- Трекинг ошибок (Sentry)
- Healthcheck системы с метриками

## 🛠 Технологический стек
- **Язык**: Go 1.23
- **База данных**: PostgreSQL
- **Брокер сообщений**: Kafka (+ Zookeeper)
- **Поиск**: Elasticsearch
- **Кеш**: Redis
- **Контейнеризация**: Docker + Docker Compose
- **Мониторинг**:	Prometheus + Grafana
- **Ошибки**:	Sentry

## 🚀 Быстрый старт
```bash
git clone https://github.com/evgeniySeleznev/wellness-step-by-step.git
cd wellness-step-by-step
docker-compose up -d
```

# Доступные эндпоинты после запуска:
•	http://localhost:8080/api/v1/health - Healthcheck
•	http://localhost:8080/metrics - Prometheus метрики
•	http://localhost:3000 - Grafana (admin/admin)
•	http://localhost:9090 - Prometheus UI

# ✨ Новые возможности

# 🎯 **Sentry Integration**
•	Мониторинг ошибок в реальном времени
•	Привязка ошибок к пользовательским сессиям
•	Трейсинг транзакций

# 📊 **Prometheus + Grafana**
•	Готовые дашборды:
  o	HTTP-метрики (RPS, latency)
  o	Database queries
  o	Kafka consumer lag
•	Кастомные метрики бизнес-логики

# ⚡ **Kafka Consumer**
•	Обработка событий:
  o	client_created
  o	client_updated
  o	client_deleted
•	Exactly-once семантика
•	Retry-механизм

# 🧩 **Elasticsearch 8.x**
•	Оптимизированный поиск по:
  o	Имени клиента
  o	Email
  o	Телефону
•	Авто-индексация при изменениях

# 📝 **Особенности реализации**
•	Graceful shutdown 
•	Модульная архитектура 
•	Полный CI/CD (GitHub Actions)
•	Конфигурация через ENV
•	Интеграционные тесты 

# 📄 **Лицензия**
MIT License
