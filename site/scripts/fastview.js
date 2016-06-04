"use strict";

var thumbnailSize=640;
var fullscreenSize=1600;

var thumbnailStyleSize=320;

$(document).ready(function() {
    $("#downloadBtn").click(function() {
        var image=$.bbq.getState("image");
        if (image) {
            window.location.href=image;
        }
    });

    // translate swipe events to keyboard events
    var hammertime = new Hammer($("#fullscreen-img").get(0), {domEvents: true, preventDefault: false});
    hammertime.on('swipe', function(ev) {
        var keycode=ev.direction==Hammer.DIRECTION_RIGHT ? 37 : 39;
        var e = jQuery.Event("keyup");
        e.which = keycode;
        $(document).trigger(e);
    });

    var decoratorTimeoutId;
    $(document).mousemove(function() {
        clearTimeout(decoratorTimeoutId);
        var decoration=$("#fullscreen-decoration");
        decoration.fadeIn();
        decoratorTimeoutId=setTimeout(function() {
            decoration.fadeOut();
        }, 2000);
    });


    $("#thumb-size").on("keydown", function(e) {
        if (e.which == 13) {
            var newVal=parseInt($(this).val());
            if (newVal>=96 && newVal<=fullscreenSize) {
                thumbnailStyleSize=newVal;
                $.bbq.removeState("index");
                $(window).trigger( 'hashchange' );
            }
        }
    });

    $(window).bind( 'hashchange', function(e) {
        console.log("hashchange "+window.location.href);
        var index=parseInt($.bbq.getState("index"));
        if (!(index>=0)) index=-1; // aware of NaN
        var dir=$.bbq.getState("dir");
        if (dir) {
            selectPage("thumbnails");
            loadDir(dir, index);
            return;
        }
        var image=$.bbq.getState("image");
        if (image) {
            if ($("#thumbnails").children().length==0) {
                // presumably opened a direct link to an image -- must load the thumbnails...
                loadDir(image.substring(0,image.lastIndexOf("/")), -1);
            }
            selectPage("fullscreen");
            showImage(image, $.bbq.getState("size"), index);
            return;
        }
    });

    if (window.location.href.indexOf("#")<0) {
        //~ window.location+="#dir=/pictures/";
        $.bbq.pushState({"dir": "/pictures"});
    } else {
        $(window).trigger( 'hashchange' );
    }
});

// single page app support
// @param page "fullscreen" or "thumbnails"
function selectPage(page) {
    $("#fullscreen-page").hide();
    $("#thumbnails-page").hide();

    $("#"+page+"-page").show();

    // direction: -1 or +1
    function nextImage(direction) {
        var newIndex=parseInt($.bbq.getState("index"), 10)+direction;
        if (newIndex<0) return undefined;
        var imgEl=$("#thumbnails img:eq("+newIndex+")");
        if (imgEl.length==0) return undefined;
        imgEl.trigger("click");
    }

    function handleFullscreenKeys(e) {
        if (e.which == 27) { // escape key maps to keycode `27`
            var image=$.bbq.getState("image");
            var dir=image.substring(0,image.lastIndexOf("/"));
            window.location.href="#index="+encodeURIComponent($.bbq.getState("index"))+"&dir="+encodeURIComponent(dir);
            return false;
        } else if (e.which == 37 || e.which == 39) { // 37: left, 39: right
            nextImage(e.which-38);
            return false;
        }
        return true;
    }

    function handleFullscreenWheel(e) {
        var delta=e.originalEvent.deltaY;
        if (delta==0) return;
        nextImage(delta>0 ? 1 : -1);
    }
    $(document).off("keyup.fastview");
    $(document).off("wheel.fastview");
    if (page==="fullscreen") {
        $("#fullscreen-decoration").hide();
        $(document).on("keyup.fastview", handleFullscreenKeys);
        $(document).on("wheel.fastview", handleFullscreenWheel);
    }
}

function getThumbnailByIndex(index) {
    var imgEl=$("#thumbnails img:eq("+(index)+")");
    if (imgEl.length==0) return null;
    return imgEl;
}

