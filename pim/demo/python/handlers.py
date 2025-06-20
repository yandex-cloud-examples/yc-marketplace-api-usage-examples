import datetime
import json
import uuid
from typing import Optional

from flask import session
from yandex.cloud.marketplace.metering.v1.product_usage_service_pb2 import WriteUsageRequest
from yandex.cloud.marketplace.metering.v1.product_usage_service_pb2_grpc import ProductUsageServiceStub
from yandex.cloud.marketplace.pim.v1.saas.product_instance_pb2 import ProductInstance
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2 import ClaimProductInstanceRequest
from yandex.cloud.marketplace.pim.v1.saas.product_instance_service_pb2_grpc import ProductInstanceServiceStub
from yandexcloud import SDK

import db
from models import Session
from models import User
import config  # Import the new config module


def login(user_login_param: str, password: str) -> Optional[str]:
    """Authenticates a user and creates a session.

    Args:
        user_login_param: The login identifier of the user.
        password: The user's password.

    Returns:
        An error message string if authentication fails, otherwise None.
    """
    db_user = db.read_user(user_login_param)

    if db_user is None or not db_user.check_password(password):
        return "Invalid password or login"

    # Create and store new session
    new_db_session = Session.create(login=db_user.login)
    db.create_session(new_db_session)
    session['session_id'] = new_db_session.id
    session.pop('login', None)  # Remove old login key if present
    return None


def register(new_login: str, password: str) -> Optional[str]:
    """Registers a new user and creates a session.

    Args:
        new_login: The desired login identifier for the new user.
        password: The desired password for the new user.

    Returns:
        An error message string if registration fails (e.g., user already exists),
        otherwise None.
    """
    user = User.create(new_login, password)
    try:
        db.create_user(user.login, user.password_hash)
    except Exception:  # Consider more specific exception handling if YDB driver provides it
        return "User already exists"

    # Create and store new session
    new_db_session = Session.create(login=user.login)
    db.create_session(new_db_session)
    session['session_id'] = new_db_session.id
    session.pop('login', None)  # Remove old login key if present
    return None


def sdk() -> SDK:
    """Initializes and returns the Yandex Cloud SDK instance.

    Returns:
        An initialized SDK instance.

    Raises:
        FileNotFoundError: If the key.json file is not found.
        json.JSONDecodeError: If key.json is not valid JSON.
        # Other exceptions from SDK initialization.
    """
    with open(config.KEY_PATH, 'r') as f:  # Use KEY_PATH from config
        key = f.read()
    service_account_key = json.loads(key)
    sdk_instance = SDK(service_account_key=service_account_key)
    return sdk_instance


def bind(user_login: str, token: str) -> Optional[str]:
    """Binds a user to a product instance using a token.

    Args:
        user_login: The login identifier of the user to bind.
        token: The product instance claim token.

    Returns:
        An error message string if binding fails, otherwise None.
    """
    # It's generally better to get the user from the session via get_user()
    # but keeping user_login param for now as per existing structure.
    # Ensure user_login corresponds to the currently authenticated user.

    pim_service = sdk().client(ProductInstanceServiceStub)
    try:
        operation = pim_service.Claim(ClaimProductInstanceRequest(token=token))
        pi = ProductInstance()
        operation.response.Unpack(pi)
    except Exception as e:
        # Log the actual exception e for debugging
        print(f"Error claiming product instance: {e}")
        return "Invalid token or error during claim operation"

    print(f"ProductInstance ID: {pi.id} claimed for user {user_login}")
    db.update_user(user_login, pi.id)
    return None


def get_user() -> Optional[User]:
    """Retrieves the currently authenticated user based on the session.

    Validates the session stored in Flask's session cookie against the database.
    If the session is invalid or not found, it's cleared.

    Returns:
        A User object if a valid session exists, otherwise None.
    """
    session_id = session.get('session_id')
    if not session_id:
        return None

    db_sess = db.read_session(session_id)

    if db_sess is None or not db_sess.is_valid:
        session.pop('session_id', None)  # Clear Flask session
        if db_sess is not None:  # Session exists in DB but is invalid (e.g. expired)
            db.delete_session(session_id)  # Clean up DB
        return None

    # Session is valid, retrieve the user
    user = db.read_user(db_sess.login)
    if user is None:
        # This case (valid session but no user) should ideally not happen
        # if data integrity is maintained.
        session.pop('session_id', None)
        db.delete_session(session_id)
        return None

    return user


def emulate_work(amount: int = 1) -> Optional[str]:
    """Emulates work and reports usage for the authenticated user.

    Args:
        amount: The quantity of usage to report. Defaults to 1.

    Returns:
        An error message string if usage reporting fails or user is not set up,
        otherwise None.
    """
    current_user = get_user()
    if current_user is None:
        return "User not authenticated or session invalid. Please log in."

    if current_user.product_instance_id is None:
        return "User is not bound to a product instance. Please bind a token first."

    metering_service = sdk().client(ProductUsageServiceStub)
    try:
        res, call = metering_service.Write.with_call(WriteUsageRequest(
            dry_run=False,  # Set to True for testing without actual metering
            product_instance_id=current_user.product_instance_id,
            usage_records=[
                {
                    "uuid": str(uuid.uuid4()),
                    "sku_id": config.SKU_ID,  # Use SKU_ID from config
                    "quantity": amount,
                    "timestamp": {
                        "seconds": int(datetime.datetime.now().timestamp()),
                        "nanos": 0,
                    }
                },
            ],
        ))
        print(f"Usage reported: {res}, Metadata: {{m.key: m.value for m in call._call._state.initial_metadata}}")
    except Exception as e:
        print(f"Error reporting usage: {e}")
        # Consider more specific error parsing if SDK provides details in e.args
        # For example, e.args[0].details() for gRPC errors
        # print({m.key: m.value for m in e.args[0].initial_metadata}) # If error has metadata
        return "Error reporting usage. Check logs for details."
    return None


def logout_user() -> None:
    """Logs out the current user by deleting their session."""
    session_id = session.pop('session_id', "")
    if session_id:
        db.delete_session(session_id)
    # Also clear any other session keys if necessary
    session.pop('login', None)  # Just in case old key is still around
