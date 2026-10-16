import logging
from datetime import datetime, timedelta
from typing import Literal

import numpy as np
import pandas as pd
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

@router.get("/retention")
async def retention(
    frm: str | None = Query(None, alias="from"),
    to: str | None = Query(None),
    weeks: int = Query(8, ge=1, le=52),
) -> dict:
    """Weekly signup cohort retention."""
    default_from, default_to = _default_dates()
    from_date = _parse_date(frm, default=default_from)
    to_date = _parse_date(to, default=default_to)

    try:
        signups = await go_client.get("/analytics/signup-dates")
        events = await go_client.get(
            "/analytics/user-events",
            params={"from": from_date, "to": to_date, "limit": 50000},
        )
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    if not isinstance(signups, list) or not signups:
        return {"cohorts": [], "weeks": weeks}

    signup_df = pd.DataFrame(signups)
    signup_df["created_at"] = pd.to_datetime(signup_df["created_at"], utc=True)
    signup_df["cohort_week"] = signup_df["created_at"].dt.to_period("W").dt.start_time

    if not isinstance(events, list) or not events:
        recent_cohorts = (
            signup_df["cohort_week"]
            .dropna()
            .sort_values(ascending=False)
            .unique()[:weeks]
        )
        cohorts = []
        for cohort_week in recent_cohorts:
            cohort_size = int((signup_df["cohort_week"] == cohort_week).sum())
            row = [
                {"week": offset, "active": 0, "pct": 0.0}
                for offset in range(weeks)
            ]
            cohorts.append(
                {
                    "cohort_week": pd.Timestamp(cohort_week).isoformat(),
                    "cohort_size": cohort_size,
                    "weeks": row,
                }
            )
        return {"cohorts": cohorts, "weeks": weeks}

    events_df = pd.DataFrame(events)
    events_df["created_at"] = pd.to_datetime(events_df["created_at"], utc=True)
    events_df["event_week"] = events_df["created_at"].dt.to_period("W").dt.start_time

    activity = events_df[["user_id", "event_week"]].drop_duplicates()

    merged = signup_df.merge(activity, on="user_id", how="inner")

    merged["week_offset"] = (
        (merged["event_week"] - merged["cohort_week"]).dt.days // 7
    ).astype("Int64")

    merged = merged[merged["week_offset"].between(0, weeks - 1)]

    recent_cohorts = (
        signup_df["cohort_week"]
        .dropna()
        .sort_values(ascending=False)
        .unique()[:weeks]
    )

    cohorts = []
    for cohort_week in recent_cohorts:
        cohort_size = int((signup_df["cohort_week"] == cohort_week).sum())
        if cohort_size == 0:
            continue

        cohort_events = merged[merged["cohort_week"] == cohort_week]
        row = []
        for offset in range(weeks):
            active = int(
                cohort_events[cohort_events["week_offset"] == offset]["user_id"].nunique()
            )
            pct = active / cohort_size
            row.append({"week": int(offset), "active": active, "pct": round(pct, 4)})

        cohorts.append(
            {
                "cohort_week": pd.Timestamp(cohort_week).isoformat(),
                "cohort_size": cohort_size,
                "weeks": row,
            }
        )

    return {"cohorts": cohorts, "weeks": weeks}


@router.get("/activity-heatmap")
async def activity_heatmap(
    frm: str | None = Query(None, alias="from"),
    to: str | None = Query(None),
) -> dict:
    """Day-of-week × hour-of-day activity grid.

    Values are event counts. The frontend renders this as a heatmap.
    """
    default_from, default_to = _default_dates()
    from_date = _parse_date(frm, default=default_from)
    to_date = _parse_date(to, default=default_to)

    try:
        events = await go_client.get(
            "/analytics/user-events",
            params={"from": from_date, "to": to_date, "limit": 50000},
        )
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    if not isinstance(events, list) or not events:
        return {
            "grid": [[0] * 24 for _ in range(7)],
            "max": 0,
            "total": 0,
        }

    df = pd.DataFrame(events)
    df["created_at"] = pd.to_datetime(df["created_at"], utc=True)
    df["dow"] = df["created_at"].dt.dayofweek 
    df["hour"] = df["created_at"].dt.hour

    pivot = df.groupby(["dow", "hour"]).size().unstack(fill_value=0)

    grid: list[list[int]] = []
    for d in range(7):
        row = []
        for h in range(24):
            row.append(int(pivot.iloc[d][h]) if d in pivot.index and h in pivot.columns else 0)
        grid.append(row)

    flat = [v for row in grid for v in row]
    return {
        "grid": grid,
        "max": max(flat) if flat else 0,
        "total": sum(flat),
    }