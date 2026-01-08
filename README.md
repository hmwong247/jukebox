### Jukebox
___
A real-time webapp to listen to audio with peers online at low latency.

- This project uses WebRTC to achieve low latency and distribute bandwidth, it might not work depends on your browser, router or network policy at your institution.

- yt-dlp is embedded to a Python application to download the audio

### Usage
___
Generate SSL certs for nginx TLS termination, and put `fullchain.pem` and `key.pem` into `nginx/certs/`
```
├── nginx
│   ├── certs
│   │   ├── fullchain.pem
│   │   └── key.pem
│   └── conf.d
```

Install `docker` and `docker-compose`

Build from source and run:
```
docker compose up
```

Navigate to `https://<domain>/home` and enjoy

### test url
___
- https://youtu.be/oxzEdm29JLw
- https://www.youtube.com/watch?v=fDNNn9polrI
- https://www.youtube.com/watch?v=UxwTIJ03f2g
- https://www.youtube.com/watch?v=dQw4w9WgXcQ
- https://www.youtube.com/watch?v=06TlZMQf2UA

for testing quality
- https://www.youtube.com/watch?v=xWMbfvYZ_SU

age restricted
- https://www.youtube.com/watch?v=x8VYWazR5mE
