import json

from flask import session
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2 import ClaimProductInstanceMetadata
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2 import ClaimProductInstanceRequest
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2_grpc import ProductInstanceServiceStub
from yandexcloud import SDK

import db
from models import User


def login(user_login: str, password: str):
    db_user = db.read_user(user_login)

    if not db_user.check_password(password):
        return "Invalid password or login"

    session['login'] = user_login
    return None


def register(login: str, password: str):
    user = User.create(login, password)
    try:
        db.create_user(user.login, user.password_hash)
    except Exception as e:
        return "User already exists"
    session['login'] = user.login
    return None


def bind(user_login: str, token: str):
    key_path = '/Users/nikthespirit/Documents/experiment/mkt-metering-demo/key.json'
    with open(key_path, 'r') as f:
        key = f.read()
    service_account_key = json.loads(key)
    sdk = SDK(service_account_key=service_account_key)
    pim = sdk.client(ProductInstanceServiceStub)
    try:
        operation = pim.Claim(ClaimProductInstanceRequest(token=token))
    except Exception as e:
        print(e)
        return "Invalid token"
    res = sdk.wait_operation_and_get_result(
        operation,
        meta_type=ClaimProductInstanceMetadata,
    )
    product_instance_id = res.response.product_instance_id
    db.update_user(user_login, product_instance_id)
    return None


def emulate_work():
    pass
