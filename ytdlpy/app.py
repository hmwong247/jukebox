import io
import json
import os
import socket
import threading
from concurrent.futures import ThreadPoolExecutor

import yt_dlp

# environment
MAX_CONCURRENT_DL = int(os.environ.get('YTDLPY_MAX_CONCURRENT_DL', 1))
SOCKET_DIR = os.environ.get('YTDLPY_SOCKET_DIR', '')
SOCKET_PATH = os.environ.get('YTDLPY_SOCKET_PATH', '')
print(f'\nDEBUG: YTDLPY ENVIRONMENT VARIABLE')
print(f'YTDLPY_MAX_CONCURRENT_DL: {MAX_CONCURRENT_DL}')
print(f'YTDLPY_SOCKET_DIR: {SOCKET_DIR}')
print(f'YTDLPY_SOCKET_PATH: {SOCKET_PATH}')


def dl_infojson(url: str) -> dict:
    def extractKeys(entry: dict) -> dict:
        info_keys = [
            'fulltitle',
            'uploader',
            'thumbnail',
            'duration',
        ]
        json = {}
        for key in info_keys:
            json[key] = entry.get(key)
        
        return json

    ydl_opts = {
        'quiet': True,
    }
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        infojson = ydl.extract_info(url, download=False)
        infojson = ydl.sanitize_info(infojson)

        if infojson.get('_type') == 'playlist':
            ret = [extractKeys(entry) for entry in infojson.get('entries')]
        else:
            ret = extractKeys(infojson)

    return ret


def dl_audio(url: str):
    buffer = io.BytesIO()
    done_event = threading.Event()

    def callback(d: dict):
        if(d.get('status') == "finished"):
            filepath = d.get('filename', '')
            with open(filepath, 'rb') as f:
                buffer.write(f.read())
            os.remove(filepath)
            done_event.set()

    ydl_opts = {
        'format': 'bestaudio',
        'restrictfilenames': True,
        'outtmpl': f'{SOCKET_DIR}/%(timestamp)s.%(id)s.%(fulltitle)s.%(ext)s',
        'progress_hooks': [callback],
        'quiet': True,
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        ydl.download(url)

    done_event.wait()
    return buffer


def handle_socket(conn: socket.socket):
    buf = conn.recv(1024)
    data = json.loads(buf)
    request = data.get('Type')
    url = data.get('URL')
    resp = ''
    match request:
        case 'json':
            resp = dl_infojson(url)
            conn.sendall(json.dumps(resp).encode())
        case 'audio':
            resp = dl_audio(url)
            conn.sendall(resp.getvalue())
        case _:
            pass

    conn.close()


def uds_server():
    tpool = ThreadPoolExecutor(max_workers=MAX_CONCURRENT_DL)
    try:
        if os.path.exists(SOCKET_PATH):
            os.remove(SOCKET_PATH)

        os.makedirs(SOCKET_DIR, exist_ok=True)
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
            s.bind(SOCKET_PATH)
            s.listen(MAX_CONCURRENT_DL)

            while True:
                conn, _ = s.accept()
                tpool.submit(handle_socket, conn)
    except KeyboardInterrupt:
        print('\nstopping ytdlpy')
        tpool.shutdown(wait=False, cancel_futures=True)


# main
if __name__ == '__main__':
    print('running ytdlpy')
    uds_server()


