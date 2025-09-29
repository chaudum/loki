# Logs Lake

## Getting started

The docker-compose setup contains Loki, MinIO, and the Iceberg REST API.

```bash
cd development
docker-compose build
docker-compose up
```

### Install the Iceberg CLI

```bash
go install github.com/apache/iceberg-go/cmd/iceberg@latest
```

### Setup Catalog

The bootstrap.py script requires the `pyiceberg` and `pyarrow` modules. Install them via `pip` in a virtualenv first.

```bash
python -m venv env
source env/bin/activate
env/bin/python -m pip install pyiceberg pyarrow
```

```bash
env/bin/python bootstrap.py
iceberg --uri http://localhost:8181 describe table loki.chunks
```
