"""Structured (JSON) logging setup for the AI inference service.

Uses only the standard library so no extra logging dependency is
required.
"""

import json
import logging
import sys
from datetime import datetime, timezone

_RESERVED_ATTRS = set(logging.LogRecord("", 0, "", 0, "", None, None).__dict__.keys())


class JSONFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "timestamp": datetime.fromtimestamp(record.created, tz=timezone.utc).isoformat(),
            "level": record.levelname.lower(),
            "logger": record.name,
            "message": record.getMessage(),
            "service": "ai-inference",
        }

        if record.exc_info:
            payload["exception"] = self.formatException(record.exc_info)

        for key, value in record.__dict__.items():
            if key not in _RESERVED_ATTRS:
                payload[key] = value

        return json.dumps(payload, default=str)


def configure_logging(level_name: str = "info") -> None:
    level = getattr(logging, level_name.upper(), logging.INFO)

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(JSONFormatter())

    root = logging.getLogger()
    root.handlers = [handler]
    root.setLevel(level)
