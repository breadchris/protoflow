const axios = require('axios')

async function handle(input) {
    const res = await axios.get('https://jsonplaceholder.typicode.com/todos/1');
    return res.data;
}

module.exports = {
    handle,
}
