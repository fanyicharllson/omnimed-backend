.PHONY: proto proto-go proto-python \
	build-gateway run-gateway \
	build-inference run-inference \
	docker-up docker-down docker-build \
	tidy

PROTO_DIR := api/proto
GO_MODULE := github.com/fanyicharllson/omnimed-backend
INFERENCE_DIR := services/ai-inference
INFERENCE_VENV := $(INFERENCE_DIR)/.venv

# Windows venvs place the interpreter under Scripts/, Unix under bin/.
ifeq ($(OS),Windows_NT)
	VENV_BIN := $(INFERENCE_VENV)/Scripts
else
	VENV_BIN := $(INFERENCE_VENV)/bin
endif

# --- Protobuf generation ---------------------------------------------

proto: proto-go proto-python

proto-go:
	protoc \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
		$(PROTO_DIR)/triage.proto

proto-python:
	$(VENV_BIN)/python -m grpc_tools.protoc \
		-I $(PROTO_DIR) \
		--python_out=$(INFERENCE_DIR)/app/pb \
		--grpc_python_out=$(INFERENCE_DIR)/app/pb \
		--pyi_out=$(INFERENCE_DIR)/app/pb \
		$(PROTO_DIR)/triage.proto
	# grpc_tools emits a bare "import triage_pb2", which breaks once
	# these stubs live inside the app.pb package — rewrite it to a
	# package-relative import.
	$(VENV_BIN)/python -c "import pathlib; p = pathlib.Path('$(INFERENCE_DIR)/app/pb/triage_pb2_grpc.py'); p.write_text(p.read_text().replace('import triage_pb2 as triage__pb2', 'from . import triage_pb2 as triage__pb2'))"

# --- Go gateway --------------------------------------------------------

tidy:
	go mod tidy

build-gateway:
	go build -o bin/gateway ./cmd/gateway

run-gateway: build-gateway
	./bin/gateway

# --- Python inference service -------------------------------------------

build-inference:
	python -m venv $(INFERENCE_VENV)
	$(VENV_BIN)/python -m pip install --upgrade pip
	$(VENV_BIN)/python -m pip install -r $(INFERENCE_DIR)/requirements.txt

run-inference:
	cd $(INFERENCE_DIR) && ../../$(VENV_BIN)/python -m app.main

# --- Docker Compose ------------------------------------------------------

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
