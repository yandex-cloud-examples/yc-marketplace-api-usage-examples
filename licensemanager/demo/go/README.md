Для запуска примера выполните следующую команду:

```bash
go run main.go --service-account-key ./key.json --resource-id resource-1 --folder-id folder --fake
```
При этом должен быть запущен фейковый сервер, который будет обрабатывать запросы.

Получить необходимые значения для параметров `resource-id` и `folder-id` внутри виртуальной машины можно с помощью команды:

```bash
curl -H Metadata-Flavor:Google 169.254.169.254/computeMetadata/v1/instance/?recursive=true | jq '{id: .id, folderId: .vendor.folderId}'
```

А также если, параметры не указать, то демо приложение попробует получить их из метаданных виртуальной машины.