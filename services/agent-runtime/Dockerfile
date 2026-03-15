FROM python:3.12-slim
WORKDIR /app
ENV PYTHONUNBUFFERED=1 PYTHONPATH=/app/packages/py-common
RUN pip install --no-cache-dir fastapi uvicorn httpx pydantic
COPY packages/py-common /app/packages/py-common
COPY services/agent-runtime /app/services/agent-runtime
WORKDIR /app/services/agent-runtime
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]

