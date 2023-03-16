package pkg

import (
	"bytes"
	"encoding/json"
	"github.com/breadchris/protoflow/runtimes"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"
)

func sendReadAndClose(c net.Conn, data []byte) []byte {
	defer c.Close()

	log.Debug().Msgf("Sending data to process: %v", string(data))

	_, err := c.Write(data)
	if err != nil {
		log.Error().Err(err).Msg("failed to send data to process")
		return nil
	}

	resultBuffer := bytes.Buffer{}
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return resultBuffer.Bytes()
		}
		resultBuffer.Write(buf[:nr])
	}
}

func findDefaultEntrypoint(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		switch file.Name() {
		case "index.js":
			fallthrough
		case "index.ts":
			return file.Name(), nil
		}
	}

	return "", nil
}

func getLanguageCmd(file string) string {
	ext := path.Ext(file)
	switch ext {
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".js":
		return "node"
	case ".ts":
		return "ts-node"
	}
	return ""
}

func getLanguageDefaultFunction(file string) string {
	ext := path.Ext(file)
	switch ext {
	case ".py":
		return "handler"
	case ".go":
		return "Handler"
	case ".js":
		return "handle"
	case ".ts":
		return "default"
	}
	return ""
}

func CallFunction(importPath, functionName string, input string) (result json.RawMessage, err error) {
	tmpDir, cleanup, err := createTempDir()
	if err != nil {
		log.Error().Err(err).Msg("failed to create temp dir")
		return
	}
	defer cleanup()

	content, err := runtimes.Runtimes.ReadFile("runtime.js")
	if err != nil {
		log.Error().Err(err).Msg("failed to read runtime.js")
		return
	}

	runtimePath := path.Join(tmpDir, "runtime.js")
	err = os.WriteFile(runtimePath, content, 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to write runtime.js")
		return
	}

	unixSocket := path.Join(tmpDir, "server.sock")

	l, err := net.Listen("unix", unixSocket)
	if err != nil {
		log.Error().Err(err).Msg("failed to listen")
		return
	}

	type RuntimeStdin struct {
		Input        string `json:"input"`
		ImportPath   string `json:"import_path"`
		FunctionName string `json:"function_name"`
	}

	absImportPath, err := filepath.Abs(importPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to get absolute path")
		return
	}

	entrypoint, err := findDefaultEntrypoint(absImportPath)
	if err != nil || entrypoint == "" {
		log.Error().Err(err).Str("importPath", absImportPath).Msg("failed to find default entrypoint")
		return
	}

	if functionName == "" {
		functionName = getLanguageDefaultFunction(entrypoint)
	}

	stdin := RuntimeStdin{
		Input:        input,
		ImportPath:   path.Join(absImportPath, entrypoint),
		FunctionName: functionName,
	}

	serData, err := json.Marshal(stdin)
	if err != nil {
		log.Error().Msgf("Error serializing data: %v", err)
		return
	}

	languageCmd := getLanguageCmd(entrypoint)
	if languageCmd == "" {
		log.Error().Msgf("Unknown language for file: %v", entrypoint)
		return
	}

	cmd := exec.Command(languageCmd, runtimePath)

	cmd.Dir = absImportPath
	cmd.Env = append(os.Environ(), "PROTOFLOW_SOCKET="+unixSocket)

	cleanup, err = startProcess(cmd)
	if err != nil {
		log.Error().Err(err).Msg("failed to send data to process")
		return
	}
	defer cleanup()

	log.Debug().Msg("Waiting for connection")

	fd, err := l.Accept()
	if err != nil {
		log.Error().Err(err).Msg("failed to accept")
		return
	}

	err = fd.SetDeadline(time.Now().Add(10 * time.Minute))
	if err != nil {
		log.Error().Err(err).Msg("failed to set deadline")
		return
	}

	processResult := sendReadAndClose(fd, serData)

	err = cmd.Wait()
	if err != nil {
		log.Error().Err(err).Msg("failed to wait for process")
		return
	}

	var resultData struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}

	err = json.Unmarshal(processResult, &resultData)
	result = resultData.Result
	return
}
