from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    analytics_app_name: str = "Rogele Analytics"
    analytics_debug: bool = False
    analytics_port: int = 9000

    rogele_api_url: str = "http://localhost:8080"
    rogele_service_token: str = ""

    http_timeout: int = 30


settings = Settings()