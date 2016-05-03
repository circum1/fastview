package main

import (
    "fmt"
    "net/http"
    //~ "time"
    //~ "github.com/tonnerre/golang-pretty"
    "path"
    "strings"
    "os"
    "io/ioutil"
    "os/exec"
    "strconv"
    //~ "log"
)

//~ const (
    //~ imageBaseUrl = "/localfs"
//~ )

const (
    NOTHING = iota
    DIRECTORY = iota
    REGULAR = iota
)

type Config struct {
    //~ localImageBaseDir string
    cacheDir string
}

// local directories to serve
var localDirsMap = map[string]string {
    "/pictures": "/home/fery/Pictures",
}

var allowedSizes=[...]int {640,1600}

//~ var localDirsMapReverse = new map[string]string

var config = Config {
    //~ "/home/fery/Pictures",
    "/home/fery/magan/fastview/fastview/data/cache",
}

type Image struct {
    // full URL path (including the source identifier, e.g. localImageBaseDir)
    contentURL string
    //lastMod time.Time
    //cache map[int]Cache
}

func (i Image) x(){

}

// Map of URL Path -> Image
var imagesMap map[string]Image = make(map[string]Image)

func sendErr(w http.ResponseWriter, error string) {
    fmt.Fprint(w, error);
}

func inspectFile(name string) int {
    fstat, err := os.Stat(name)
	if err != nil {
        return NOTHING
    }
    if fstat.IsDir() {
        return DIRECTORY
    } else {
        return REGULAR
    }
}

// ContentProvider & implementers

type ContentProvider interface{
    // Convert ContentProvider-specific path to public (fastview) URL
    Path2Url(cpPath string) string
    // Convert public (fastview) URL (only path part) to ContentProvider-specific path
    Url2Path(url string) string
    mkCacheFilename(abspath, sizeStr string) string
}

type LocalFilesystemProvider struct {
    // base of url path
    baseUrl string
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

func (cp *LocalFilesystemProvider) serveDir(urlPath string, fsPath string, w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(fsPath)
	if err != nil {
		sendErr(w, err.Error())
	}
    filenames := make([]string, 0, len(files))
    dirnames := make([]string, 0, len(files))
	for _, file := range files {
        url := fmt.Sprintf("\"%v\"", path.Join(urlPath, file.Name()))
        if inspectFile(path.Join(fsPath, file.Name()))==DIRECTORY {
            dirnames=append(dirnames, url)
        } else {
            filenames=append(filenames, url)
        }
	}
    fmt.Fprintf(w, "{\n\"images\": [\n%v\n],\n\"dirs\": [\n%v\n]\n}",
        strings.Join(filenames, ",\n"), strings.Join(dirnames, ",\n"))
    w.Header().Set("Content-Type", "application/json")
}

func (cp *LocalFilesystemProvider) serveSingleImage(abspath string, w http.ResponseWriter, r *http.Request) {
    sizeStr := r.FormValue("size")
    size, err := strconv.Atoi(sizeStr)
    if err!=nil || len(sizeStr)==0 || sizeStr=="full" || size>allowedSizes[len(allowedSizes)-1] {
        http.ServeFile(w, r, abspath)
        return
    }
    for _, val:=range allowedSizes {
        if size<=val {
            size=val
            break;
        }
    }
    sizeStr=strconv.Itoa(size)
    fmt.Printf("serving from cache, size: %v, %v\n", size, sizeStr)
    cachepath := cp.mkCacheFilename(abspath, sizeStr)
    if !mkThumbnail(cachepath, abspath, size, true) {
        // fallback
        http.ServeFile(w, r, abspath)
        return
    }
    http.ServeFile(w, r, cachepath)
}

func mkThumbnail(cachepath, abspath string, size int, sync bool) bool {
    if inspectFile(cachepath)==REGULAR {
        cacheStat, _ := os.Stat(cachepath)
        origStat, _ := os.Stat(abspath)
        if  cacheStat.ModTime().After(origStat.ModTime()) {
            // we already have it
            return true
        }
    }
    var callback chan bool
    if sync {
        callback=make(chan bool)
    }
    task := RescaleTask{abspath, cachepath, size, callback}
    go func () {urgentRescaleTasks <-task}()
    if sync {
        // this will synchronize
        return <-callback
    }
    return true
}

func (cp *LocalFilesystemProvider) mkCacheFilename(abspath, sizeStr string) string {
    return path.Join(config.cacheDir, sizeStr, cp.Path2Url(abspath))
}

type RescaleTask struct {
    ImagePath string
    CachePath string
    Size int
    // channel to indicate the result of conversion (true: success, false: failure)
    Callback chan bool
}

var urgentRescaleTasks chan RescaleTask = make(chan RescaleTask)

func resizeImage(dest, src string, size int) bool {
	var args = []string{
		"-s", fmt.Sprintf("%v", size),
		"-o", dest,
		src,
	}
	var cmd *exec.Cmd
	path, _ := exec.LookPath("vipsthumbnail")
	cmd = exec.Command(path, args...)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error while running '%v': %v", args, err)
	}
	return err==nil
}

func thumbnailGenerator() {
    for task := range urgentRescaleTasks {
        fmt.Printf("Rescale task: %v\n", task)
        dirname := path.Dir(task.CachePath)
        result := true
        if err := os.MkdirAll(dirname, 0755); err != nil {
            fmt.Fprintf(os.Stderr, "error creating cache directory: %v - error: %v\n", dirname, err)
            result=false
        } else {
            //~ var size int
            //~ switch task.ThumbSize {
            //~ case THUMB_SMALL: size=320
            //~ case THUMB_MED: size=800
            //~ default: result=false;
            //~ }
            if result {
                result = resizeImage(task.CachePath, task.ImagePath, task.Size)
            }
        }
        if task.Callback!=nil {
            task.Callback <- result
            close(task.Callback)
        }
    }
}


// Serves images and directories from the local filesystem.
//
// If the request points to a directory, the directory contents is
// put to the response in JSON
//
// If the request points to an image, the image itself is put to the response.
func (cp *LocalFilesystemProvider) serveLocal(w http.ResponseWriter, r *http.Request) {
    //~ fmt.Printf("serveLocal called: %# v\n", pretty.Formatter(r))

    cleanedPath := path.Clean(r.URL.Path)
    abspath := cp.Url2Path(cleanedPath)
    fmt.Printf("serveLocal() cleanedPath: %v, fs path: %v\n", cleanedPath, abspath)
    if len(abspath)==0 {
        fmt.Fprintf(w, "invalid URL")
        return
    }

    switch inspectFile(abspath) {
    case NOTHING: fmt.Fprintf(w, "Bad path")
    case DIRECTORY: cp.serveDir(cleanedPath, abspath, w, r)
    case REGULAR: cp.serveSingleImage(abspath, w, r)
    }
}

func main() {
    fmt.Printf("Starting\n")
    go thumbnailGenerator()
    http.Handle("/site/", http.FileServer(http.Dir(".")))

    for urlbase, dir := range localDirsMap {
        cp := LocalFilesystemProvider{urlbase, dir}
        http.HandleFunc(urlbase+"/", cp.serveLocal)
    }
    http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {http.Redirect(w, r, "/site/", http.StatusMovedPermanently)})
    fmt.Println(http.ListenAndServe("localhost:8080", nil))
}
