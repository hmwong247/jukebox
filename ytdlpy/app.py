import io
import json
import os
import socket
import yt_dlp
import asyncio


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
        # infojson = ydl.extract_info(URL, download=False)
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



async def handle_socket(reader, writer):
    msg = await reader.read() # read until EOF
    data = json.loads(msg)
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

    writer.write(json.dumps(resp).encode())
    writer.write_eof()
    await writer.drain()

    writer.close()
    await writer.wait_closed()


async def uds_server():
    server = await asyncio.start_unix_server(handle_socket, path=SOCKET_PATH)
    async with server:
        await server.serve_forever()


# main
if __name__ == '__main__':
    print('running ytdlpy')

    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)

    os.makedirs(SOCKET_DIR, exist_ok=True)
    # s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    # s.bind(SOCKET_PATH)
    # s.setblocking(False)
    # s.listen(MAX_SOCKET)

    asyncio.run(uds_server())


