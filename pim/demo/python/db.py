import os

import ydb

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
    session = driver.table_client.session().create()
except TimeoutError:
    raise RuntimeError("Connect failed to YDB")

pool = ydb.QuerySessionPool(driver)


def create_tables():
    pool.execute_with_retries("""
        CREATE TABLE IF NOT EXISTS users (
            login Utf8,
            password_hash Utf8,
            product_instance_id Utf8,

            PRIMARY KEY(login)
        );
        CREATE TABLE IF NOT EXISTS sessions (
            id Utf8,
            login Utf8,

            PRIMARY KEY(id)
        );
    """)


def create_user(login, password_hash):
    pool.execute_with_retries("""
        DECLARE $login AS Utf8;
        DECLARE $password_hash AS Utf8;
        INSERT INTO users (login, password_hash)
        VALUES ($login, $password_hash);
    """, {
        # "$seriesData": (basic_example_data.get_series_data(), basic_example_data.get_series_data_type()),
        "$login": (login, ydb.PrimitiveType.Utf8),
        "$password_hash": (password_hash, ydb.PrimitiveType.Utf8),
    }, )


def create_session(id, login):
    pool.execute_with_retries("""
        DECLARE $id AS Utf8;
        DECLARE $login AS Utf8;
        
        INSERT INTO sessions (id, login)
        VALUES ($id, $login);
    """, {
        "$id": (id, ydb.PrimitiveType.Utf8),
        "$login": (login, ydb.PrimitiveType.Utf8),
    })


def read_user(login: str) -> User:
    result_sets = pool.execute_with_retries("""
        DECLARE $login AS Utf8;
        SELECT login, password_hash, product_instance_id
        FROM users
        WHERE login = $login;
    """, {
        "$login": (login, ydb.PrimitiveType.Utf8),
    })
    first_set = result_sets[0]
    row = first_set.rows[0]
    return User(
        login=row.login,
        password_hash=row.password_hash,
        product_instance_id=row.product_instance_id,
    )

def update_user(login, product_instance_id):
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

