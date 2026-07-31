"""gRPC server exposing the TriageService contract."""

import logging
from concurrent import futures

import grpc
from grpc_health.v1 import health, health_pb2, health_pb2_grpc

from app.config import settings
from app.inference.model import BreastCancerModel
from app.inference.preprocess import ImageDecodeError, preprocess_image
from app.pb import triage_pb2, triage_pb2_grpc

logger = logging.getLogger(__name__)


class TriageServicer(triage_pb2_grpc.TriageServiceServicer):
    def __init__(self, model: BreastCancerModel, model_version: str):
        self._model = model
        self._model_version = model_version

    def DiagnoseBreastCancer(self, request, context):
        request_id = request.request_id or "unknown"

        if not request.image_data:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("image_data must not be empty")
            return triage_pb2.DiagnosisResponse()

        try:
            tensor = preprocess_image(request.image_data)
        except ImageDecodeError as exc:
            logger.warning(
                "image decode failed", extra={"request_id": request_id, "error": str(exc)}
            )
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details(str(exc))
            return triage_pb2.DiagnosisResponse()

        try:
            confidences = self._model.predict(tensor)
        except Exception as exc:
            logger.error(
                "inference failed", extra={"request_id": request_id, "error": str(exc)}
            )
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details("model inference failed")
            return triage_pb2.DiagnosisResponse()

        predicted_class = max(confidences, key=confidences.get)

        logger.info(
            "diagnosis completed",
            extra={
                "request_id": request_id,
                "predicted_class": predicted_class,
                "model_version": self._model_version,
            },
        )

        return triage_pb2.DiagnosisResponse(
            predicted_class=predicted_class,
            class_confidences=[
                triage_pb2.ClassConfidence(label=label, confidence=confidence)
                for label, confidence in confidences.items()
            ],
            model_version=self._model_version,
            request_id=request_id,
        )


def create_server(model: BreastCancerModel) -> grpc.Server:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    triage_pb2_grpc.add_TriageServiceServicer_to_server(
        TriageServicer(model, settings.model_version), server
    )

    # Standard gRPC health checking protocol (grpc.health.v1) — lets the
    # Go gateway (and later k8s probes) ask "is this service actually
    # ready" rather than just "does the port accept a TCP connection".
    health_servicer = health.HealthServicer()
    health_pb2_grpc.add_HealthServicer_to_server(health_servicer, server)
    # "" (empty service name) is the overall server health, checked by
    # callers that don't care about a specific service.
    health_servicer.set("", health_pb2.HealthCheckResponse.SERVING)
    health_servicer.set(
        "omnimed.triage.v1.TriageService", health_pb2.HealthCheckResponse.SERVING
    )

    server.add_insecure_port(f"{settings.grpc_host}:{settings.grpc_port}")
    return server
