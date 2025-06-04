FROM python:3-alpine
WORKDIR /flask_app

COPY requirement.txt .
RUN pip install -r requirement.txt

COPY . .
EXPOSE 5000
CMD ["python", "-m", "flask", "run", "--host=0.0.0.0"]
