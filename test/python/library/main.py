import requests


async def handler(params):
    data = requests.get("https://jsonplaceholder.typicode.com/todos/1")
    return {
        "Hello": data.json()
    }
