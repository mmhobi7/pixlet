// main.go
package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"tidbyt.dev/pixlet/encode"
	"tidbyt.dev/pixlet/runtime"
)

var (
	apiPort  int
    applets []string
    APPLETS_PATH = "./applets"
)

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().IntVarP(&apiPort, "port", "p", 8080, "Port for serving API")
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Serves the api to render starlark scripts over HTTP.",
	// Args:  cobra.ExactArgs(1),
	Run:   api,
}

func api(cmd *cobra.Command, args []string) {
    handleReqs()
}

func homePage(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello!")
}

func listApplets(w http.ResponseWriter, r *http.Request) {
    applets = nil
    files, err := os.ReadDir(APPLETS_PATH)
	if err != nil {
        log.Println("There was an error listing files: ")
    }

    re := regexp.MustCompile("[.]star$")
	for _, file := range files {
        if !file.IsDir(){
            match := re.MatchString(file.Name())
            if match {
                applets = append(applets, strings.Replace(file.Name(),".star", "",1))
            }
        }
	}
    payload, err := json.Marshal(applets,)
    if err != nil {
		log.Printf("failed to marshal json: %v\n", err)
	}
    fmt.Fprintf(w, "%s", payload)
}

func addApplet(w http.ResponseWriter, r *http.Request) {
    file, handler, err := r.FormFile("file")
    fileName := handler.Filename
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        log.Println("There was an error uploading the file: ", err)

    }
    defer file.Close()

    match, _ := regexp.MatchString("[.]star$", fileName)
    if !match {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprintf(w, "Wrong file extension. .star needed");
        return;
    }
    f, err := os.OpenFile(fmt.Sprintf("%s/%s",APPLETS_PATH, fileName), os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        log.Println("There was an error writing the file to disk: ", err)
        return;
    }
    defer f.Close()
    _, err = io.Copy(f, file)
    if err != nil {
        fmt.Fprintf(w, "File %s uploaded successfully", fileName)
    } else {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprint(w, "There was an error writing the file to disk: ", err)
    }
}


func deleteApplet(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    file := vars["name"]
    if file != "" {
        file = file + ".star"
        filePath := fmt.Sprintf("%s/%s", APPLETS_PATH, file)
        if _, err := os.Stat(filePath); err == nil {
            err = os.Remove(filePath)
            if err != nil{
                w.WriteHeader(http.StatusNotFound)
                fmt.Fprintf(w, "There was an error deleting the file: %s", err)
                return
            } else {
                fmt.Fprintf(w, "File deleted")
            }
        } else if os.IsNotExist(err) {
            w.WriteHeader(http.StatusNotFound)
            fmt.Fprintf(w, "File not found")
        }
    }
}

func getApplet(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    file := vars["name"]
    config := map[string]string{}
    if file != "" {
        file = file + ".star"
        aplt, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", APPLETS_PATH,file))
        if err == nil {
            w.Header().Set("Content-Type", "text/plain")
            runtime.InitCache(runtime.NewInMemoryCache())
            applet := runtime.Applet{}
            err = applet.Load(file, aplt, nil)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                log.Println("Something happened loading applet: ")
            }
            roots, err := applet.Run(config)
            if err != nil {
                log.Printf("Error running script: %s\n", err)
                return
            }
    
            gif, err := encode.ScreensFromRoots(roots).EncodeGIF()
            if err != nil {
                log.Printf("Error rendering: %s\n", err)
                return
            }
            gif64 := b64.StdEncoding.EncodeToString([]byte(gif))
            fmt.Fprintf(w, "%s", gif64)
        } else {
            log.Println("File not found: ", err)
            w.WriteHeader(http.StatusNotFound)
        }
    }
}

func handleReqs() {
    myRouter := mux.NewRouter().StrictSlash(true)
    myRouter.HandleFunc("/", homePage)
    myRouter.HandleFunc("/list", listApplets).Methods("GET")
    myRouter.HandleFunc("/applet", addApplet).Methods("POST")
    myRouter.HandleFunc("/applet/{name}", deleteApplet).Methods("DELETE")
    myRouter.HandleFunc("/applet/{name}", getApplet).Methods("GET")
	fmt.Printf("listening on tcp/%d\n", apiPort)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", apiPort), myRouter))
}