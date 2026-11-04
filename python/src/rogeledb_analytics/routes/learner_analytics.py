import logging
from datetime import datetime, timedelta
from typing import Literal

import numpy as np
import pandas as pd
from cachetools import TTLCache
from fastapi import APIRouter, HTTPException, Query

from ..go_client import GoAPIError, go_client

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/analytics/learner", tags=["learner-analytics"])

learner_cache: TTLCache = TTLCache(maxsize=1000, ttl=60)
name_cache: TTLCache = TTLCache(maxsize=2000, ttl=300)

Interval = Literal["day", "week"]


def _parse_date(value: str | None, *, default: str) -> str:
    if not value:
        return default
    try:
        datetime.strptime(value, "%Y-%m-%d")
    except ValueError:
        raise HTTPException(400, f"date must be YYYY-MM-DD, got {value}") from None
    return value


def _default_range() -> tuple[str, str]:
    to = datetime.utcnow()
    frm = to - timedelta(days=30)
    return frm.strftime("%Y-%m-%d"), to.strftime("%Y-%m-%d")


async def _learner_first_name(learner_id: int) -> str | None:
    if learner_id in name_cache:
        return name_cache[learner_id]

    try:
        profile = await go_client.get(f"/admin/users/{learner_id}/details")
    except GoAPIError:
        name_cache[learner_id] = None
        return None

    user = (profile or {}).get("user") if isinstance(profile, dict) else None
    first = (user or {}).get("first_name") if isinstance(user, dict) else None
    first = str(first).strip() if first else None
    name_cache[learner_id] = first or None
    return first or None


def _prefix(first_name: str | None) -> str:
    if first_name:
        return f"{first_name}, "
    return ""


def _recommendations(
    hw_df: pd.DataFrame,
    quiz_df: pd.DataFrame,
    daily_activity: pd.DataFrame,
    first_name: str | None,
) -> list[dict]:
    recs: list[dict] = []
    pre = _prefix(first_name)

    if not quiz_df.empty:
        by_subject = (
            quiz_df.dropna(subset=["subject_name", "score"])
            .groupby("subject_name")["score"]
            .agg(["mean", "count"])
            .reset_index()
        )
        weak = by_subject[(by_subject["mean"] < 60) & (by_subject["count"] >= 2)]
        for _, row in weak.iterrows():
            recs.append(
                {
                    "tone": "warning",
                    "title": f"Struggling in {row['subject_name']}",
                    "body": (
                        f"{pre}your average quiz score in {row['subject_name']} is "
                        f"{row['mean']:.0f}%. Consider reviewing recent lessons."
                    ),
                }
            )

    if not hw_df.empty:
        graded = hw_df.dropna(subset=["grade"]).sort_values("submitted_at")
        if len(graded) >= 4:
            mid = len(graded) // 2
            early_avg = graded.iloc[:mid]["grade"].mean()
            recent_avg = graded.iloc[mid:]["grade"].mean()
            if recent_avg > early_avg + 5:
                recs.append(
                    {
                        "tone": "success",
                        "title": "You're improving!",
                        "body": (
                            f"{pre}your recent homework average is {recent_avg:.0f}%, "
                            f"up from {early_avg:.0f}%. Keep it up!"
                        ),
                    }
                )
            elif recent_avg < early_avg - 5:
                recs.append(
                    {
                        "tone": "warning",
                        "title": "Homework grades slipped",
                        "body": (
                            f"{pre}your recent homework average is {recent_avg:.0f}%, "
                            f"down from {early_avg:.0f}%. Consider revisiting recent topics."
                        ),
                    }
                )

    if not daily_activity.empty:
        active_days = len(daily_activity)
        if active_days <= 3:
            recs.append(
                {
                    "tone": "warning",
                    "title": "Low activity this period",
                    "body": (
                        f"{pre}you were only active on {active_days} day"
                        f"{'s' if active_days != 1 else ''}. "
                        "Try a small daily study habit."
                    ),
                }
            )
        elif active_days >= 15:
            recs.append(
                {
                    "tone": "success",
                    "title": "Great consistency",
                    "body": f"{pre}you were active on {active_days} days. That consistency pays off.",
                }
            )

    if not daily_activity.empty and "hour" in daily_activity.columns:
        peak_hour = (
            int(daily_activity["hour"].mode().iloc[0])
            if not daily_activity["hour"].empty
            else None
        )
        if peak_hour is not None:
            label = f"{peak_hour:02d}:00–{peak_hour + 1:02d}:00"
            recs.append(
                {
                    "tone": "info",
                    "title": "Best time to study",
                    "body": f"{pre}most of your work happens around {label}. Block that time for study.",
                }
            )

    return recs[:6]


