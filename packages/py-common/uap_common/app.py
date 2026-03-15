from fastapi import FastAPI

from .config import AppConfig


def create_app(config: AppConfig) -> FastAPI:
    app = FastAPI(title=config.service_name, version=config.version)

    @app.get("/api/health")
    async def health() -> dict[str, str]:
        return {"service": config.service_name, "status": "ok", "version": config.version}

    @app.get("/health/ready")
    async def ready() -> dict[str, str]:
        return {"status": "ok"}

    return app

