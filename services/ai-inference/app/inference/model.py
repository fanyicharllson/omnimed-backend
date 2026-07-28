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
        logits = np.asarray(outputs[0]).reshape(-1)

        if len(logits) != len(self._labels):
            raise ValueError(
                f"model produced {len(logits)} outputs but "
                f"{len(self._labels)} labels are configured"
            )

        probabilities = _softmax(logits)

        return {label: float(prob) for label, prob in zip(self._labels, probabilities)}


def _softmax(logits: np.ndarray) -> np.ndarray:
    # The exported graph's output is named "output_probabilities" but is
    # actually raw logits (PyTorch's CrossEntropyLoss applies softmax
    # internally during training, so the traced inference graph stops
    # short of it) — softmax here so confidences are a valid, non-negative
    # distribution that sums to 1.
    shifted = logits - np.max(logits)
    exp = np.exp(shifted)
    return exp / np.sum(exp)
