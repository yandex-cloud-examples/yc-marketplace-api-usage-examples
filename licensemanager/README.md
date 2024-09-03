В этой директории находятся фейковый сервер и примеры клиентов для работы с ним, написанные на Go и Python.

## Запуск сервера

```bash
docker-compose up --build
```

Для корректной работы фейкового сервера необходимо в `/etc/hosts` добавить запись `api.yc.local`:

```
127.0.0.1 api.yc.local
::1 api.yc.local
```

Для конфигурации сервера используются переменные окружения:

`PORT` - порт, на котором будет запущен сервер (по умолчанию 8080)

`LOCKS` - список блокировок. В данном примере шаблон `template-1` заблокирован для ресурса `resource-1`. Пример:
```json
[
  {
    "template_id": "template-1",
    "resource_id": "resource-1"
  }
]
```

`ENDPOINT` - адрес, на котором будет доступен сервер. По умолчанию `api.yc.local`.

## Подготовка к запуску примеров

Для того чтобы подготовить окружение для запуска примера, выполните следующие команды:

```bash
yc iam service-account create --name lm
SA_ID=$(yc iam service-account get --name lm --format json | jq -r .id)
FOLDER_ID=$(yc config get folder-id)
yc resource-manager folder add-access-binding --role license-manager.user --subject serviceAccount:$SA_ID --id $FOLDER_ID
yc iam key create --service-account-id $SA_ID --output key.json --description "Key for lm service account"
```

Несмотря на то, что фейковый сервер не проверяет подлинность ключа и не выпускает токен, ключ все равно понадобится,
чтобы максимально приблизить пример к реальной работе с сервисом. И чтобы было легко переключаться между реальным и
фейковым сервером.


## Запуск примера на Go

Для запуска примера выполните следующую команду:

```bash
(
  cd ./demo/go
  go run main.go --service-account-key ./key.json --resource-id resource-1 --folder-id folder --fake
)
```