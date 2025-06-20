import os
from typing import Optional  # Added Optional, Dict

import ydb

from models import Session  # Added Session
from models import User

driver_config = ydb.DriverConfig(
    endpoint=os.getenv("YDB_ENDPOINT"),
    database=os.getenv("YDB_DATABASE"),
    credentials=ydb.credentials_from_env_variables(),
    root_certificates=ydb.load_ydb_root_certificate(),
)
try:
    driver = ydb.Driver(driver_config)
    driver.wait(timeout=5)
except TimeoutError:
    raise RuntimeError("Connect failed to YDB")

pool = ydb.QuerySessionPool(driver)


def create_tables() -> None:  # Added return type
    """Creates database tables (users, sessions) if they don't already exist."""
    pool.execute_with_retries("""
                              CREATE TABLE IF NOT EXISTS users
                              (
                                  login
                                  Utf8,
                                  password_hash
                                  Utf8,
                                  product_instance_id
                                  Utf8,

                                  PRIMARY
                                  KEY
                              (
                                  login
                              )
                                  );
                              CREATE TABLE IF NOT EXISTS sessions
                              (
                                  id
                                  Utf8,
                                  login
                                  Utf8,
                                  created_at
                                  Timestamp,
                                  expires_at
                                  Timestamp,
                                  PRIMARY
                                  KEY
                              (
                                  id
                              )
                                  );
                              """)


def create_user(login: str, password_hash: str) -> None:  # Added type hints and return type
    """Creates a new user in the database.

    Args:
        login: User's login identifier.
        password_hash: Hashed password for the user.
    """
    pool.execute_with_retries("""
        DECLARE $login AS Utf8;
        DECLARE $password_hash AS Utf8;
        INSERT INTO users (login, password_hash)
        VALUES ($login, $password_hash);
    """, {
        "$login": (login, ydb.PrimitiveType.Utf8),
        "$password_hash": (password_hash, ydb.PrimitiveType.Utf8),
    })


def create_session(session: Session) -> None:  # Changed signature, added type hints
    """Creates a new session in the database, storing ID, login, created_at, and expires_at.

    Args:
        session: The Session object.
    """  # Updated docstring
    pool.execute_with_retries("""
        DECLARE $id AS Utf8;
        DECLARE $login AS Utf8;
        DECLARE $created_at AS Timestamp;
        DECLARE $expires_at AS Timestamp;
        
        INSERT INTO sessions (id, login, created_at, expires_at)
        VALUES ($id, $login, $created_at, $expires_at);
    """, {
        "$id": (session.id, ydb.PrimitiveType.Utf8),
        "$login": (session.login, ydb.PrimitiveType.Utf8),
        "$created_at": (int(session.created_at.timestamp()) * 10 ** 6, ydb.PrimitiveType.Timestamp),
        "$expires_at": (int(session.expires_at.timestamp()) * 10 ** 6 if session.expires_at else None,
                        ydb.PrimitiveType.Timestamp),
    })


def read_user(login: str) -> Optional[User]:  # Added Optional to return type
    """Reads a user from the database by login.

    Args:
        login: The login identifier of the user to retrieve.

    Returns:
        A User object if found, otherwise None.
    """
    result_sets = pool.execute_with_retries("""
        DECLARE $login AS Utf8;
        SELECT login, password_hash, product_instance_id
        FROM users
        WHERE login = $login;
    """, {
        "$login": (login, ydb.PrimitiveType.Utf8),
    })
    if not result_sets or not result_sets[0].rows:  # Added check for record not found
        return None
    row = result_sets[0].rows[0]
    return User(
        login=row.login,
        password_hash=row.password_hash,
        product_instance_id=row.product_instance_id,
    )


def update_user(login: str, product_instance_id: str) -> None:  # Added type hints and return type
    """Updates the product_instance_id for a given user.

    Args:
        login: The login identifier of the user to update.
        product_instance_id: The new product instance ID.
    """
    pool.execute_with_retries("""
        DECLARE $login AS Utf8;
        DECLARE $product_instance_id AS Utf8;
        UPDATE users
        SET product_instance_id = $product_instance_id
        WHERE login = $login;   
    """, {
        "$login": (login, ydb.PrimitiveType.Utf8),
        "$product_instance_id": (product_instance_id, ydb.PrimitiveType.Utf8),
    })


def read_session(session_id: str) -> Optional[Session]:
    """Reads a session from the database by session ID.

    Retrieves all session fields (id, login, created_at, expires_at)
    and reconstructs the Session object.

    Args:
        session_id: The ID of the session to retrieve.

    Returns:
        A Session object if found, otherwise None.
    """
    result_sets = pool.execute_with_retries("""
        DECLARE $id AS Utf8;
        SELECT id, login, created_at, expires_at
        FROM sessions
        WHERE id = $id;
    """, {
        "$id": (session_id, ydb.PrimitiveType.Utf8),
    })
    if not result_sets or not result_sets[0].rows:
        return None
    row = result_sets[0].rows[0]

    return Session(
        id=row.id,
        login=row.login,
        created_at=row.created_at,
        expires_at=row.expires_at
    )


def delete_session(session_id: str) -> None:
    """Deletes a session from the database by session ID.

    Args:
        session_id: The ID of the session to delete.
    """
    pool.execute_with_retries("""
        DECLARE $id AS Utf8;
        DELETE FROM sessions
        WHERE id = $id;
    """, {
        "$id": (session_id, ydb.PrimitiveType.Utf8),
    })
