export var LogLevel;
(function (LogLevel) {
    LogLevel[LogLevel["SILENT"] = 0] = "SILENT";
    LogLevel[LogLevel["ERROR"] = 1] = "ERROR";
    LogLevel[LogLevel["WARN"] = 2] = "WARN";
    LogLevel[LogLevel["INFO"] = 3] = "INFO";
    LogLevel[LogLevel["DEBUG"] = 4] = "DEBUG";
})(LogLevel || (LogLevel = {}));
;
export function LogLevelString(logLevel) {
    switch (logLevel) {
        case LogLevel.SILENT: return "SILENT";
        case LogLevel.ERROR: return "ERROR";
        case LogLevel.WARN: return "WARN";
        case LogLevel.INFO: return "INFO";
        case LogLevel.DEBUG: return "DEBUG";
        default: throw new Error("Unknown log level");
    }
}
;
export function recursiveLoggerID(logger) {
    var ids = [];
    var parent = logger.parent();
    while (parent != null) {
        ids.push(parent.id());
        parent = parent.parent();
    }
    ids.push(logger.id());
    ids.reverse();
    return ids.join(' ').trim();
}
export function recursiveLoggerLogLevel(logger) {
    var logLevel = logger.logLevel();
    var parent = logger.parent();
    while (parent != null) {
        logLevel = Math.min(logLevel, parent.logLevel());
        parent = parent.parent();
    }
    return logLevel;
}
var InMemoryLogger = /** @class */ (function () {
    function InMemoryLogger(_id, _parent) {
        this._id = _id;
        this._parent = _parent;
        this._logLevel = LogLevel.ERROR;
        this.dateFormatter = function (timestamp) {
            return "".concat(timestamp.getHours(), ":").concat(timestamp.getMinutes(), ":").concat(timestamp.getSeconds(), ".").concat(timestamp.getMilliseconds());
        };
        this._logs = [];
        this._childLoggers = [];
        if (_parent != null) {
            _parent._childLoggers.push(this);
            this._logLevel = _parent._logLevel;
        }
    }
    InMemoryLogger.prototype.id = function () {
        return this._id;
    };
    InMemoryLogger.prototype.parent = function () {
        return this._parent || null;
    };
    InMemoryLogger.prototype.logLevel = function () {
        return this._logLevel;
    };
    InMemoryLogger.prototype.setLogLevel = function (level) {
        this._logLevel = level;
        for (var _i = 0, _a = this._childLoggers; _i < _a.length; _i++) {
            var child = _a[_i];
            child.setLogLevel(level);
        }
    };
    InMemoryLogger.prototype.emitLog = function (logLevel, log) {
        var thisLogLevel = recursiveLoggerLogLevel(this);
        if (thisLogLevel == LogLevel.SILENT || logLevel > thisLogLevel) {
            return;
        }
        var id = recursiveLoggerID(this);
        if (id != "") {
            id += " ";
        }
        var timestamp = new Date();
        var timeStr = this.dateFormatter(timestamp);
        var logLevelStr = LogLevelString(logLevel);
        var logStatement = {
            id: this._id,
            // space is included in if statement above.
            // no id == no leading space.
            log: "".concat(id, "[").concat(logLevelStr, "][").concat(timeStr, "]: ").concat(log),
            logLevel: logLevel,
            timestamp: timestamp,
        };
        ;
        if (this._parent != undefined) {
            this._parent.logs().push(logStatement);
        }
        else {
            this._logs.push(logStatement);
        }
    };
    InMemoryLogger.prototype.childLogger = function (childID) {
        return new InMemoryLogger(childID, this);
    };
    InMemoryLogger.prototype.logs = function () {
        if (this._parent != undefined) {
            return this._parent.logs();
        }
        return this._logs;
    };
    InMemoryLogger.prototype.clearLogs = function () {
        this._logs = [];
        for (var _i = 0, _a = this._childLoggers; _i < _a.length; _i++) {
            var child = _a[_i];
            child.clearLogs();
        }
    };
    return InMemoryLogger;
}());
export { InMemoryLogger };
var ConsoleLogger = /** @class */ (function () {
    function ConsoleLogger(_id, _parent) {
        this._id = _id;
        this._parent = _parent;
        this._logLevel = LogLevel.ERROR;
        this.dateFormatter = function (timestamp) {
            return "".concat(timestamp.getHours(), ":").concat(timestamp.getMinutes(), ":").concat(timestamp.getSeconds(), ".").concat(timestamp.getMilliseconds());
        };
        this._logs = [];
        if (_parent != null) {
            this._logLevel = _parent._logLevel;
        }
    }
    ConsoleLogger.prototype.id = function () {
        return this._id;
    };
    ConsoleLogger.prototype.parent = function () {
        return this._parent || null;
    };
    ConsoleLogger.prototype.logLevel = function () {
        return this._logLevel;
    };
    ConsoleLogger.prototype.setLogLevel = function (level) {
        this._logLevel = level;
    };
    ConsoleLogger.prototype.emitLog = function (logLevel, log) {
        var thisLogLevel = recursiveLoggerLogLevel(this);
        if (thisLogLevel == LogLevel.SILENT || logLevel > thisLogLevel) {
            return;
        }
        var id = recursiveLoggerID(this);
        var timestamp = new Date();
        var timeStr = this.dateFormatter(timestamp);
        var logLevelStr = LogLevelString(logLevel);
        console.log("".concat(id, " [").concat(logLevelStr, "][").concat(timeStr, "]: ").concat(log));
    };
    ConsoleLogger.prototype.childLogger = function (childID) {
        return new ConsoleLogger(childID, this);
    };
    return ConsoleLogger;
}());
export { ConsoleLogger };
