package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/breadchris/protoflow/runtimes"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
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
		case "main.py":
			return file.Name(), nil
		}
	}

	return "", nil
}

type Runtime struct {
	LanguageCmd     string
	DefaultFunction string
	File            string
}

func loadLanguageRuntime(file string) (Runtime, error) {
	ext := path.Ext(file)
	switch ext {
	case ".py":
		return Runtime{
			LanguageCmd:     "python",
			DefaultFunction: "handler",
			File:            "runtime.py",
		}, nil
	case ".js":
		return Runtime{
			LanguageCmd:     "node",
			DefaultFunction: "handle",
			File:            "runtime.js",
		}, nil
	case ".ts":
		return Runtime{
			LanguageCmd:     "ts-node",
			DefaultFunction: "default",
			File:            "runtime.js",
		}, nil
	}
	return Runtime{}, fmt.Errorf("unknown language for file %s", file)
}

type RuntimeStdin struct {
	Input        string `json:"input"`
	ImportPath   string `json:"import_path"`
	FunctionName string `json:"function_name"`
}

func CallFunction(importPath, functionName string, input string) (result json.RawMessage, err error) {
	tmpDir, cleanup, err := createTempDir()
	if err != nil {
		log.Error().Err(err).Msg("failed to create temp dir")
		return
	}
	defer cleanup()

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

	runtime, err := loadLanguageRuntime(entrypoint)
	if err != nil {
		log.Error().Err(err).Msg("failed to load language runtime")
		return
	}

	content, err := runtimes.Runtimes.ReadFile(runtime.File)
	if err != nil {
		log.Error().Err(err).Msg("failed to read runtime.js")
		return
	}

	runtimePath := path.Join(tmpDir, runtime.File)
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

	if functionName == "" {
		functionName = runtime.DefaultFunction
	}

	ext := filepath.Ext(entrypoint)
	basename := strings.TrimSuffix(entrypoint, ext)

	stdin := RuntimeStdin{
		Input: input,
		//ImportPath:   path.Join(absImportPath, entrypoint),
		ImportPath:   basename,
		FunctionName: functionName,
	}

	serData, err := json.Marshal(stdin)
	if err != nil {
		log.Error().Msgf("Error serializing data: %v", err)
		return
	}

	cmd := exec.Command(runtime.LanguageCmd, runtimePath)

	cmd.Dir = absImportPath
	cmd.Env = append(os.Environ(), "PROTOFLOW_SOCKET="+unixSocket)
	cmd.Env = append(cmd.Env, "PYTHONPATH="+absImportPath)

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
