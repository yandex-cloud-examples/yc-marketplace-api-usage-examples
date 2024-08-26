import argparse
import json
import os
import time
from os import path
from uuid import uuid4

import grpc
import yandexcloud
from yandex.cloud.marketplace.metering.v1.image_product_usage_service_pb2 import (
    WriteImageProductUsageRequest,
)
from yandex.cloud.marketplace.metering.v1.image_product_usage_service_pb2_grpc import (
    ImageProductUsageServiceStub,
)
from yandex.cloud.marketplace.metering.v1.usage_record_pb2 import UsageRecord

script_dir = os.path.dirname(os.path.realpath(__file__))


def build_product_usage_write_request(product_id, sku_id, quantity, timestamp=None, uuid=None):
    """Builds an image product usage write request."""

    usage_record = UsageRecord()

    # NOTE: Behaves like idempotency key. Should be unique to prevent duplicates.
    usage_record.uuid = str(uuid4()) if uuid is None else str(uuid)
    usage_record.sku_id = sku_id
    usage_record.quantity = int(quantity)

    # NOTE: UTC timezone
    usage_record.timestamp.seconds = int(time.time()) if timestamp is None else int(timestamp)

    request = WriteImageProductUsageRequest()

    request.product_id = product_id
    request.usage_records.extend([usage_record])

    return request


def business_logic(product_id, sku_id):
    """Example of service."""

    if product_id == "Secure Firewall" and sku_id == "Ingress network traffic":
        return 1 + 1

    if product_id == "Secure Firewall" and sku_id == "Egress network traffic":
        return 1 * 1

    return 0


def validate_write_response(response):
    # NOTE: Some usage records can be accepted or rejected. Please pay attention to the following fields:

    # response.rejected - list of rejected usage records
    # response.accepted - list of accepted usage records

    if len(response.rejected) > 0:
        error_msg = "Unable to provide the service to customer. Rejected: %s, Accepted: %s."
        raise ValueError(error_msg % (str(response.rejected), str(response.accepted)))
    elif len(response.accepted) == 0:
        error_msg = "Unable to provide the service to customer. Got empty list of accepted metrics."
        raise ValueError(error_msg)


def main(product_id, sku_id, quantity, timestamp=None, uuid=None, fake=None, service_account_key_path=None):
    # NOTE: IAM token will be taken automatically from metadata agent of VM
    interceptor = yandexcloud.RetryInterceptor(max_retry_count=5, retriable_codes=[grpc.StatusCode.UNAVAILABLE])
    root_certificates = None
    endpoint = None

    # If you want to use fake local server, you should provide root certificates
    if fake:
        endpoint = "api.yc.local:8080"
        with open(path.join(script_dir, "../../fake/x509/ca.crt"), "rb") as f:
            root_certificates = f.read()
    # If you are running this script not on Yandex.Cloud VM, you should provide service account key
    if service_account_key_path:
        with open(service_account_key_path, "r") as f:
            service_account_key = json.loads(f.read())

    sdk = yandexcloud.SDK(
        interceptor=interceptor,
        endpoint=endpoint,
        root_certificates=root_certificates,
        service_account_key=service_account_key,
    )
    client = sdk.client(ImageProductUsageServiceStub, endpoint=endpoint)
    request = build_product_usage_write_request(product_id, sku_id, quantity, timestamp, uuid)

    # Step 0. Ensure consumer has all permissions to use the product (validate_only=True)

    request.validate_only = True
    response = client.Write(request)

    validate_write_response(response)

    # Step 1. Provide your service to the customer

    business_logic(product_id, sku_id)

    # Step 2. Write the product usage to Yandex.Cloud API (validate_only=False)

    request.validate_only = False
    response = client.Write(request)

    validate_write_response(response)

    return response


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--product-id", help="Marketplace image product ID", required=True)
    parser.add_argument("--sku-id", help="Marketplace image product SKU", required=True)
    parser.add_argument("--quantity", help="Usage quantity", required=True)
    parser.add_argument("--timestamp", help="Usage time", required=False)
    parser.add_argument("--uuid", help="Usage request unique identifier", required=False)
    parser.add_argument("--fake", help="Use local fake endpoint", action="store_true")
    parser.add_argument("--service-account-key", help="Service account key", required=False)

    args = parser.parse_args()

    print(
        main(
            args.product_id,
            args.sku_id,
            args.quantity,
            args.timestamp,
            args.uuid,
            args.fake,
            args.service_account_key,
        )
    )
