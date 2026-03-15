FROM python:3.12-slim

WORKDIR /bundle
COPY examples/rag-agent-bundle /bundle

ENV PYTHONUNBUFFERED=1
ENV ADMIN_API_URL=http://admin-api:8080

ENTRYPOINT ["python", "/bundle/install.py"]
