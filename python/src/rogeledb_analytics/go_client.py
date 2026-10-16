import logging

import httpx

from .config import settings

logger = logging.getLogger(__name__)


class GoAPIError(Exception):
    """Raised when the Go API returns an error response."""

    def __init__(self, status_code: int, message: str):
        self.status_code = status_code
        self.message = message
        super().__init__(f"Go API error {status_code}: {message}")


class GoAPIClient:
    """Async HTTP client for the Go Rogele API.

    Uses a service-account JWT for authentication. Every request goes through
    `get` / `post`, both of which attach the Authorization header and decode
    the JSON response.
    """

    def __init__(self) -> None:
        self._base_url = settings.rogele_api_url.rstrip("/")
        self._token = settings.rogele_service_token
        self._timeout = settings.http_timeout

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict | None = None,
        json_body: dict | None = None,
    ) -> dict | list:
        url = f"{self._base_url}/api/v1{path}"
        headers = {
            "Accept": "application/json",
            "Content-Type": "application/json",
        }
        if self._token:
            headers["Authorization"] = f"Bearer {self._token}"

        async with httpx.AsyncClient(timeout=self._timeout) as client:
            response = await client.request(
                method,
                url,
                params=params,
                json=json_body,
                headers=headers,
            )

        if response.status_code >= 400:
            message = response.text or "unknown error"
            logger.warning("Go API %s %s -> %s: %s", method, path, response.status_code, message)
            raise GoAPIError(response.status_code, message)

        if response.status_code == 204:
            return {}

        return response.json()

    async def get(self, path: str, *, params: dict | None = None) -> dict | list:
        return await self._request("GET", path, params=params)

    async def post(self, path: str, *, json_body: dict | None = None) -> dict | list:
        return await self._request("POST", path, json_body=json_body)


go_client = GoAPIClient()