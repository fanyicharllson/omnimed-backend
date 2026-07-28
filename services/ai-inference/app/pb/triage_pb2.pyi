from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Mapping as _Mapping, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ImageRequest(_message.Message):
    __slots__ = ("image_data", "content_type", "request_id")
    IMAGE_DATA_FIELD_NUMBER: _ClassVar[int]
    CONTENT_TYPE_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    image_data: bytes
    content_type: str
    request_id: str
    def __init__(self, image_data: _Optional[bytes] = ..., content_type: _Optional[str] = ..., request_id: _Optional[str] = ...) -> None: ...

class ClassConfidence(_message.Message):
    __slots__ = ("label", "confidence")
    LABEL_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_FIELD_NUMBER: _ClassVar[int]
    label: str
    confidence: float
    def __init__(self, label: _Optional[str] = ..., confidence: _Optional[float] = ...) -> None: ...

class DiagnosisResponse(_message.Message):
    __slots__ = ("predicted_class", "class_confidences", "model_version", "request_id")
    PREDICTED_CLASS_FIELD_NUMBER: _ClassVar[int]
    CLASS_CONFIDENCES_FIELD_NUMBER: _ClassVar[int]
    MODEL_VERSION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_ID_FIELD_NUMBER: _ClassVar[int]
    predicted_class: str
    class_confidences: _containers.RepeatedCompositeFieldContainer[ClassConfidence]
    model_version: str
    request_id: str
    def __init__(self, predicted_class: _Optional[str] = ..., class_confidences: _Optional[_Iterable[_Union[ClassConfidence, _Mapping]]] = ..., model_version: _Optional[str] = ..., request_id: _Optional[str] = ...) -> None: ...
