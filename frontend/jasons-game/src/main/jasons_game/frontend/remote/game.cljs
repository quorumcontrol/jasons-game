(ns jasons-game.frontend.remote.game
  (:require
   ["@improbable-eng/grpc-web" :as grpc-lib :refer [grpc]]
   ["/frontend/remote/jasonsgame_pb" :as game-lib]
   ["/frontend/remote/jasonsgame_pb_service" :as game-service :refer [GameService]]))

(def unary (.-unary grpc))
(def invoke (.-invoke grpc))

(def game-send-command (.-SendCommand GameService))
(def game-receive-usermessages (.-ReceiveUserMessages GameService))

(defn new-session [id]
  (doto (game-lib/Session.)
    (.setUuid id)))

(defn start-game-listener [host session on-message on-end]
  (invoke game-receive-usermessages (clj->js {:request session
                                              :host host
                                              :onMessage on-message
                                              :onEnd on-end})))
