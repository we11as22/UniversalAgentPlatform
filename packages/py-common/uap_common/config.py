from pydantic import BaseModel


class AppConfig(BaseModel):
    service_name: str
    version: str = "0.1.0"
    database_url: str = "postgresql://uap:uap@localhost:5432/uap"

