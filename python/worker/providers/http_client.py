from __future__ import annotations

import json
from typing import Any
from urllib import error, request


def post_json(url: str, headers: dict[str, str], payload: dict[str, Any], timeout: float = 60.0) -> dict[str, Any]:
    data = json.dumps(payload).encode("utf-8")
    req = request.Request(url, data=data, headers=headers, method="POST")
    try:
        with request.urlopen(req, timeout=timeout) as response:
            body = response.read().decode("utf-8")
    except error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{exc.code} {exc.reason}: {body}") from exc
    except error.URLError as exc:
        raise RuntimeError(str(exc.reason)) from exc

    if not body:
        return {}
    return json.loads(body)
