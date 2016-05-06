package main

// TODO:
// config file
// preload cache (resize in backend, preload on frontend)

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
    "crypto/md5"
    "github.com/BurntSushi/toml"
    "github.com/abbot/go-http-auth"
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

type MappedDir struct {
    Url string
    Rootdir string
    Username string
    Password string
}

type Config struct {
    //~ localImageBaseDir string
    CacheDir string
    Port int
    LocalDirs []MappedDir
}


var allowedSizes=[...]int {640,1600}

var config Config

type Image struct {
    // full URL path (including the source identifier, e.g. localImageBaseDir)
    contentURL string
    //lastMod time.Time
    //cache map[int]Cache
}

func (i Image) x(){

}

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

// Map of URL Path -> Image
//~ var imagesMap map[string]Image = make(map[string]Image)

// ContentProvider & implementers

type ContentProvider interface{
    // Convert ContentProvider-specific path to public (fastview) URL
    Path2Url(cpPath string) string
    // Convert public (fastview) URL (only path part) to ContentProvider-specific path
    Url2Path(url string) string
    mkCacheFilename(abspath, sizeStr string) string
}

type LocalFilesystemProvider struct {
    MappedDir
    //~ // base of url path
    //~ baseUrl string
    //~ // base directory
    //~ baseDir string
}

func (cp *LocalFilesystemProvider) Path2Url(cpPath string) string {
    if path.IsAbs(cpPath) {
        if strings.HasPrefix(cpPath, cp.Rootdir) {
            cpPath=cpPath[len(cp.Rootdir):]
        }
    }
    return path.Join(cp.Url, cpPath)
}

func (cp *LocalFilesystemProvider) Url2Path(url string) string {
    if !strings.HasPrefix(url, cp.Url) {
        return ""
    }
    url=url[len(cp.Url):]
    return path.Join(cp.Rootdir, url)
}

func (cp *LocalFilesystemProvider) mkCacheFilename(abspath, sizeStr string) string {
    return path.Join(config.CacheDir, sizeStr, cp.Path2Url(abspath))
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
    cachepath := mkThumbnail(cp, abspath, size, true)
    if (len(cachepath)==0) {
        // fallback
        http.ServeFile(w, r, abspath)
        return
    }
    http.ServeFile(w, r, cachepath)
    // create other size(s)
    for _, val:=range allowedSizes {
        if size!=val {
            fmt.Printf("creating others in advance, size: %v, fname: %v\n", val, path.Base(abspath))
            mkThumbnail(cp, abspath, val, false)
        }
    }
}

// Creates resized images (if it is not created already).
// sync: if true, waits until the resized image is created, and returns its path.
func mkThumbnail(cp ContentProvider, abspath string, size int, sync bool) string {
    cachepath := cp.mkCacheFilename(abspath, strconv.Itoa(size))
    if inspectFile(cachepath)==REGULAR {
        cacheStat, _ := os.Stat(cachepath)
        origStat, _ := os.Stat(abspath)
        if  cacheStat.ModTime().After(origStat.ModTime()) {
            // we already have it
            return cachepath
        }
    }
    var callback chan bool
    if sync {
        callback=make(chan bool)
    }
    task := RescaleTask{abspath, cachepath, size, callback}
    if sync {
        go func () {urgentRescaleTasks <-task}()
        // this will synchronize
        if (<-callback) {
            return cachepath
        }
        // an error occurred
        return ""
    } else {
        go func () {rescaleTasks <-task}()
    }
    // async
    return ""
}

type RescaleTask struct {
    ImagePath string
    CachePath string
    Size int
    // channel to indicate the result of conversion (true: success, false: failure)
    Callback chan bool
}

var urgentRescaleTasks chan RescaleTask = make(chan RescaleTask)
var rescaleTasks chan RescaleTask = make(chan RescaleTask)

func resizeImage(dest, src string, size int) bool {
	var args = []string{
		"-s", fmt.Sprintf("%v", size),
		"-o", dest,
		src,
	}
	var cmd *exec.Cmd
    var err error
	if path, err := exec.LookPath("vipsthumbnail"); err==nil {
        cmd = exec.Command(path, args...)
        err = cmd.Run()
        if err == nil {
            return true
        }
    }
    fmt.Printf("Error while resizing: %v", err)
    return false
}

