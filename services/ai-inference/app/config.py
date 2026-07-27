"""Environment-driven configuration for the AI inference service."""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    environment: str = "development"
    log_level: str = "info"

    grpc_host: str = "0.0.0.0"
    grpc_port: int = 50051

    model_path: str = "/models/omnimed_breast_ultrasound.onnx"
    model_version: str = "omnimed-breast-ultrasound-v1.0.0"

    breast_cancer_labels: tuple[str, ...] = ("benign", "malignant", "normal")


settings = Settings()
