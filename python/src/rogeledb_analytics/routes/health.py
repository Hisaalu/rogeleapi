from fastapi import APIRouter

from ..go_client import GoAPIError, go_client

router = APIRouter(tags=["health"])


@router.get("/health")
async def health() -> dict:
    """Liveness for the analytics service itself."""
    return {"status": "ok", "service": "rogeledb-analytics"}


@router.get("/health/upstream")
async def health_upstream() -> dict:
    """Readiness: can this service reach the Go API with its service token?"""
    try:
        await go_client.get("/admin/stats")
        return {"status": "ok", "upstream": "reachable"}
    except GoAPIError as exc:
        return {
            "status": "error",
            "upstream": "unreachable",
            "code": exc.status_code,
            "message": exc.message,
        }