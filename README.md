# Обработчик задач

## Собрать контейнер
```
docker build -t job-worker .
```

## Запустить контейнер
```
docker run -d -p 8080:8080 --name my-worker job-worker
```

⚙️ Доступные флаги

Вы можете настроить приложение с помощью следующих флагов:

Флаг | 	Описание |	Значение по умолчанию
--- | --- | --- |
-goroutines_count	| Количесто воркеров | 5
-port	| Порт приложения | 8080
-retrues | Количество повторных попыток | 3
-timeout | Ограничение времени на выполнение одной задачи(таймаут) | 3

Пример запуска с параметрами
```
docker run -d -p 8080:8081 \
  --name custom-worker \
  job-worker \
  --port=8081 \
  --workers_count=10 \
  --retries=5 \
  --timeout=5
```

## Посмотреть логи
```
docker logs -f my-worker
```

## Остановить контейнер
```
docker stop my-worker
```

## Удалить контейнер
```
docker rm my-worker
```