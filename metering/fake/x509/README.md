Чтобы сгенерировать самоподписанный сертификат, выполните следующую команду:

Шаг 1: Создание корневого сертификата (ca.crt)
```bash
openssl genrsa -passout pass:1111 -des3 -out ca.key 4096
openssl req -passin pass:1111 -new -x509 -days 365 -key ca.key -out ca.crt -config <(cat csr_ca.txt)
```

Шаг 2: Создание приватного ключа сервера (server.key)
```bash
openssl genrsa -passout pass:1111 -des3 -out server.key 4096
```

Шаг 3: Получение запроса на подпись сертификата от корневого сертификата (server.csr)
```bash
openssl req -passin pass:1111 -new -key server.key -out server.csr -config <(cat api_answer.txt)
```

Шаг 4: Подпись сертификата корневым сертификатом (server.crt)
```bash
openssl x509 -req -passin pass:1111 -CA ca.crt -CAkey ca.key -CAcreateserial \
  -in server.csr -out server.crt -days 365 \
  -extensions 'req_ext' -extfile <(cat api_answer.txt)
```

Шаг 5: Преобразование сертификата сервера в формат .pem (server.pem) - используемый gRPC
```bash
openssl pkcs8 -topk8 -nocrypt -passin pass:1111 -in server.key -out server.pem
```

Все шаги сразу можно выполнить с помощью скрипта:
```bash
./ssl.sh
```