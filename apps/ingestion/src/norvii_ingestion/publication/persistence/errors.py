"""Safe public errors for persistence configuration and connectivity commands."""


class PersistenceError(RuntimeError):
    """Represent a persistence failure whose message is safe for operator output."""


class PersistenceConnectionError(PersistenceError):
    """Indicate that a production driver could not be initialized."""
