FROM python:3.12-slim
WORKDIR /app
ENV PYTHONUNBUFFERED=1 PYTHONPATH=/app/packages/py-common
RUN apt-get update \
  && apt-get install -y --no-install-recommends curl ca-certificates tar \
  && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://github.com/grafana/k6/releases/download/v1.1.0/k6-v1.1.0-linux-amd64.tar.gz | tar -xz --strip-components=1 -C /tmp k6-v1.1.0-linux-amd64/k6 \
  && install -m 0755 /tmp/k6 /usr/local/bin/k6
RUN pip install --no-cache-dir fastapi uvicorn asyncpg httpx
COPY packages/py-common /app/packages/py-common
COPY perf /app/perf
COPY services/workflow-workers /app/services/workflow-workers
WORKDIR /app/services/workflow-workers
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
