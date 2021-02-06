package websocket

var socketJsContents = `var Socket = /** @class */ (function () {
    function Socket(path) {
        var self = this;
        this.webSocket = new WebSocket(path);
        this.webSocket.addEventListener("message", function (event) {
            self._handleMessage(this, event);
        });
        this.callbacks = new Map();
    }
    Socket.prototype.onConnect = function (callback) {
        var self = this;
        this.webSocket.addEventListener("open", function (event) {
            callback(self, event);
        });
    };
    Socket.prototype.onMessage = function (callback) {
        var self = this;
        this.webSocket.addEventListener("message", function (event) {
            callback(self, event);
        });
    };
    Socket.prototype.onEvent = function (eventName, callback) {
        this.callbacks[eventName] = callback;
    };
    Socket.prototype.send = function (event, data) {
        var text = JSON.stringify({ name: event, data: data });
        this.webSocket.send(text);
    };
    Socket.prototype.close = function (code, reason) {
        this.webSocket.close(code, reason);
    };
    Socket.prototype.readyState = function () {
        return this.webSocket.readyState;
    };
    Socket.prototype._handleMessage = function (webSocket, event) {
        var _this = this;
        try {
            var reader = new FileReader();
            reader.addEventListener('loadend', function (e) {
                _this._messageParsed(webSocket, e.target.result);
            });
            reader.readAsText(event.data);
        }
        catch (_a) {
            // data isn't a blob, it must be a string. Pass it directly.
            this._messageParsed(webSocket, event.data);
        }
    };
    Socket.prototype._messageParsed = function (webSocket, jsonString) {
        var obj = JSON.parse(jsonString);
        var eventName = obj["name"];
        var callback = this.callbacks[eventName];
        if (callback == undefined) {
            return;
        }
        callback(this, obj.data);
    };
    Socket.STATE_CONNECTING = 0;
    Socket.STATE_OPEN = 1;
    Socket.STATE_CLOSING = 2;
    Socket.STATE_CLOSED = 3;
    return Socket;
}());
`

