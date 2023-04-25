package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

const (
	SocketPath = "/tmp/tosfs.sock"
)

func main() {
	flag.Parse()

	prepareSocketDir()

	r := mux.NewRouter()
	r.HandleFunc("/launcher", launcherHandler).Methods("POST")

	server := http.Server{
		Handler: r,
	}

	unixListener, err := net.Listen("unix", SocketPath)
	if err != nil {
		glog.Error(err)
		return
	}

	glog.Infoln("run launcher server.")
	if err := server.Serve(unixListener); err != nil {
		glog.Errorf("tosfs launcher server closed unexpected. %v", err)
	}
}

func prepareSocketDir() {
	if !isFileExisted(SocketPath) {
		pathDir := filepath.Dir(SocketPath)
		if !isFileExisted(pathDir) {
			os.MkdirAll(pathDir, os.ModePerm)
		}
	} else {
		os.Remove(SocketPath)
	}

	glog.Infof("socket dir %s is ready\n", filepath.Dir(SocketPath))
}

func launcherHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infoln("enter launcherHandler...")

	extraFields := make(map[string]string)

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		extraFields["errmsg"] = "read request body failed"
		glog.Errorf("%s: %v", extraFields["errmsg"], err)
		generateHttpResponse(w, "failure", http.StatusInternalServerError, extraFields)
		return
	}

	var bodyMap map[string]string
	if err := json.Unmarshal(body, &bodyMap); err != nil {
		extraFields["errmsg"] = "unmarshal request body failed"
		glog.Errorf("%s: %v\n", extraFields["errmsg"], err)
		generateHttpResponse(w, "failure", http.StatusInternalServerError, extraFields)
		return
	}

	cmd, ok := bodyMap["command"]
	if !ok {
		extraFields["errmsg"] = "request body is empty. we need field `command`"
		glog.Errorln(extraFields["errmsg"])
		generateHttpResponse(w, "failure", http.StatusBadRequest, extraFields)
		return
	}

	output, err := execCmd(cmd)
	if err != nil {
		extraFields["errmsg"] = fmt.Sprintf("exec command %s failed. output: %s, error: %v", cmd, output, err)
		glog.Errorln(extraFields["errmsg"])
		generateHttpResponse(w, "failure", http.StatusInternalServerError, extraFields)
		return
	}
	glog.Infof("exec command %s success. output: %s", cmd, output)

	extraFields["output"] = output
	generateHttpResponse(w, "success", http.StatusOK, extraFields)
}

func isFileExisted(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func execCmd(cmd string) (string, error) {
	output, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command %s failed: output %s, error: %v", cmd, string(output), err)
	}
	return string(output), nil
}

func generateHttpResponse(w http.ResponseWriter, result string, statusCode int, extra map[string]string) {
	res := make(map[string]string)
	res["result"] = result
	for k, v := range extra {
		res[k] = v
	}

	response, _ := json.Marshal(res)
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
