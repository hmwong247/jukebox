import io
import json
import os
import sys
import socket
import threading
from concurrent.futures import ThreadPoolExecutor

import yt_dlp

"""
This python application is purely for embedding yt-dlp and socket communication

The YTDLP options can be found on: https://github.com/yt-dlp/yt-dlp/blob/master/yt_dlp/YoutubeDL.py#183
"""

# environment
MAX_CONCURRENT_DL = int(os.environ.get('YTDLPY_MAX_CONCURRENT_DL', 1))
SOCKET_DIR = os.environ.get('YTDLPY_SOCKET_DIR', '')
SOCKET_PATH = os.environ.get('YTDLPY_SOCKET_PATH', '')


def print_env():
    print(f'\nDEBUG: YTDLPY ENVIRONMENT VARIABLE')
    print(f'YTDLPY_MAX_CONCURRENT_DL: {MAX_CONCURRENT_DL}')
    print(f'YTDLPY_SOCKET_DIR: {SOCKET_DIR}')
    print(f'YTDLPY_SOCKET_PATH: {SOCKET_PATH}')


# const
AUDIO_CODEC = 'm4a'


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
    ret = {}
    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            infojson = ydl.extract_info(url, download=False)
            infojson = ydl.sanitize_info(infojson)

            if infojson.get('_type') == 'playlist':
                ret = [extractKeys(entry) for entry in infojson.get('entries')]
            else:
                ret = extractKeys(infojson)
    except Exception as e:
        ret = {'Err': f'{e}'}

    return ret


def dl_audio(url: str):
    buffer = io.BytesIO()
    filepath = ''
    event_fin = threading.Event()

    def hook(d: dict):
        if(d.get('status') == 'finished'):
            nonlocal filepath
            filepath = d.get('filename', '')
            print(f'downloaded size: {d.get('downloaded_bytes')}')

    def pp_hook(d: dict):
        # print(f'pp_hook, d: {d.get('status')}, pp name: {d.get('postprocessor')}')
        if(d.get('postprocessor') == 'MoveFiles' and d.get('status') == 'finished'):
            audio_filepath = f'{os.path.splitext(filepath)[0]}.{AUDIO_CODEC}'
            try:
                with open(audio_filepath, 'rb') as f:
                    buffer.write(f.read())

                # original file is removed automatically by yt-dlp, we only need to remove the extracted file
                os.remove(audio_filepath)
            except OSError as err:
                print(err)
            finally:
                event_fin.set()

    """
    extractor_args is a workaround solution for HTTP 403 when fetching HLS fragments

    https://github.com/yt-dlp/yt-dlp/issues/13511#issuecomment-2993001328
    """
    ydl_opts = {
        'format': 'worstaudio',
        'extractor_args': {
            'youtube': {
                'player_client': ['default','-ios'],
            },
        },
        'restrictfilenames': True,
        'outtmpl': f'{SOCKET_DIR}/%(timestamp)s.%(id)s.%(fulltitle)s.%(ext)s',
        'postprocessors': [{
            'key': 'FFmpegExtractAudio',
            'preferredcodec': AUDIO_CODEC,
        }],
        'progress_hooks': [hook],
        'postprocessor_hooks': [pp_hook],
        # 'quiet': True,
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        ydl.download(url)

    event_fin.wait()
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


def auto_update():
    import subprocess
    print('updating yt-dlp')
    subprocess.check_call([sys.executable, '-m', 'pip', 'install', '--upgrade', 'yt-dlp'])


# main
if __name__ == '__main__':
    auto_update()
    print('running ytdlpy')
    print_env()
    sys.stdout.flush()
    uds_server()


