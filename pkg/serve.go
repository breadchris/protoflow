package pkg

/*
http server that forwards requests sent to it to the corresponding block.

Example:

POST /run/basic
{
  "hello": "world"
}

will call the block `basic` with the POST body as arguments.

The routes are registered by accepting a directory and listing all dirs. The dirs will be used as the route path to call when invoking the function
*/
