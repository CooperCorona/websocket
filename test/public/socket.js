import { LogLevel } from "./logger.js";
var Socket = /** @class */ (function () {
    function Socket(path) {
        this.logger = null;
        var self = this;
        this.webSocket = new WebSocket(path);
        this.webSocket.addEventListener("message", function (event) {
            self._handleMessage(this, event);
        });
        this.webSocket.addEventListener("error", function (event) {
            self._handleError(this, event);
        });
        this.webSocket.addEventListener("close", function (event) {
            self._handleClose(this, event);
        });
        this.callbacks = new Map();
        this.errorCallbacks = new Map();
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
        this.callbacks.set(eventName, callback);
    };
    Socket.prototype.onError = function (callback) {
        this.errorCallbacks.set("error", callback);
    };
    Socket.prototype.onClose = function (callback) {
        this.errorCallbacks.set("close", callback);
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
    Socket.prototype._handleError = function (webSocket, event) {
        var _a;
        (_a = this.logger) === null || _a === void 0 ? void 0 : _a.emitLog(LogLevel.ERROR, "WebSocket error occurred");
        var callback = this.errorCallbacks.get("error");
        if (callback) {
            callback(this, event);
        }
    };
    Socket.prototype._handleClose = function (webSocket, event) {
        var _a;
        (_a = this.logger) === null || _a === void 0 ? void 0 : _a.emitLog(LogLevel.DEBUG, "WebSocket closed with code: ".concat(event.code, ", reason: ").concat(event.reason));
        // Handle any close events
        var callback = this.errorCallbacks.get("close");
        if (callback) {
            callback(this, event);
        }
    };
    Socket.prototype._messageParsed = function (webSocket, jsonString) {
        var _a, _b;
        var obj = JSON.parse(jsonString);
        var callback = this.callbacks.get(obj.name);
        if (callback == undefined) {
            (_a = this.logger) === null || _a === void 0 ? void 0 : _a.emitLog(LogLevel.DEBUG, "No callback for ".concat(obj.name));
            return;
        }
        (_b = this.logger) === null || _b === void 0 ? void 0 : _b.emitLog(LogLevel.DEBUG, "Handling event ".concat(obj.name));
        callback(this, obj.data);
    };
    Socket.STATE_CONNECTING = 0;
    Socket.STATE_OPEN = 1;
    Socket.STATE_CLOSING = 2;
    Socket.STATE_CLOSED = 3;
    return Socket;
}());
export { Socket };
