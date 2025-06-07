import io
import json
import os
import socket
from concurrent.futures import ThreadPoolExecutor
import yt_dlp


# environment
URL_LIST = ['https://www.youtube.com/watch?v=q6EoRBvdVPQ']
URL_PLAYLIST = 'https://www.youtube.com/playlist?list=PLTMzl6sYFj5mgFDqdJv7zt_oRxY0LkVcx'
MAX_SOCKET = 4
SOCKET_DIR = os.environ.get('YTDLPY_SOCKET_DIR', '')
print(f'YTDLPY SOCKET_DIR: {SOCKET_DIR}')
SOCKET_PATH = os.environ.get('YTDLPY_SOCKET_PATH', '')
print(f'YTDLPY SOCKET_PATH: {SOCKET_PATH}')


def dl_infojson(url: str):
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

    ydl_opts = {}
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        infojson = ydl.extract_info(url, download=False)
        infojson = ydl.sanitize_info(infojson)

        if infojson.get('_type') == 'playlist':
            ret = [extractKeys(entry) for entry in infojson.get('entries')]
        else:
            ret = extractKeys(infojson)

    return ret


def dl_audio(urls):
    buffer = io.BytesIO()
    def hook(d):
        if(d.get('status') == "finished"):
            buffer.write(d.get('downloaded_bytes', b''))

    ydl_opts = {
        'format': 'bestaudio',
        'outtmpl': 'tmp/',
        'progress_hooks': [hook],
        'quiet': True,
    }

    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        ydl.download(urls)


# async def read_message(reader):



# async def send_message(writer, data):



# async def handle_socket(reader, writer):
#     msg = await reader.read() # read until EOF
#     data = json.loads(msg)
#     request = data.get('Type')
#     url = data.get('URL')
#     resp = ''
#     match request:
#         case 'json':
#             resp = dl_infojson(url)
#         case 'audio':
#             pass
#         case _:
#             pass
#
#     writer.write(json.dumps(resp).encode())
#     writer.write_eof()
#     await writer.drain()
#
#     writer.close()
#     await writer.wait_closed()


# async def uds_server():
#     server = await asyncio.start_unix_server(handle_socket, path=SOCKET_PATH)
#     async with server:
#         await server.serve_forever()


def handle_socket2(conn: socket.socket):
    buf = conn.recv(1024)
    data = json.loads(buf)
    request = data.get('Type')
    url = data.get('URL')
    resp = ''
    match request:
        case 'json':
            resp = dl_infojson(url)
        case 'audio':
            pass
        case _:
            pass

    conn.sendall(json.dumps(resp).encode())
    conn.close()


def uds_server2():
    tpool = ThreadPoolExecutor(max_workers=MAX_SOCKET)
    try:
        if os.path.exists(SOCKET_PATH):
            os.remove(SOCKET_PATH)

        os.makedirs(SOCKET_DIR, exist_ok=True)
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
            s.bind(SOCKET_PATH)
            s.listen(MAX_SOCKET)
            # s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            # s.setblocking(False)

            while True:
                conn, addr = s.accept()
                print(f'accpeted, {addr}')
                tpool.submit(handle_socket2, conn)
    except KeyboardInterrupt:
        print('\nstopping ytdlpy')
        tpool.shutdown(wait=False, cancel_futures=True)


# main
if __name__ == '__main__':
    print('running ytdlpy')
    uds_server2()


