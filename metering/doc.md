---
title: "Начало работы с Marketplace Metering API"
description: "вы научитесь отправлять метрики потребления вашего продукта в Marketplace Metering API."
---


# Как начать работать с Marketplace Metering API

В этом разделе вы научитесь отправлять метрики потребления вашего продукта в Marketplace Metering API.

## Перед началом работы {#before-begin}

Чтобы начать работать c Marketplace Metering API:

1. В Партнерском кабинете найдите Product ID и SKU ID вашего продукта.
2. Назначьте [сервисному аккаунту](../../iam/concepts/users/service-accounts.md), от имени которого вы будете отправлять
   метрики, роль `marketplace.meteringAgent` в вашем [фолдере](../../resource-manager/concepts/folder.md).
3. [Получите](./authentication.md) IAM-токен для сервисного аккаунта, от имени которого вы будете
   аутентифицироваться в Marketplace Metering API.

Чтобы воспользоваться примерами, установите [cURL](https://curl.haxx.se)
и [gRPCurl](https://github.com/fullstorydev/grpcurl) (при использовании [gRPC API](../api-ref/grpc/)).

## Проверка возможности отправки метрик {#dry-run}

Запрос на проверку возможности отправки метрик позволяет убедиться, что вы можете отправлять метрики в Marketplace
Metering API, чтобы начать выполнять полезную работу. От реальной отправки метрик запрос отличается только параметром
`validate_only`, который устанавливается в `true`.
Чтобы проверить возможность отправки метрик, выполните следующую команду:

{% list tabs group=api_type %}

- REST API {#rest-api}

  ```bash
  curl \
    --request POST \
    --url 'https://marketplace.{{ api-host }}/marketplace/metering/v1/imageProductUsage/write' \
    --header 'Authorization: Bearer <IAM-токен>' \
    --header 'Content-Type: application/json' \
    --data '{
      "validate_only": true,
      "product_id": "<product_id>",
      "usage_records": [
        {
          "uuid": "<uuid>",
          "sku_id": "<sku_id>",
          "quantity": <quantity>,
          "timestamp": "<timestamp>"
        }
      ]
    }'
  ```

  Где:
    * `<IAM-токен>` — полученный перед началом работы IAM-токен.
    * `<product_id>` — идентификатор продукта.
    * `<sku_id>` — идентификатор SKU.
    * `<uuid>` — уникальный идентификатор записи. Может быть сгенерирован с помощью `uuidgen`.
    * `<timestamp>` — временная метка в формате ISO 8601. Например, `2024-09-16T19:01:10.591128Z`.
    * `<quantity>` — целое число, количество потребленных единиц продукта. Должно быть больше 0. Например, 1.

- gRPC API {#grpc-api}

  ```bash
  grpcurl \
    -rpc-header "Authorization: Bearer <IAM-токен>" \
    -d '{
      "validate_only": true,
      "product_id": "<product_id>",
      "usage_records": [
        {
          "uuid": "<uuid>",
          "sku_id": "<sku_id>",
          "quantity": <quantity>,
          "timestamp": <timestamp>
        }
      ]
    }' \
    marketplace.{{ api-host }}:443 yandex.cloud.marketplace.metering.v1.ImageProductUsageService/Write
  ```


  Где:
    * `<IAM-токен>` — полученный перед началом работы IAM-токен.
    * `<product_id>` — идентификатор продукта.
    * `<sku_id>` — идентификатор SKU.
    * `<uuid>` — уникальный идентификатор записи. Может быть сгенерирован с помощью `uuidgen`.
    * `<timestamp>` — временная метка в формате ISO 8601. Например, `2024-09-16T19:01:10.591128Z`.
    * `<quantity>` — целое число, количество потребленных единиц продукта. Должно быть больше 0. Например, 1.

{% endlist %}

## Отправка метрики {#send-metric}

Чтобы отправить метрику, выполните следующую команду:

{% list tabs group=api_type %}

- REST API {#rest-api}

  ```bash
  curl \
    --request POST \
    --url 'https://marketplace.{{ api-host }}/marketplace/metering/v1/imageProductUsage/write' \
    --header 'Authorization: Bearer <IAM-токен>' \
    --header 'Content-Type: application/json' \
    --data '{
      "product_id": "<product_id>",
      "usage_records": [
        {
          "uuid": "<uuid>",
          "sku_id": "<sku_id>",
          "quantity": <quantity>,
          "timestamp": "<timestamp>"
        }
      ]
    }'
  ```

  Где:
    * `<IAM-токен>` — полученный перед началом работы IAM-токен.
    * `<product_id>` — идентификатор продукта.
    * `<sku_id>` — идентификатор SKU.
    * `<uuid>` — уникальный идентификатор записи. Может быть сгенерирован с помощью `uuidgen`.
    * `<timestamp>` — временная метка в формате `2024-09-16T19:01:10.591128Z`.
    * `<quantity>` — целое число, количество потребленных единиц продукта. Должно быть больше 0. Например, 1.


- gRPC API {#grpc-api}

  ```bash
  grpcurl \
    -rpc-header "Authorization: Bearer <IAM-токен>" \
    -d '{
      "product_id": "<product_id>",
      "usage_records": [
        {
          "uuid": "<uuid>",
          "sku_id": "<sku_id>",
          "quantity": <quantity>,
          "timestamp": "<timestamp>"
        }
      ]
    }' \
    marketplace.{{ api-host }}:443 yandex.cloud.marketplace.metering.v1.ImageProductUsageService/Write
  ```

  Где:
    * `<IAM-токен>` — полученный перед началом работы IAM-токен.
    * `<product_id>` — идентификатор продукта.
    * `<sku_id>` — идентификатор SKU.
    * `<uuid>` — уникальный идентификатор записи. Может быть сгенерирован с помощью `uuidgen`.
    * `<timestamp>` — временная метка в формате ISO 8601. Например, `2024-09-16T19:01:10.591128Z`.
    * `<quantity>` — целое число, количество потребленных единиц продукта. Должно быть больше 0. Например, 1.

{% endlist %}

Успешный результат:

```json
{
  "accepted": [
    {
      "uuid": "<uuid>",
      "sku_id": "<sku_id>",
      "quantity": 1,
      "timestamp": "<timestamp>"
    }
  ],
  "rejected": []
}
```

Если метрика не была принята, возвращается список отклоненных записей.

```json
{
  "accepted": [],
  "rejected": [
    {
      "uuid": "<uuid>",
      "reason": "<reason>"
    }
  ]
}
```

Где:

- `<uuid>` — уникальный идентификатор записи.
- `<reason>` — причина отклонения метрики. Возможные значения:
    * `DUPLICATE` — дубликат записи. Запись с таким же UUID уже существует.
    * `EXPIRED` — запись просрочена. Нельзя досылать метрики за потребление старше часа.
    * `INVALID_TIMESTAMP` — неверная временная метка. Метрики нельзя слать в будущее.
    * `INVALID_SKU_ID` — неверный идентификатор SKU. SKU не найден.
    * `INVALID_PRODUCT_ID` — неверный идентификатор продукта. Продукт не найден.
    * `INVALID_QUANTITY` — неверное количество. Количество должно быть больше 0.
    * `INVALID_ID` — неверный идентификатор записи. Идентификатор не является UUID.

Также API может вернуть ошибку сервера с кодом `403` и сообщением `Forbidden`. Это означает, что у сервисного аккаунта
нет прав на отправку метрик. То есть они либо были отозваны пользователем, либо платежный аккаунт оплачивающий облако,
где создан сервисный аккаунт, перешел в статус, в котором потребление невозможно.
Подробнее в документации
по [статусам аккаунтов](../../billing/concepts/billing-account-statuses).

Успешное выполнение проверки возможности отправки метрик не гарантирует, что в итоге метрика будет принята. Так как эти
два события разнесены во времени, между ними могли произойти изменения в правах доступа или в статусе платежного
аккаунта. Делите полезную работу на минимально значимые части и отправляйте метрики после каждой из них. Например, если
ваше приложение отправляет письма, то метрику можно отправить после отправки каждого письма.

Вы можете группировать метрики по разным SKU в один запрос. А также репортить метрики сразу за некоторое количество
работы. Например, если ваше приложение шлет запросы с большим RPS, то метрику оно может отправлять раз в минуту, где
в `quantity` указывать количество успешно отправленных запросов за эту минуту.