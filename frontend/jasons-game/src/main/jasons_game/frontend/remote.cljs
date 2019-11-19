(ns jasons-game.frontend.remote
  (:require [clojure.walk :refer [keywordize-keys]]
            [re-frame.core :as re-frame]
            [re-frame.db :refer [app-db]]
            ["@improbable-eng/grpc-web" :as grpc-lib :refer [grpc]]
            ["/frontend/remote/jasonsgame_pb" :as game-lib]
            ["/frontend/remote/jasonsgame_pb_service" :as game-service :refer [GameService]]))

(goog-define dev-host false)

(defonce default-host
  (if dev-host
    dev-host
    (-> js/window .-location .-origin)))

(def unary (.-unary grpc))
(def invoke (.-invoke grpc))

(def game-send-command (.-SendCommand GameService))
(def game-receive-usermessages (.-ReceiveUIMessages GameService))

(def ui-message-types
  (-> game-lib/jasonsgame.UserInterfaceMessage .-UiMessageCase js->clj keywordize-keys))

(defn user-message? [msg]
  (-> msg
      .getUiMessageCase
      (= (:USER_MESSAGE ui-message-types))))

(defn command-update? [msg]
  (-> msg
      .getUiMessageCase
      (= (:COMMAND_UPDATE ui-message-types))))

(defn new-session [id]
  (doto (game-lib/jasonsgame.Session.)
    (.setUuid id)))

(defn send-user-input [host session input callback]
  (let [req (doto (game-lib/jasonsgame.UserInput.)
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

(defn resp->message [resp]
  (-> resp
      .toObject
      js->clj
      keywordize-keys))

(defn handle-user-message [resp]
  (let [msg (.getUserMessage resp)]
    (if (.getHeartbeat msg)
      (re-frame/dispatch [:user/heartbeat])
      (do
        (.log js/console "user message" (.toObject msg))
        (let [game-msg (-> msg
                           resp->message
                           (assoc :user false))]
          (re-frame/dispatch [:user/message game-msg]))))))

(defn handle-command-update [resp]
  (let [msg (.getCommandUpdate resp)]
    (let [cmd-update (-> msg
                         resp->message
                         :commandsList)]
      (re-frame/dispatch [:command/update cmd-update]))))

(defn handle-game-message [resp]
  (let [msg-type (.getUiMessageCase resp)]
    (cond
      (user-message? resp) (handle-user-message resp)
      (command-update? resp) (handle-command-update resp)
      :default (.log js/console "unrecognized message type" msg-type))))

(defn handle-game-end [resp]
  (.log js/console "game end, redoing" resp)
  (re-frame/dispatch [::game-end]))

(re-frame/reg-fx
 ::listen
 (fn [{:keys [host session]}]
   (let [req (start-game-listener host session
                                  handle-game-message handle-game-end)]
     (swap! app-db assoc ::current-listener req))))

(re-frame/reg-fx
 ::send-input
 (fn [{:keys [host session command]}]
   (send-user-input host session command
                    (fn [resp] (.log js/console resp)))))

(re-frame/reg-sub
 ::messages
 (fn [{::keys [messages] :as db} _]
   messages))
