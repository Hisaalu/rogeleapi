import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from .routes import analytics, health

from . import __version__
from .config import settings
from .routes import health

logging.basicConfig(
    level=logging.DEBUG if settings.analytics_debug else logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info(
        "starting %s v%s (api=%s)",
        settings.analytics_app_name,
        __version__,
        settings.rogele_api_url,
    )
    yield
    logger.info("analytics service stopped")


app = FastAPI(
    title=settings.analytics_app_name,
    version=__version__,
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["*"],
)

app.include_router(health.router)
app.include_router(analytics.router)