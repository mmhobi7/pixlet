// main.go
package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
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
	apiPort      int
	applets      []string
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
	Run: api,
}

func api(cmd *cobra.Command, args []string) {
	handleReqs()
}

func homePage(w http.ResponseWriter, r *http.Request) {
	html, err := os.ReadFile("web/preview.html")
	if err != nil {
		log.Printf("There was an error opening preview.html: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(html))
}

func listApplets(w http.ResponseWriter, r *http.Request) {
	applets = nil
	files, err := os.ReadDir(APPLETS_PATH)
	if err != nil {
		log.Println("There was an error listing files: ")
	}

	re := regexp.MustCompile("[.]star$")
	for _, file := range files {
		if !file.IsDir() {
			match := re.MatchString(file.Name())
			if match {
				applets = append(applets, strings.Replace(file.Name(), ".star", "", 1))
			}
		}
	}
	payload, err := json.Marshal(applets)
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
		fmt.Fprintf(w, "Wrong file extension. .star needed")
		return
	}
	f, err := os.OpenFile(fmt.Sprintf("%s/%s", APPLETS_PATH, fileName), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("There was an error writing the file to disk: ", err)
		return
	}
	defer f.Close()
	_, err = io.Copy(f, file)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "There was an error writing the file to disk: ", err)
	} else {
		fmt.Fprintf(w, "File %s uploaded successfully", fileName)
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
			if err != nil {
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
	magnify := 1
	config := map[string]string{}
	for k, vals := range r.URL.Query() {
		config[k] = vals[0]
		fmt.Printf("%s -> %s\n", k, vals[0])
	}
	if file != "" {
		file = file + ".star"
		aplt, err := os.ReadFile(fmt.Sprintf("%s/%s", APPLETS_PATH, file))
		if err == nil {
			w.Header().Set("Content-Type", "text/plain")
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
			screens := encode.ScreensFromRoots(roots)

			filter := func(input image.Image) (image.Image, error) {
				if magnify <= 1 {
					return input, nil
				}
				in, ok := input.(*image.RGBA)
				if !ok {
					return nil, fmt.Errorf("image not RGBA, very weird")
				}

				out := image.NewRGBA(
					image.Rect(
						0, 0,
						in.Bounds().Dx()*magnify,
						in.Bounds().Dy()*magnify),
				)
				for x := 0; x < in.Bounds().Dx(); x++ {
					for y := 0; y < in.Bounds().Dy(); y++ {
						for xx := 0; xx < magnify; xx++ {
							for yy := 0; yy < magnify; yy++ {
								out.SetRGBA(
									x*magnify+xx,
									y*magnify+yy,
									in.RGBAAt(x, y),
								)
							}
						}
					}
				}

				return out, nil
			}

			var buf []byte

			maxDuration := 0

			buf, err = screens.EncodeGIF(maxDuration, filter)

			// gif, err := encode.ScreensFromRoots(roots).EncodeGIF()
			if err != nil {
				log.Printf("Error rendering: %s\n", err)
				return
			}
			gif64 := b64.StdEncoding.EncodeToString(buf)
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
	myRouter.HandleFunc("/applets", listApplets).Methods("GET")
	myRouter.HandleFunc("/applet", addApplet).Methods("POST")
	myRouter.HandleFunc("/applet/{name}", deleteApplet).Methods("DELETE")
	myRouter.HandleFunc("/applet/{name}", getApplet).Methods("GET")
	myRouter.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))
	fmt.Printf("listening on tcp/%d\n", apiPort)
	runtime.InitCache(runtime.NewInMemoryCache())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", apiPort), myRouter))
}
