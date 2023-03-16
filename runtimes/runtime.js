const net = require('net');

if (process.stdout._handle) {
	process.stdout._handle.setBlocking(true);
}

function log(msg, data) {
	process.stdout.write(JSON.stringify({
		"level": "debug",
		"context": "runtime",
		"data": data,
		"msg": msg,
	}) + "\n")
}

function promisify(fn) {
	return function (...args) {
		return new Promise((resolve, reject) => {
			fn(...args, (err, result) => {
				if (err) {
					return reject(err);
				}
				resolve(result);
			});
		});
	};
}

const socket = process.env.PROTOFLOW_SOCKET;
if (!socket) {
	throw new Error("PROTOFLOW_SOCKET environment variable not set");
}

const sock = new net.createConnection({ path : socket });

sock.on('error', function(err) {
	console.error(err);
	flushOutputAndExit(-1);
});

sock.on('close', function() {
	flushOutputAndExit(0);
});

sock.on('data', function(data) {
	const inputData = JSON.parse(data.toString());
	log('received data', inputData);

	void init(inputData);
})

async function returnResult(socket, programOutput) {
	sock.write(JSON.stringify(programOutput));
	sock.end();
}

async function init(inputData) {
	try {
		// Inside of the try catch statement to ensure that any exceptions thrown here are logged.
		const socket = inputData[ "socket" ];
		const input = inputData[ "input" ];

		const importPath = inputData["import_path"];
		const functionName = inputData["function_name"];

		const importedFile = require(importPath);
		const mainEntrypoint = importedFile[functionName];

		const result = await mainEntrypoint(input);

		await returnResult(socket, {
			"result": result,
		});
	} catch ( e ) {
		log("error", e.toString());
		if( e.stack ) {
			e = e.stack.toString()
		} else {
			e = e.toString();
		}
		flushOutputAndExit(-1);
	}
}

function flushOutputAndExit(exitCode) {
	// Configure the streams to be blocking
	blockStreams(process.stdout);
	blockStreams(process.stderr);


	// Allow Node to cleanup any internals before the next process tick
	setImmediate(function callProcessExitWithCode() {
		process.exit(exitCode);
	});
}

function blockStreams(stream) {
	if (!stream || !stream._handle || !stream._handle.setBlocking) {
		// Not able to set blocking so just bail out
		return;
	}
	stream._handle.setBlocking(true);
}