@router.get("/overview")
async def learner_overview(
    learner_id: int = Query(..., description="The learner's user ID"),
    frm: str | None = Query(None, alias="from"),
    to: str | None = Query(None),
    interval: Interval = "day",
) -> dict:
    default_from, default_to = _default_range()
    from_date = _parse_date(frm, default=default_from)
    to_date = _parse_date(to, default=default_to)

    cache_key = f"{learner_id}:{from_date}:{to_date}:{interval}"
    if cache_key in learner_cache:
        return learner_cache[cache_key]

    try:
        raw = await go_client.get(
            f"/analytics/learner/raw/{learner_id}",
            params={"from": from_date, "to": to_date},
        )
    except GoAPIError as exc:
        raise HTTPException(
            exc.status_code,
            exc.detail if hasattr(exc, "detail") else exc.message,
        ) from exc

    first_name = await _learner_first_name(learner_id)

    homeworks = raw.get("homeworks") or []
    quizzes = raw.get("quiz_attempts") or []

    hw_df = pd.DataFrame(homeworks)
    quiz_df = pd.DataFrame(quizzes)

    hw_submitted = len(hw_df)
    hw_graded = hw_df.dropna(subset=["grade"]) if "grade" in hw_df.columns else pd.DataFrame()
    hw_grades = hw_graded["grade"].tolist() if not hw_graded.empty else []

    hw_kpis = {
        "submitted": hw_submitted,
        "graded": len(hw_grades),
        "average": round(float(np.mean(hw_grades)), 2) if hw_grades else None,
        "highest": round(float(np.max(hw_grades)), 2) if hw_grades else None,
        "lowest": round(float(np.min(hw_grades)), 2) if hw_grades else None,
    }

    quiz_scores = quiz_df["score"].dropna().tolist() if not quiz_df.empty else []
    quiz_kpis = {
        "attempted": len(quiz_df),
        "average": round(float(np.mean(quiz_scores)), 2) if quiz_scores else None,
        "highest": round(float(np.max(quiz_scores)), 2) if quiz_scores else None,
        "lowest": round(float(np.min(quiz_scores)), 2) if quiz_scores else None,
    }

    activity_frames = []
    if not hw_df.empty and "submitted_at" in hw_df.columns:
        hw_ts = pd.to_datetime(
            hw_df["submitted_at"].dropna(), utc=True, format="ISO8601"
        )
        activity_frames.append(pd.DataFrame({"ts": hw_ts, "kind": "homework"}))
    if not quiz_df.empty and "completed_at" in quiz_df.columns:
        q_ts = pd.to_datetime(
            quiz_df["completed_at"].dropna(), utc=True, format="ISO8601"
        )
        activity_frames.append(pd.DataFrame({"ts": q_ts, "kind": "quiz"}))

    if activity_frames:
        activity_df = pd.concat(activity_frames, ignore_index=True)
        activity_df["date"] = activity_df["ts"].dt.strftime("%Y-%m-%d")
        activity_df["dow"] = activity_df["ts"].dt.dayofweek
        activity_df["hour"] = activity_df["ts"].dt.hour
        daily = activity_df.groupby("date").size().reset_index(name="count")
    else:
        activity_df = pd.DataFrame(columns=["ts", "kind", "date", "dow", "hour"])
        daily = pd.DataFrame(columns=["date", "count"])

    grid = [[0] * 24 for _ in range(7)]
    peak = 0
    for _, row in activity_df.iterrows():
        d, h = int(row["dow"]), int(row["hour"])
        grid[d][h] += 1
        peak = max(peak, grid[d][h])

    if not daily.empty:
        daily_dates = pd.to_datetime(daily["date"])
        daily["week"] = daily_dates.dt.to_period("W").dt.start_time.dt.strftime("%Y-%m-%d")
        weekly = (
            daily.groupby("week")["count"]
            .sum()
            .reset_index()
            .rename(columns={"count": "activities"})
        )
        weekly_series = weekly.to_dict(orient="records")
    else:
        weekly_series = []

    hw_trend = []
    if not hw_graded.empty:
        hw_graded = hw_graded.copy()
        hw_graded["date"] = pd.to_datetime(
            hw_graded["submitted_at"], utc=True, format="ISO8601"
        ).dt.strftime("%Y-%m-%d")
        grouped = hw_graded.groupby("date")["grade"].mean().reset_index()
        hw_trend = [
            {"date": row["date"], "value": round(float(row["grade"]), 2)}
            for _, row in grouped.iterrows()
        ]

    quiz_trend = []
    if not quiz_df.empty:
        q = quiz_df.dropna(subset=["score", "completed_at"]).copy()
        q["date"] = pd.to_datetime(
            q["completed_at"], utc=True, format="ISO8601"
        ).dt.strftime("%Y-%m-%d")
        grouped = q.groupby("date")["score"].mean().reset_index()
        quiz_trend = [
            {"date": row["date"], "value": round(float(row["score"]), 2)}
            for _, row in grouped.iterrows()
        ]

    recommendations = _recommendations(hw_df, quiz_df, activity_df, first_name)

    result = {
        "learner_id": learner_id,
        "from": from_date,
        "to": to_date,
        "interval": interval,
        "homework": hw_kpis,
        "quizzes": quiz_kpis,
        "activity": {
            "heatmap": grid,
            "heatmap_max": peak,
            "daily_counts": daily.to_dict(orient="records"),
            "weekly_counts": weekly_series,
            "active_days": int(len(daily)),
        },
        "trends": {
            "homework": hw_trend,
            "quizzes": quiz_trend,
        },
        "recommendations": recommendations,
    }

    learner_cache[cache_key] = result
    return result