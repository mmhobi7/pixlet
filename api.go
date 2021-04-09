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

	for _, file := range files {
        if !file.IsDir(){
            r, _ := regexp.MatchString("[.]star$", file.Name()) // Remove extensions?
            if r {
                applets = append(applets, file.Name())
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
    fileName := r.FormValue("filename")
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        log.Println("There was an error uploading the file: ", err)

    }
    defer file.Close()

    f, err := os.OpenFile(fmt.Sprintf("%s/%s",APPLETS_PATH, handler.Filename), os.O_WRONLY|os.O_CREATE, 0666)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        log.Println("There was an error writing the file to disk: ", err)
    }
    defer f.Close()
    fmt.Fprintf(w, "File %s uploaded successfully", fileName)
    _, _ = io.Copy(f, file)
}


// func deletePixlet(w http.ResponseWriter, r *http.Request) {
//     fmt.Println("delete pixlets")
// }

func getApplet(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    file := vars["name"]
    config := map[string]string{}
    if file != "" {
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
                fmt.Printf("Error rendering: %s\n", err)
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
    // myRouter.HandleFunc("/applet/{name}", deleteApplet).Methods("DELETE")
    myRouter.HandleFunc("/applet/{name}", getApplet).Methods("GET")
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), myRouter))
}