$(function() {
    var username = pstfgrpnt(true).join('');
    var conn;
    var msg = $("#msg");
    var log = $("#log");

    $.cookie('online_from', username, {
        expires: 7,
        path: '/'
    });

    function appendLog(msg) {
        var d = log[0]
        var doScroll = d.scrollTop == d.scrollHeight - d.clientHeight;
        msg.appendTo(log)
        if (doScroll) {
            d.scrollTop = d.scrollHeight - d.clientHeight;
        }
    }

    $(document).on('submit', '#form', function(e) {
        e.preventDefault();

        if (!conn) {
            return false;
        }
        if (!msg.val()) {
            return false;
        }
        var data = JSON.stringify({
            from: username,
            message: msg.val()
        });

        console.log(data);
        conn.send(data);

        msg.val("");

        return false
    });

    if (window["WebSocket"]) {
        conn = new WebSocket("ws://localhost:8080/ws");
        conn.onclose = function(e) {
            appendLog($("<div><b>Connection closed.</b></div>"))
        }
        conn.onmessage = function(e) {
            var data = jQuery.parseJSON(e.data),
                from = data.from == username ? "Вы" : data.from,
                className = data.from == username ? "message-item" : "message-item reply",
                tpl = "<p class='from'>" + from + "</p>" +
                    "<p class='message'>" + data.message + "</p>";

            appendLog($("<div/>").addClass(className).html(tpl));
        }
    } else {
        appendLog($("<div><b>Your browser does not support WebSockets.</b></div>"))
    }
});
