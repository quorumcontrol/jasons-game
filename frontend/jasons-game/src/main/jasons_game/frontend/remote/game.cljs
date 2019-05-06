(ns jasons-game.frontend.remote.game
  (:require
   ["@improbable-eng/grpc-web" :as grpc-lib :refer (grpc)]
   ["/frontend/remote/jasonsgame_pb" :as game-lib]
   ["/frontend/remote/jasonsgame_pb_service" :as game-service :refer [GameService]]))

(def unary (.-unary grpc))

(def game-send-command (.-SendCommand GameService))

(defn send-user-input [host input callback]
  (let [req (game-lib/UserInput.)]
    (.setMessage req input)
    (unary game-send-command (clj->js {:request req
                                       :host host
                                       :onEnd callback}))))


; var client = grpc.unary(GameService.SendCommand, {
;     request: requestMessage,
;     host: this.serviceHost,
;     metadata: metadata,
;     transport: this.options.transport,
;     debug: this.options.debug,
;     onEnd: function (response) {
;       if (callback) {
;         if (response.status !== grpc.Code.OK) {
;           var err = new Error(response.statusMessage);
;           err.code = response.status;
;           err.metadata = response.trailers;
;           callback(err, null);
;         } else {
;           callback(null, response.message);
;         }
;       }
;     }
;   });
;   return {
;     cancel: function () {
;       callback = null;
;       client.close();
;     }
;   };