package main

import (
    "fmt"
    "net/http"
    "time"
    "github.com/tonnerre/golang-pretty"
    "path"
    "strings"
    "os"
    "io/ioutil"
    //~ "log"
)

//~ const (
    //~ imageBaseUrl = "/localfs"
//~ )

const (
    THUMB_SMALL = iota
    THUMB_MED = iota
)

type Config struct {
    //~ localImageBaseDir string
    cacheDir string
}

// local directories to serve
var localDirsMap = map[string]string {
    "/pictures": "/home/fery/Pictures"
}

var localDirsMapReverse = new map[string]string

var config = Config {
    //~ "/home/fery/Pictures",
    "/home/fery/magan/fastview/fastview/data/cache"
}



type Image struct {
    // full URL path (including the source identifier, e.g. localImageBaseDir)
    contentURL string
    //lastMod time.Time
    cache map[int]Cache
}

func (i Image) x(){

}

// ContentProvider & implementers

type ContentProvider interface{
    // Convert ContentProvider-specific path to public (fastview) URL
    Path2Url(cpPath string) string
    // Convert public (fastview) URL (only path part) to ContentProvider-specific path
    Url2Path(url string) string
}

type LocalFilesystemProvider struct {
    // base of url path
    baseUrl string,
    // base directory
    baseDir string
}

func (cp *LocalFilesystemProvider) Path2Url(cpPath string) string {
    if path.IsAbs(cpPath) {
        if strings.HasPrefix(cpPath, cp.baseDir) {
            cpPath=cpPath[len(cp.baseDir):]
        }
    }
    return path.Join(cp.baseUrl, cpPath)
}

func (cp *LocalFilesystemProvider) Url2Path(url string) string {
    if !strings.HasPrefix(url, cp.baseUrl) {
        return ""
    }
    url=url[len(cp.baseUrl):]
    return path.Join(cp.baseDir, url)
}

var contentProvider ContentProvider = &LocalFilesystemProvider{}

type Cache struct {
    contentPath string
    lastMod time.Time
    // the longer dimension
    resolution int
    cachePath string
}

// Map of URL Path -> Image
var imagesMap map[string]Image = make(map[string]Image)

func sendErr(w http.ResponseWriter, error string) {
    fmt.Fprint(w, error);
}

func serveDir(abspath string, w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(abspath)
	if err != nil {
		sendErr(w, err.Error())
	}
    filenames := make([]string, 0, len(files))
    dirnames := make([]string, 0, len(files))
	for _, file := range files {
        fullPath := path.Join(abspath, file.Name())
        url := fmt.Sprintf("\"%v\"", contentProvider.Path2Url(fullPath))
        fstat, _ := os.Stat(fullPath)
        if fstat.IsDir() {
            dirnames=append(dirnames, url)
        } else {
            filenames=append(filenames, url)
        }
	}
    fmt.Fprintf(w, "{\n\"images\": [\n%v\n],\n\"dirs\": [\n%v\n]\n}",
        strings.Join(filenames, ",\n"), strings.Join(dirnames, ",\n"))
    w.Header().Set("Content-Type", "application/json")
}

func serveSingleImage(abspath string, w http.ResponseWriter, r *http.Request) {
    size := r.FormValue("size")
    if len(size)==0 || size=="full" {
        http.ServeFile(w, r, abspath)
        return
    }
    if size=="med" {
        fullpath := path.join(config.cacheDir, contentProvider.Path2Url(abspath))
        dirname := path.Dir(fullpath)
        if err := os.MkdirAll(dirname, 0644); err != nil {
            fmt.Fprintf(os.Stderr, "error creating cache directory: %v", )
            // fallback
            http.ServeFile(w, r, abspath)
            return
        }

    }
}

// Serves images and directories from the local filesystem.
//
// If the request points to a directory, the directory contents is
// put to the response in JSON
//
// If the request points to an image, the image itself is put to the response.
func serveLocal(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("serveLocal called: %# v\n", pretty.Formatter(r))
    //~ relpath := r.URL.Path
    //~ if len(relpath)<=len(imageBaseUrl) {
        //~ relpath="."
    //~ } else {
        //~ relpath=path.Clean(relpath[len(imageBaseUrl):])
        //~ if strings.ContainsAny(relpath[:1], "./") {
            //~ fmt.Fprintf(w, "invalid URL")
            //~ return
        //~ }
    //~ }
    //~ abspath := path.Join(config.localImageBaseDir, relpath)

    cleanedPath := path.Clean(r.URL.Path)
    abspath := contentProvider.Url2Path(cleanedPath)
    fmt.Printf("cleanedPath: %v\nabspath: %v\n", cleanedPath, abspath)
    if len(abspath)==0 {
        fmt.Fprintf(w, "invalid URL")
        return
    }

    //~ fmt.Printf("abspath: %v\n", abspath)
    fstat, err := os.Stat(abspath)
	if err != nil {
        fmt.Fprintf(w, "Bad path")
        return
    }
    if fstat.IsDir() {
        serveDir(abspath, w, r)
    } else {
        serveSingleImage(abspath, w, r)
    }
}

//~ func myFileServer(w http.ResponseWriter, r *http.Request) {
    //~ handler.ServeHTTP(w, r)
//~ }

//~ var handler http.Handler =

func main() {
    fmt.Printf("Starting\n")
    http.Handle("/site/", http.FileServer(http.Dir(".")))
    http.HandleFunc("/"+imageBaseUrl+"/", serveLocal)
    http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {http.Redirect(w, r, "/site/", http.StatusMovedPermanently)})
    fmt.Println(http.ListenAndServe("localhost:8080", nil))
}
