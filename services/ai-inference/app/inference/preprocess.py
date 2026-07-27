"""Image preprocessing matching the model's training pipeline:
resize to 224x224 and normalize with ImageNet mean/std.
"""

import io

import numpy as np
from PIL import Image

_TARGET_SIZE = (224, 224)
_IMAGENET_MEAN = np.array([0.485, 0.456, 0.406], dtype=np.float32)
_IMAGENET_STD = np.array([0.229, 0.224, 0.225], dtype=np.float32)


class ImageDecodeError(ValueError):
    """Raised when the uploaded bytes cannot be decoded as an image."""


def preprocess_image(image_bytes: bytes) -> np.ndarray:
    """Decode raw image bytes into a normalized NCHW float32 tensor of
    shape (1, 3, 224, 224), matching the ONNX model's expected input.
    """
    try:
        image = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    except Exception as exc:
        raise ImageDecodeError(f"could not decode image: {exc}") from exc

    image = image.resize(_TARGET_SIZE, resample=Image.BILINEAR)

    array = np.asarray(image, dtype=np.float32) / 255.0
    array = (array - _IMAGENET_MEAN) / _IMAGENET_STD

    # HWC -> CHW, then add batch dimension.
    array = array.transpose(2, 0, 1)
    array = np.expand_dims(array, axis=0)

    return np.ascontiguousarray(array, dtype=np.float32)
