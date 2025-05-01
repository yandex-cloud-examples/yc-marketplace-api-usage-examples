import datetime
import json
import uuid

from flask import session
from yandex.cloud.marketplace.metering.v1.product_usage_service_pb2 import WriteUsageRequest
from yandex.cloud.marketplace.metering.v1.product_usage_service_pb2_grpc import ProductUsageServiceStub
from yandex.cloud.marketplace.pim.v1.saas.product_instance_pb2 import ProductInstance
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2 import ClaimProductInstanceRequest
from yandexcloud import SDK

import db
from models import User

sku_id = 'dn2e395635hoblkrj1at'

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

def sdk():
    key_path = '../../../key.json'
    with open(key_path, 'r') as f:
        key = f.read()
    service_account_key = json.loads(key)
    sdk = SDK(service_account_key=service_account_key)
    return sdk

def bind(user_login: str, token: str):

    pim = sdk().client(ProductInstanceServiceStub)
    try:
        operation = pim.Claim(ClaimProductInstanceRequest(token=token))
        pi = ProductInstance()
        operation.response.Unpack(pi)
    except Exception as e:
        print(e)
        return "Invalid token"
    print(pi.id)
    db.update_user(user_login, pi.id)
    return None


def get_user():
    if 'login' not in session:
        return None
    user_login = session['login']
    db_user = db.read_user(user_login)
    return db_user


def emulate_work(amount: int = 1):
    if 'login' not in session:
        return None
    user_login = session['login']
    db_user = db.read_user(user_login)
    if db_user.product_instance_id is None:
        return None

    metering = sdk().client(ProductUsageServiceStub)
    try:

        res = metering.Write(WriteUsageRequest(
            dry_run=False,
            product_instance_id=db_user.product_instance_id,
            usage_records=[
                {
                    "uuid": str(uuid.uuid4()),
                    "sku_id": sku_id,
                    "quantity": amount,
                    "timestamp": {
                        "seconds": int(datetime.datetime.now().timestamp()),
                        "nanos": 0,
                    }
                },
            ],
        ))
        print(res)
    except Exception as e:
        print("err", e)
        return "Invalid token"