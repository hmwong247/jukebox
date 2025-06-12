FROM python:3-alpine
WORKDIR /app

RUN apk update && apk add --no-cache ffmpeg
RUN rm -rf /var/cache/apk/*

COPY requirement.txt .
RUN pip install --no-cache-dir -r requirement.txt
RUN rm requirement.txt

COPY app.py .
CMD ["python", "app.py"]
