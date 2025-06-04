import json
import yt_dlp
from flask import Flask

app = Flask(__name__)

@app.route('/')
def index():
    res = getInfojson()
    return res, 202


def getInfojson():
    URL = 'https://www.youtube.com/watch?v=q6EoRBvdVPQ'
    URL_PLAYLIST = 'https://www.youtube.com/playlist?list=PLTMzl6sYFj5mgFDqdJv7zt_oRxY0LkVcx'

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

    # ℹ️ See help(yt_dlp.YoutubeDL) for a list of available options and public functions
    ydl_opts = {}
    with yt_dlp.YoutubeDL(ydl_opts) as ydl:
        infojson = ydl.extract_info(URL_PLAYLIST, download=False)
        # infojson = ydl.extract_info(URL, download=False)
        # ℹ️ ydl.sanitize_info makes the info json-serializable
        infojson = ydl.sanitize_info(infojson)

        if infojson.get('_type') == 'playlist':
            ret = [extractKeys(entry) for entry in infojson.get('entries')]
        else:
            ret = extractKeys(infojson)

    return ret

if __name__ == '__main__':
    app.run(debug=True)


# URL = 'https://www.youtube.com/watch?v=q6EoRBvdVPQ'

# ℹ️ See help(yt_dlp.YoutubeDL) for a list of available options and public functions
# ydl_opts = {}
# with yt_dlp.YoutubeDL(ydl_opts) as ydl:
#     info = ydl.extract_info(URL, download=False)

#     # ℹ️ ydl.sanitize_info makes the info json-serializable
#     print(json.dumps(ydl.sanitize_info(info)))


