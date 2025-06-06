import asyncio

async def client():
    reader, writer = await asyncio.open_unix_connection('/tmp/jukebox/ytdlpy.sock')

    writer.write(b'awfawfawf')
    writer.write_eof()
    await writer.drain()

    data = await reader.read()
    print(f'recv: {data.decode()}')

    writer.close()
    await writer.wait_closed()

asyncio.run(client())
