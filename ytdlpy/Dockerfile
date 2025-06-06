FROM python:3-alpine
WORKDIR /app

COPY requirement.txt .
RUN pip install -r requirement.txt

COPY app.py .
EXPOSE 5000
CMD ["python", "app.py"]
