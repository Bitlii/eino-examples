#!/bin/bash

python -m vllm.entrypoints.openai.api_server \
    --model Qwen/Qwen3-Embedding-0.6B \
    --served-model-name Qwen3-Embedding-0.6B \
    --enforce-eager \
    --max-model-len 4096 \
    --port 8000 \
    --host 127.0.0.1 \
    --convert embed \
    --runner pooling