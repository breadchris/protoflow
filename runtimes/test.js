const importedFile = require('./index');
const mainEntrypoint = importedFile[functionName];

console.log(input)
const result = mainEntrypoint(input);
console.log(result)
