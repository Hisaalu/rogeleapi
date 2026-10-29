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
    signup_df["created_at"] = pd.to_datetime(signup_df["created_at"], utc=True, format="ISO8601").dt.tz_localize(None)
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
    events_df["created_at"] = pd.to_datetime(events_df["created_at"], utc=True, format="ISO8601").dt.tz_localize(None)
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

    grid: list[list[int]] = [[0] * 24 for _ in range(7)]

    if not isinstance(events, list) or not events:
        return {"grid": grid, "max": 0, "total": 0}

    df = pd.DataFrame(events)
    df["created_at"] = pd.to_datetime(df["created_at"], utc=True, format="ISO8601")
    df["dow"] = df["created_at"].dt.dayofweek  
    df["hour"] = df["created_at"].dt.hour

    counts = df.groupby(["dow", "hour"]).size()

    total = 0
    max_value = 0

    for (dow, hour), count in counts.items():
        d = int(dow)
        h = int(hour)
        c = int(count)
        if 0 <= d < 7 and 0 <= h < 24:
            grid[d][h] = c
            total += c
            if c > max_value:
                max_value = c

    return {"grid": grid, "max": max_value, "total": total}


@router.get("/at-risk-learners")
async def at_risk_learners(
    min_score: int = Query(3, ge=0, le=20),
    limit: int = Query(50, ge=1, le=200),
) -> dict:
    try:
        signals = await go_client.get("/analytics/learner-signals")
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    if not isinstance(signals, list) or not signals:
        return {"learners": [], "summary": {"at_risk": 0, "watch": 0, "healthy": 0, "total": 0}}

    learners: list[dict] = []
    at_risk = watch = healthy = 0

    for sig in signals:
        score = 0
        reasons: list[str] = []

        days = sig.get("days_since_last_activity")
        if days is None:
            score += 3
            reasons.append("No recorded activity")
        elif days >= 14:
            score += 3
            reasons.append(f"No activity in {days} days")
        elif days >= 7:
            score += 2
            reasons.append(f"No activity in {days} days")
        elif days >= 3:
            score += 1
            reasons.append(f"No activity in {days} days")

        attempts = sig.get("quiz_attempts", 0)
        quiz_avg = sig.get("quiz_average")
        if attempts > 0 and quiz_avg is not None:
            if quiz_avg < 50:
                score += 2
                reasons.append(f"Quiz average {quiz_avg:.0f}%")
            elif quiz_avg < 65:
                score += 1
                reasons.append(f"Quiz average {quiz_avg:.0f}%")

        missing = sig.get("missing_assignments", 0)
        if missing >= 3:
            score += 3
            reasons.append(f"{missing} missing assignments")
        elif missing >= 1:
            score += 1
            reasons.append(f"{missing} missing assignment{'s' if missing > 1 else ''}")

        graded = sig.get("homework_graded", 0)
        hw_avg = sig.get("homework_average_grade")
        if graded > 0 and hw_avg is not None:
            if hw_avg < 50:
                score += 2
                reasons.append(f"Homework average {hw_avg:.0f}%")
            elif hw_avg < 65:
                score += 1
                reasons.append(f"Homework average {hw_avg:.0f}%")

        if score >= 6:
            band = "at_risk"
            at_risk += 1
        elif score >= 3:
            band = "watch"
            watch += 1
        else:
            band = "healthy"
            healthy += 1

        learners.append(
            {
                "user_id": sig.get("user_id"),
                "email": sig.get("email"),
                "first_name": sig.get("first_name"),
                "last_name": sig.get("last_name"),
                "class_id": sig.get("class_id"),
                "class_name": sig.get("class_name"),
                "last_activity": sig.get("last_activity"),
                "days_since_last_activity": days,
                "quiz_attempts": attempts,
                "quiz_average": quiz_avg,
                "homework_submitted": sig.get("homework_submitted", 0),
                "homework_graded": graded,
                "homework_average_grade": hw_avg,
                "missing_assignments": missing,
                "risk_score": score,
                "risk_band": band,
                "reasons": reasons,
            }
        )

    learners.sort(key=lambda item: item["risk_score"], reverse=True)
    learners = [item for item in learners if item["risk_score"] >= min_score][:limit]

    return {
        "learners": learners,
        "summary": {
            "at_risk": at_risk,
            "watch": watch,
            "healthy": healthy,
            "total": len(signals),
        },
    }


@router.get("/revenue-forecast")
async def revenue_forecast(
    months_ahead: int = Query(3, ge=1, le=12),
) -> dict:
    try:
        payload = await go_client.get(
            "/analytics/revenue-over-time",
            params={"interval": "month"},
        )
    except GoAPIError as exc:
        raise HTTPException(status_code=exc.status_code, detail=exc.message) from exc

    points = payload.get("points") or []

    if len(points) < 2:
        return {
            "history": points,
            "forecast": [],
            "trend": {"slope": 0, "intercept": 0},
            "note": "Not enough historical data to forecast. At least two months required.",
        }

    x = np.arange(len(points), dtype=float)
    y = np.array([p["amount"] for p in points], dtype=float)

    slope, intercept = np.polyfit(x, y, 1)

    forecast = []
    last_period = pd.Timestamp(points[-1]["period"]).to_pydatetime()

    for i in range(1, months_ahead + 1):
        next_period = last_period + pd.DateOffset(months=i)
        projected = slope * (len(points) - 1 + i) + intercept
        projected = max(projected, 0)
        forecast.append(
            {
                "period": next_period.isoformat(),
                "amount": round(float(projected), 2),
            }
        )

    return {
        "history": points,
        "forecast": forecast,
        "trend": {
            "slope": round(float(slope), 2),
            "intercept": round(float(intercept), 2),
        },
    }