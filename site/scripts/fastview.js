 "use strict";

$(document).ready(function() {
    console.log("initFastview()");

    $("#testbtn").click(function() {
       var thumbnails=$("#thumbnails");
        thumbnails.empty();
    })

    $(document).on("click", "#directories a", function(e) {
        $.bbq.pushState({dir: $(this).attr('href')});
        return false;
    });

    $(window).bind( 'hashchange', function(e) {
        var dir=$.bbq.getState("dir");
        if (dir) {
            loadDir(dir);
        }
    });

    if (window.location.href.indexOf("#")<0) {
        window.location+="#dir=/localfs/";
    }

    $(window).trigger( 'hashchange' );
});

function loadDir(dir) {
    $.ajax({
      url: dir,
    })
    .done(function( data ) {
        console.log( "Data loaded: ", data);
        $("#current-dir-name").text(dir);
        var list=JSON.parse(data);
        var imagelist=list.images;
        if (imagelist) {
            var thumbnails=$("#thumbnails");
            thumbnails.empty();
            imagelist.forEach(function(path) {
                console.log("image path:", path);
                var img=$('<img />', {
                    src: path,
                    alt: path.substring(path.lastIndexOf("/")+1),
                    class: "thumbnail"
                });
                img.css("padding", "10px");
                img.click(function() {
                    window.location.href=path;
                });
                var span=$('<span/>');
                span.append(img);
                thumbnails.append(span);
            });
        }

        var dirlist=list.dirs;
        if (dirlist) {
            var dirs=$("#directories");
            dirs.empty();
            dirlist.forEach(function(path) {
                console.log("dir path:", path);
                var link=$('<a />', {
                    href: path,
                });
                link.text(path);
                //~ var span=$('<span/>', {
                //~ });
                //~ span.append(img);
                dirs.append(link);
                dirs.append($('<br/>'));
            });
        }

    });
}
