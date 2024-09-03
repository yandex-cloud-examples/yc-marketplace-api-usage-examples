import argparse
import json
import os
import time
from os import path

import grpc
import yandexcloud
from yandex.cloud.api.operation_pb2 import operation
from yandex.cloud.marketplace.licensemanager.v1.lock_pb2 import Lock
from yandex.cloud.marketplace.licensemanager.v1.lock_service_pb2 import ListLocksRequest, EnsureLockRequest, \
    EnsureLockMetadata
from yandex.cloud.marketplace.licensemanager.v1.lock_service_pb2_grpc import (
    LockServiceStub
)

script_dir = os.path.dirname(os.path.realpath(__file__))

KNOWN_TEMPLATES = {
    "template-1": "Basic Subscription",
    "template-2": "Advanced Subscription",
}


def main(resource_id, folder_id, fake=None, service_account_key_path=None):
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
    client = sdk.client(LockServiceStub, endpoint=endpoint)

    # Step 1. List locks
    response = client.List(ListLocksRequest(
        resource_id=resource_id,
        folder_id=folder_id,
    ))
    lock = None
    # Step 2. Write the product usage to Yandex.Cloud API (validate_only=False)
    for l in response.locks:
        if l.state == Lock.State.LOCKED and l.template_id in KNOWN_TEMPLATES:
            # If the l is locked, we can write the product usage
            lock = l
            break

    if lock is None:
        return "No locked license found"

    for i in range(5):
        op = client.Ensure(EnsureLockRequest(
            resource_id=resource_id,
            instance_id=lock.instance_id,
        ))  # type: operation.Operation

        if not op.response:
            return f"Error: {op.error}"

        op_resp = Lock()
        op.response.Unpack(op_resp)
        print(f"Lock ensured. Operation status: {op_resp}")
        time.sleep(1)

    return response


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--resource-id", help="Resource ID", required=True)
    parser.add_argument("--folder-id", help="Folder ID", required=True)
    parser.add_argument("--fake", help="Use local fake endpoint", action="store_true")
    parser.add_argument("--service-account-key", help="Service account key", required=False)

    args = parser.parse_args()

    print(
        main(
            args.resource_id,
            args.folder_id,
            args.fake,
            args.service_account_key,
        )
    )
