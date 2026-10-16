import logging
from datetime import datetime, timedelta
from typing import Literal

import numpy as np
from fastapi import APIRouter, HTTPException, Query

from ..go_client import GoAPIError, go_client

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/analytics", tags=["analytics"])

Interval = Literal["day", "week", "month"]


def _default_dates() -> tuple[str, str]:
    to = datetime.utcnow()
    frm = to - timedelta(days=180)
    return frm.strftime("%Y-%m-%d"), to.strftime("%Y-%m-%d")


def _parse_date(value: str | None, *, default: str) -> str:
    if not value:
        return default
    try:
        datetime.strptime(value, "%Y-%m-%d")
    except ValueError:
        raise HTTPException(
            status_code=400,
            detail=f"date must be YYYY-MM-DD, got: {value}",
        ) from None
    return value


@router.get("/enrollment-trend")
async def enrollment_trend(
    interval: Interval = "month",
    frm: str | None = Query(None, alias="from"),
    to: str | None = Query(None),
) -> dict:
    """Signups per period, with a moving average.

    The moving average is computed here because it is a Python-strength
    calculation (numpy) and we want the frontend to render it directly
    without a second round trip.
    """
    default_from, default_to = _default_dates()
    from_date = _parse_date(frm, default=default_from)
    to_date = _parse_date(to, default=default_to)

    try:
        payload = await go_client.get(
            "/analytics/enrollment-trend",
            params={"interval": interval, "from": from_date, "to": to_date},
        )
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    points = payload.get("points") or []
    counts = [p["count"] for p in points]
    periods = [p["period"] for p in points]

    if counts:
        window = 3 if len(counts) >= 3 else len(counts)
        moving = np.convolve(counts, np.ones(window) / window, mode="valid").tolist()
        # Pad the start so lengths align with `periods`
        padding = [None] * (len(counts) - len(moving))
        moving = padding + moving
    else:
        moving = []

    series = [
        {"period": p, "count": c, "moving_average": m}
        for p, c, m in zip(periods, counts, moving, strict=True)
    ]

    return {
        "interval": interval,
        "from": from_date,
        "to": to_date,
        "points": series,
        "total": payload.get("total", 0),
    }


@router.get("/revenue-over-time")
async def revenue_over_time(
    interval: Interval = "month",
    frm: str | None = Query(None, alias="from"),
    to: str | None = Query(None),
) -> dict:
    """Completed revenue per period, with a cumulative total."""
    default_from, default_to = _default_dates()
    from_date = _parse_date(frm, default=default_from)
    to_date = _parse_date(to, default=default_to)

    try:
        payload = await go_client.get(
            "/analytics/revenue-over-time",
            params={"interval": interval, "from": from_date, "to": to_date},
        )
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    points = payload.get("points") or []
    amounts = [p["amount"] for p in points]

    cumulative: list[float] = []
    running = 0.0
    for a in amounts:
        running += a
        cumulative.append(round(running, 2))

    series = [
        {
            "period": p["period"],
            "amount": p["amount"],
            "count": p["count"],
            "cumulative": c,
        }
        for p, c in zip(points, cumulative, strict=True)
    ]

    return {
        "interval": interval,
        "from": from_date,
        "to": to_date,
        "points": series,
        "total": payload.get("total", 0),
        "total_count": payload.get("total_count", 0),
    }


@router.get("/score-distribution")
async def score_distribution() -> dict:
    """Score buckets plus percentile markers for the whole platform.

    Percentiles add context to the histogram: the frontend can show "median
    score", "top 10% start at X", and so on.
    """
    try:
        payload = await go_client.get("/analytics/score-distribution")
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    return {
        "buckets": payload.get("buckets", []),
        "total": payload.get("total", 0),
        "average": payload.get("average", 0),
        "median": payload.get("median", 0),
        "pass_count": payload.get("pass_count", 0),
        "fail_count": payload.get("fail_count", 0),
        "passing_pct": payload.get("passing_pct", 0),
    }