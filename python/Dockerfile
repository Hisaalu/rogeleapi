FROM python:3.15-slim

COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

WORKDIR /app

COPY pyproject.toml uv.lock README.md ./
RUN uv sync --frozen --no-dev

COPY src ./src

ENV PYTHONPATH=/app/src
ENV PATH="/app/.venv/bin:$PATH"

EXPOSE 9000

CMD ["uvicorn", "rogeledb_analytics.main:app", "--host", "0.0.0.0", "--port", "9000"]