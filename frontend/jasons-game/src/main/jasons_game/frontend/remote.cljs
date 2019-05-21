(ns jasons-game.frontend.remote
  (:require
   ["@improbable-eng/grpc-web" :as grpc-lib :refer [grpc]]
   ["/frontend/remote/jasonsgame_pb" :as game-lib]
   ["/frontend/remote/jasonsgame_pb_service" :as game-service :refer [GameService]]))

(goog-define dev-host false)

(defonce default-host
  (if dev-host
    dev-host
    (-> js/window (.-location) (.-origin))))

(def unary (.-unary grpc))
(def invoke (.-invoke grpc))

(def game-send-command (.-SendCommand GameService))
(def game-receive-usermessages (.-ReceiveUserMessages GameService))

(defn new-session [id]
  (doto (game-lib/Session.)
    (.setUuid id)))

(defn send-user-input [host session input callback]
  (let [req (doto (game-lib/UserInput.)
              (.setMessage input)
              (.setSession session))]
    (unary game-send-command (clj->js {:request req
                                       :host host
                                       :onEnd callback}))))

(defn start-game-listener [host session on-message on-end]
  (invoke game-receive-usermessages (clj->js {:request session
                                              :host host
                                              :onMessage on-message
                                              :onEnd on-end})))
