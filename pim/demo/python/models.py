import dataclasses
from datetime import datetime, timedelta  # Changed import
import uuid
from typing import Optional
import bcrypt


@dataclasses.dataclass
class User:
    """User model for authentication and identification."""
    login: str
    password_hash: str
    product_instance_id: Optional[str] = None

    @classmethod
    def create(cls, login: str, password: str) -> "User":
        """Create a new user with hashed password.

        Args:
            login: User's login identifier
            password: Plain text password to be hashed

        Returns:
            A new User instance
        """
        # Use a stronger work factor (12 is a good default)
        password_hash = bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt(rounds=12))  # Added rounds=12
        return cls(login=login, password_hash=password_hash.decode('utf-8'), product_instance_id=None)

    def check_password(self, password: str) -> bool:
        """Verify if the provided password matches the stored hash.

        Args:
            password: Plain text password to check

        Returns:
            True if password matches, False otherwise
        """
        try:
            return bcrypt.checkpw(password.encode('utf-8'), self.password_hash.encode('utf-8'))
        except (ValueError, TypeError):
            return False


@dataclasses.dataclass
class Session:
    """User session for maintaining authentication state."""
    id: str
    login: str
    created_at: datetime = dataclasses.field(default_factory=datetime.utcnow)
    expires_at: Optional[datetime] = None

    @classmethod
    def create(cls, login: str, expiry_hours: int = 24) -> "Session":
        """Create a new session for a user.

        Args:
            login: User's login identifier
            expiry_hours: Number of hours until session expiration

        Returns:
            A new Session instance
        """
        session_id = str(uuid.uuid4())
        expires_at = datetime.utcnow() + timedelta(hours=expiry_hours)
        return cls(id=session_id, login=login, expires_at=expires_at)

    @property
    def is_valid(self) -> bool:
        """Check if the session is still valid (not expired)."""
        if not self.expires_at:
            return True
        return datetime.utcnow() < self.expires_at
