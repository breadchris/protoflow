import asyncio
import json
import os
import sys
import traceback
import socket


def log(msg, data):
    content = json.dumps({
        "type": "log",
        "context": "runtime",
        "data": data,
        "msg": msg
    })
    print(content, file=sys.stderr)


def read_json_from_socket(s: socket.socket):
    data = s.recv(1024)
    return json.loads(data.decode("utf-8"))


def print_return_data(s, result, error=None):
    return_data = json.dumps({
        "result": result,
        "error": error
    })
    s.sendall(return_data.encode())


async def main():
    unix_socket = os.environ["PROTOFLOW_SOCKET"]
    if not unix_socket:
        log("PROTOFLOW_SOCKET not set", {})
        exit(-1)

    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
        s.connect(unix_socket)

        input_data = read_json_from_socket(s)

        block_input = input_data["input"]

        import_path = input_data["import_path"]
        function_name = input_data["function_name"]

        try:
            imported_file = __import__(import_path)
            func = getattr(imported_file, function_name)

            result = await func(block_input)

            print_return_data(s, result)
        except Exception as e:
            log("Error running function", {
                "error": str(e),
                "traceback": traceback.format_exc()
            })
            print_return_data(s, None, traceback.format_exc())
            exit(-1)

        exit(0)


if __name__ == "__main__":
    asyncio.run(main())
