type WebSocketEvent = Event;//Event | CloseEvent | MessageEvent;
type SocketCallback = (socket:Socket, event:WebSocketEvent) => void;
type SocketEvent = { event:string, data:any }

class Socket {

    public static STATE_CONNECTING = 0;
    public static STATE_OPEN = 1;
    public static STATE_CLOSING = 2;
    public static STATE_CLOSED = 3;

    private webSocket:WebSocket
    private callbacks:Map<string, any>

    constructor(path:string) {
        const self = this;
        this.webSocket = new WebSocket(path);
        this.webSocket.addEventListener("message", function (this: WebSocket, event: MessageEvent) {
            self._handleMessage(this, event);
        });
        this.callbacks = new Map<string, any>();
    }

    onConnect(callback:SocketCallback) {
        const self = this;
        this.webSocket.addEventListener("open", function (this: WebSocket, event: Event) {
            callback(self, event);
        });
    }

    onMessage(callback:SocketCallback) {
        const self = this;
        this.webSocket.addEventListener("message", function(this:WebSocket, event:Event) {
            callback(self, event);
        });
    }

    onEvent<T>(eventName:string, callback:(socket:Socket, data:T) => void) {
        this.callbacks[eventName] = callback;
    }

    send<T>(event:string, data:T) {
        const text = JSON.stringify({ name: event, data: data });
        this.webSocket.send(text);
    }

    close(code?:number, reason?:string) {
        this.webSocket.close(code, reason);
    }

    readyState():number {
        return this.webSocket.readyState;
    }

    private _handleMessage(webSocket: WebSocket, event: MessageEvent) {
        try {
            const reader = new FileReader();
            reader.addEventListener('loadend', e => {
                this._messageParsed(webSocket, e.target!.result as string);
            });
            reader.readAsText(event.data as Blob);
        } catch {
            // data isn't a blob, it must be a string. Pass it directly.
            this._messageParsed(webSocket, event.data as string);
        }
    }

    private _messageParsed(webSocket: WebSocket, jsonString:string) {
        const obj = JSON.parse(jsonString) as SocketEvent;
        const eventName = obj["name"];
        const callback = this.callbacks[eventName];
        if (callback == undefined) {
            return;
        }
        callback(this, obj.data);
    }
}