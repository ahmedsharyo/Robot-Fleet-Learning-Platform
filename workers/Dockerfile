FROM python:3.11-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ libgl1 libosmesa6 libglfw3 && rm -rf /var/lib/apt/lists/*
COPY requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY datasets ./datasets
COPY workers ./workers
COPY learning ./learning
COPY benchmarks ./benchmarks
ENV PYTHONPATH=/app PYTHONUNBUFFERED=1
CMD ["python", "-m", "workers.simulation.worker"]
