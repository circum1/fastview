"use strict";

var thumbnailSize=640;
var fullscreenSize=1600;

var thumbnailStyleSize=320;

$(document).ready(function() {
    console.log("document.ready()");
    $("#testbtn").click(function() {
       var thumbnails=$("#thumbnails");
        thumbnails.empty();
    });

    $("#thumb-size").on("keydown", function(e) {
        if (e.keyCode == 13) {
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
        var dir=$.bbq.getState("dir");
        if (dir) {
            selectPage("thumbnails");
            loadDir(dir, $.bbq.getState("index"));
            return;
        }
        var image=$.bbq.getState("image");
        if (image) {
            if ($("#thumbnails").children().length==0) {
                // presumably opened a direct link to an image -- must load the thumbnails...
                loadDir(image.substring(0,image.lastIndexOf("/")));
            }
            selectPage("fullscreen");
            showImage(image);
            return;
        }
    });

    if (window.location.href.indexOf("#")<0) {
        //~ window.location+="#dir=/pictures/";
        $.bbq.pushState({"dir": "/pictures"});
    }

    $(window).trigger( 'hashchange' );
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
        if (e.keyCode == 27) { // escape key maps to keycode `27`
            var image=$.bbq.getState("image");
            var dir=image.substring(0,image.lastIndexOf("/"));
            window.location.href="#index="+encodeURIComponent($.bbq.getState("index"))+"&dir="+encodeURIComponent(dir);
            return false;
        } else if (e.keyCode == 37 || e.keyCode == 39) { // 37: left, 39: right
            nextImage(e.keyCode-38);
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
        $(document).on("keyup.fastview", handleFullscreenKeys);
        $(document).on("wheel.fastview", handleFullscreenWheel);
    }
}

function loadDir(dir) {
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
            imagelist.forEach(function(path, i) {
                //~ console.log("image path:", path);
                var img=$('<img />', {
                    src: path+"?size="+(thumbnailStyleSize>thumbnailSize ? fullscreenSize : thumbnailSize),
                    alt: path.substring(path.lastIndexOf("/")+1),
                    class: "thumbnail"
                });
                img.css("padding", "10px");
                img.css("max-height", thumbnailStyleSize+"px");
                img.css("max-width", thumbnailStyleSize+"px");
                img.click(function() {
                    window.location.href="#size="+fullscreenSize+"&index="+i+"&image="+encodeURIComponent(path);
                });
                //~ var span=$('<span/>');
                //~ span.append(img);
                thumbnails.append(img);
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

    if (parseInt($.bbq.getState("index"))>0) {
        var index=parseInt($.bbq.getState("index"), 10);
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

function showImage(path) {
    var fullscreen=$("#fullscreen-page");
    fullscreen.empty();
    console.log("showImage("+path+")");

    var img=$('<img />', {
        src: path+"?size="+$.bbq.getState("size"),
        alt: path.substring(path.lastIndexOf("/")+1),
    });
    // this is ok for not-so-narrow browser sizes; otherwise should do something like in
    // http://stackoverflow.com/questions/20590239/maintain-aspect-ratio-of-div-but-fill-screen-width-and-height-in-css
    img.photoResize({bottomSpacing: 0});
    img.on("load", function() { // prevent flickering
        fullscreen.append(img);
    });
    img.click(function() {
        window.location.href=path;
    });
}