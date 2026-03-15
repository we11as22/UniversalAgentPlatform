FROM python:3.12-slim
WORKDIR /app
ENV PYTHONUNBUFFERED=1 PYTHONPATH=/app/packages/py-common
RUN pip install --no-cache-dir fastapi uvicorn pydantic
COPY packages/py-common /app/packages/py-common
COPY services/tool-runner /app/services/tool-runner
WORKDIR /app/services/tool-runner
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]

