import json
import os
import sys
import traceback
import socket

root = os.path.abspath(os.path.join(os.path.dirname(__file__)))
sys.path.insert(0, root)



def get_return_data(result, error):
    return


def print_return_data(unix_socket, result, error=None):
    with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as s:
        s.connect(unix_socket)

        return_data = json.dumps({
            "result": result,
            "error": error
        })
        s.sendall(return_data)


def main():
    raw_input_data = sys.stdin.read()
    input_data = json.loads(raw_input_data)

    block_input = input_data["input"]
    unix_socket = input_data["socket"]

    import_path = input_data["import_path"]
    function_name = input_data["function_name"]

    try:
        imported_file = __import__(import_path)
        func = getattr(imported_file, function_name)

        result = func(block_input)

        print_return_data(unix_socket, result)
    except:
        print_return_data(unix_socket, None, traceback.format_exc())
        exit(-1)

    exit(0)


if __name__ == "__main__":
    main()
