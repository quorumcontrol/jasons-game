(ns jasons-game.frontend.remote.game
  (:require
   ["@improbable-eng/grpc-web" :as grpc-lib :refer (grpc)]
   ["/frontend/remote/jasonsgame_pb" :as game-lib]
   ["/frontend/remote/jasonsgame_pb_service" :as game-service :refer [GameService]]))

(def unary (.-unary grpc))
(def invoke (.-invoke grpc))

(def game-send-command (.-SendCommand GameService))
(def game-receive-usermessages (.-ReceiveUserMessages GameService))

(defn new-session [id]
  (let [req (game-lib/Session.)]
    (.setUuid req id)
    req))

(defn send-user-input [host session input callback]
  (let [req (game-lib/UserInput.)]
    (.setMessage req input)
    (.setSession req session)
    (unary game-send-command (clj->js {:request req
                                       :host host
                                       :onEnd callback}))))

(defn start-game-listener [host session on-message on-end]
  (invoke game-receive-usermessages (clj->js {:request session
                                              :host host
                                              :onMessage on-message
                                              :onEnd on-end})))



; var client = grpc.invoke(GameService.ReceiveUserMessages, {
;     request: requestMessage,
;     host: this.serviceHost,
;     metadata: metadata,
;     transport: this.options.transport,
;     debug: this.options.debug,
;     onMessage: function (responseMessage) {
;       listeners.data.forEach(function (handler) {
;         handler(responseMessage);
;       });
;     },
;     onEnd: function (status, statusMessage, trailers) {
;       listeners.end.forEach(function (handler) {
;         handler();
;       });
;       listeners.status.forEach(function (handler) {
;         handler({ code: status, details: statusMessage, metadata: trailers });
;       });
;       listeners = null;
;     }
;   });