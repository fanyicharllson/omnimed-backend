"""ONNX Runtime wrapper for the breast cancer ultrasound classifier."""

import logging

import numpy as np
import onnxruntime as ort

logger = logging.getLogger(__name__)


class BreastCancerModel:
    """Loads the breast ultrasound ONNX model and runs inference."""

    def __init__(self, model_path: str, labels: tuple[str, ...]):
        self._labels = labels
        logger.info("loading onnx model", extra={"model_path": model_path})

        self._session = ort.InferenceSession(
            model_path, providers=["CPUExecutionProvider"]
        )
        self._input_name = self._session.get_inputs()[0].name
        self._output_name = self._session.get_outputs()[0].name

        logger.info(
            "onnx model loaded",
            extra={
                "input_name": self._input_name,
                "output_name": self._output_name,
                "labels": list(self._labels),
            },
        )

    def predict(self, input_tensor: np.ndarray) -> dict[str, float]:
        """Runs inference on a preprocessed (1, 3, 224, 224) tensor and
        returns a label -> confidence mapping.
        """
        outputs = self._session.run([self._output_name], {self._input_name: input_tensor})
        probabilities = np.asarray(outputs[0]).reshape(-1)

        if len(probabilities) != len(self._labels):
            raise ValueError(
                f"model produced {len(probabilities)} outputs but "
                f"{len(self._labels)} labels are configured"
            )

        return {label: float(prob) for label, prob in zip(self._labels, probabilities)}
