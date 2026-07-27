"""Entrypoint for the AI inference service: starts the gRPC TriageService
server and a small FastAPI app exposing a health check for
docker-compose / orchestrator probes.
"""

import logging
import threading

import uvicorn
from fastapi import FastAPI

from app.config import settings
from app.grpc_server import create_server
from app.inference.model import BreastCancerModel
from app.logging_config import configure_logging

configure_logging(settings.log_level)
logger = logging.getLogger(__name__)

api = FastAPI(title="OmniMed AI Inference Service")


@api.get("/healthz")
def healthz():
    return {"status": "ok", "model_version": settings.model_version}


def _serve_grpc() -> None:
    model = BreastCancerModel(settings.model_path, settings.breast_cancer_labels)
    server = create_server(model)
    server.start()
    logger.info(
        "grpc server listening",
        extra={"host": settings.grpc_host, "port": settings.grpc_port},
    )
    server.wait_for_termination()


def main() -> None:
    grpc_thread = threading.Thread(target=_serve_grpc, daemon=True, name="grpc-server")
    grpc_thread.start()

    uvicorn.run(api, host="0.0.0.0", port=8000, log_config=None)


if __name__ == "__main__":
    main()