function loadDir(dir, index) {
    console.log("loadDir", dir, index);
    var thumbnails=$("#thumbnails");
    var dirs=$("#directories");
    thumbnails.empty();
    dirs.empty();

    $.ajax({
      url: dir,
    })
    .done(function( data ) {
        //~ console.log( "Data loaded: ", data);
        $("#current-dir-name").text(dir);
        var list=JSON.parse(data);
        var imagelist=list.images;
        if (imagelist) {
            thumbnails.empty();
            imagelist.forEach(function(path, i) {
                //~ console.log("image path:", path);
                var filename=path.substring(path.lastIndexOf("/")+1);
                var img=$('<img />', {
                    src: path+"?size="+(thumbnailStyleSize>thumbnailSize ? fullscreenSize : thumbnailSize),
                    alt: filename,
                });
                img.css("max-height", thumbnailStyleSize+"px");
                img.css("max-width", thumbnailStyleSize+"px");
                img.click(function() {
                    window.location.href="#size="+fullscreenSize+"&index="+i+"&image="+encodeURIComponent(path);
                });
                var fnameLabel=$('<span/>', {
                    class: "filename-label"
                });
                fnameLabel.text(filename);
                var container=$('<div/>', {class: "thumbnail"});
                container.append(fnameLabel);
                container.append(img);

                thumbnails.append(container);
            });
        }

        function addDirLink(path, text) {
            console.log("addDirLink ", path, text);
            var link=$('<a />', {
                href: "#dir="+encodeURIComponent(path),
            });
            link.text(text ? text : path);
            dirs.append(link);
            dirs.append($('<br/>'));
        }

        dirs.empty();
        var dirlist=list.dirs;
        if (dir.lastIndexOf("/")>0) {
            var parent=dir.substring(0,dir.lastIndexOf("/"));
            if (parent.length>1) {
                addDirLink(parent, "..")
            }
        }
        if (dirlist) {
            // forEach unfortunately calls its param with 2 arguments :(
            dirlist.forEach(function(path) {addDirLink(path);});
        }
    });

    if (index>0) {
        var imgEl=$("#thumbnails img:eq("+(index)+")");

        var scroll=function() {
            var imgEl=$("#thumbnails img:eq("+(index)+")");
            if (imgEl.offset()==undefined) {
                console.log("wait for scroll...");
                window.requestAnimationFrame(scroll);
                return;
            }
            console.log("scroll!");
            $('html, body').animate({
                scrollTop: imgEl.offset().top
            }, 500);
        }
        scroll();
    }
}

// if at least one fullscreen image was served, then we don't hassle with the smaller ones
// (else some annoying flickering can be seen if both images showed in quick succession)
var dirsWithFullscreenImages={}

var imagesPrefetchCache=[]

imagesPrefetchCache.addItem=function(item) {
    if (!item || !item.url) return;
    for (var i=0; i<this.length; i++) {
        if (this[i].url===item.url) return;
    }
    if (this.length>=5) {
        this.shift();
    }
    this.push(item);
}

imagesPrefetchCache.getImg=function(url) {
    for (var i=0; i<this.length; i++) {
        if (this[i].url===url) return this[i].img;
    }
    return undefined;
}

function showImage(path, size, index) {
    var fullscreen=$("#fullscreen-img");
    fullscreen.empty();
    console.log("showImage("+path+", "+size+", "+index+")");

    var basedir=path.substring(0, path.lastIndexOf("/"));
    var thumb=getThumbnailByIndex(index);
    var srcPath=path+"?size="+size;

    function prefetchByIndex(ind) {
        if (ind<0) return undefined;
        var imgEl=$("#thumbnails img:eq("+(ind)+")");
        if (imgEl.length==1) {
            var imgPath=imgEl.attr("src").substring(0, imgEl.attr("src").indexOf("?"));
            var srcUrl=imgPath+"?size="+size;
            var img=$('<img />', {
                src: srcUrl,
                alt: imgPath.substring(imgPath.lastIndexOf("/")+1),
            });
            return {url: srcUrl, img: img};
        }
        return undefined;
    }

    // don't prefetch until we don't know if it is fast from server side
    if (dirsWithFullscreenImages[basedir]==true) {
        imagesPrefetchCache.addItem(prefetchByIndex(index-1));
        imagesPrefetchCache.addItem(prefetchByIndex(index+1));
        imagesPrefetchCache.addItem(prefetchByIndex(index+2));
    }

    var img=imagesPrefetchCache.getImg(srcPath);
    if (img && !img.get(0).complete) { // img.attr() is not in sync with the DOM current state
        img=undefined;
    }

    // we have thumb, but don't know if bigger pics are ready or we already prefetched the image, but it is not downloaded yet
    if (thumb && thumb.attr("src")!==srcPath && !img) {
        var tmpimg=$('<img />', {
            src: thumb.attr("src"),
            alt: path.substring(path.lastIndexOf("/")+1),
        });
        doit(tmpimg, false);
    }

    if (!img) {
        img=$('<img />', {
            src: srcPath,
            alt: path.substring(path.lastIndexOf("/")+1),
        });
    }
    doit(img, true);

    function pressEsc() {
        var e = jQuery.Event("keyup");
        e.which = 27;
        $(document).trigger(e);
    }

    function doit(img, isFinal) {
            //~ console.log("doit called: ",isFinal, img.attr("src"));
        // this is ok for not-so-narrow browser sizes; otherwise should do something like in
        // http://stackoverflow.com/questions/20590239/maintain-aspect-ratio-of-div-but-fill-screen-width-and-height-in-css
        //img.photoResize({bottomSpacing: 0});
        img.on("error", pressEsc);
        img.on("load", function() { // prevent flickering
            //~ console.log("load called: ",isFinal, img.attr("src"));
            // the bigger (final) img or the bigger one not yet displayed,
            // AND it is the one what is currently in the url (could be navigated away)
            if ((isFinal || fullscreen.children().length===0) && (index===parseInt($.bbq.getState("index")) || index==-1)) {
                fullscreen.empty();
                fullscreen.append(img);
            }
            if (isFinal) {
                dirsWithFullscreenImages[basedir]=true;
            }
            img.click(pressEsc);
        });
        if (img.get(0).complete) { // for prefetched images
            img.trigger("load");
        }
    }
}