func thumbnailGenerator() {
    for {
        var task RescaleTask
        select {
        case task=<-urgentRescaleTasks: // has priority
        default:
            select {
            case task=<-rescaleTasks:
            default:
                select { // both are empty right now; wait for any
                    case task=<-urgentRescaleTasks:
                    case task=<-rescaleTasks:
                }
            }
        }
        // Here or another, we have a task :)
        fmt.Printf("Rescale task: %v\n", task)

        // recheck if exists -- a file can be put more than once in the chan...
        if inspectFile(task.CachePath)==REGULAR {
            cacheStat, _ := os.Stat(task.CachePath)
            origStat, _ := os.Stat(task.ImagePath)
            if  cacheStat.ModTime().After(origStat.ModTime()) {
                if task.Callback!=nil {
                    task.Callback <- true
                    close(task.Callback)
                }
                return
            }
        }

        dirname := path.Dir(task.CachePath)
        result := true
        if err := os.MkdirAll(dirname, 0755); err != nil {
            fmt.Fprintf(os.Stderr, "error creating cache directory: %v - error: %v\n", dirname, err)
            result=false
        } else {
            result = resizeImage(task.CachePath, task.ImagePath, task.Size)
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

func (cp *LocalFilesystemProvider) Secret(user, realm string) string {
    //~ fmt.Printf("Secret(%v, %v)\nusername: %v, realm: %v, pass: |%v\n",user, realm, cp.Username, cp.Url, cp.Password)
    // HA1: MD5(username:realm:password)
    if cp.Username!=user || cp.Url!=realm {
        return ""
    }
    h := md5.New()
    fmt.Fprintf(h, "%v:%v:%v", cp.Username, cp.Url, cp.Password)
    return fmt.Sprintf("%x", h.Sum(nil))
}

func main() {
    if confData, err := ioutil.ReadFile(path.Join(os.Getenv("HOME"),".fastviewrc")); err==nil {
        confString:=string(confData)
        _, err := toml.Decode(confString, &config)
        if err!=nil {
            fmt.Printf(".fastviewrc parse error: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Println("Missing config file ~/.fastviewrc -- running with hardcoded defaults")
        localDirsMap := MappedDir{"/pictures", "/home/fery/Pictures", "", ""}
        config = Config {
            "/home/fery/magan/fastview/fastview/data/cache",
            8080,
            []MappedDir{localDirsMap},
        }
    }
    if len(config.CacheDir)==0 || inspectFile(config.CacheDir)!=DIRECTORY {
        fmt.Printf("Invalid cache dir '%v' -- check config file or create cache directory\n", config.CacheDir)
        return
    }
    fmt.Printf("Using cache dir %v\n", config.CacheDir)
    if len(config.LocalDirs)==0 {
        fmt.Printf("No local directory is served -- check config file\n")
        return
    }

    go thumbnailGenerator()
    http.Handle("/site/", http.FileServer(http.Dir(".")))
    //~ http.HandleFunc("/site/", func (w http.ResponseWriter, r *http.Request) {
            //~ w.WriteHeader(http.StatusUnauthorized)
        //~ fmt.Fprintf(w, "Authorization required")
        //~ })



    for _, val := range config.LocalDirs {
        fmt.Printf("Serving directory %v under %v\n", val.Rootdir, val.Url)
        cp := LocalFilesystemProvider{val}
        auth := auth.NewDigestAuthenticator(cp.Url, cp.Secret)
        http.HandleFunc(val.Url+"/", auth.JustCheck(cp.serveLocal))
    }
    http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {http.Redirect(w, r, "/site/", http.StatusMovedPermanently)})
    hostString := ":8080"
    if config.Port!=0 {
        hostString=":"+strconv.Itoa(config.Port)
    }
    fmt.Printf("Listening on %v\n", hostString)
    fmt.Println(http.ListenAndServe(hostString, nil))
}
