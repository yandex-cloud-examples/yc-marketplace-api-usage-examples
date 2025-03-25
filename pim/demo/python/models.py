import dataclasses

import bcrypt


@dataclasses.dataclass
class User:
    login: str
    password_hash: str
    product_instance_id: str | None

    @classmethod
    def create(cls, login:str, password: str):
        password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt())
        return cls(login=login, password_hash=password_hash.decode('utf-8'), product_instance_id=None)

    def check_password(self, password):
        return bcrypt.checkpw(password.encode('utf-8'), self.password_hash.encode('utf-8'))


@dataclasses.dataclass
class Session:
    id: str
    login: str

