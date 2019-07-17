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
(def game-import-command (.-Import GameService))
(def game-receive-usermessages (.-ReceiveUIMessages GameService))

(def ui-message-types
  (-> game-lib/UserInterfaceMessage .-UiMessageCase js->clj keywordize-keys))

(defn user-message? [msg]
  (-> msg
      .getUiMessageCase
      (= (:USER_MESSAGE ui-message-types))))

(defn command-update? [msg]
  (-> msg
      .getUiMessageCase
      (= (:COMMAND_UPDATE ui-message-types))))

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

(defn new-object-spec [{:keys [description inscriptions]}]
  (doto (game-lib/ObjectSpec.)
    (.setDescription description)
    (.setInscriptionsList (clj->js inscriptions))))

(defn add-objects [import-req objects]
  (let [obj-map (.getObjectsMap import-req)]
    (reduce (fn [m obj]
              (.set m (:name obj) (new-object-spec obj)))
            obj-map objects)))

(defn set-location-spec-links [s links]
  (reduce (fn [lnks {:keys [name location] :as lnk}]
            (.set lnks name location))
          (.getLinksMap s) links))

(defn new-location-spec [{:keys [description links objects]}]
  (doto (game-lib/LocationSpec.)
    (.setDescription description)
    (set-location-spec-links links)
    (.setObjectsList (clj->js objects))))

(defn add-locations [import-req locations]
  (let [loc-map (.getLocationsMap import-req)]
    (reduce (fn [m loc]
              (.set m (:name loc) (new-location-spec loc)))
            loc-map locations)))

(defn new-import-request [session]
  (doto (game-lib/ImportRequest.)
    (.setSession session)))

(defn new-import-location-request [session location-spec]
  (doto (new-import-request session)
    (.setLocation (clj->js location-spec))))

(defn new-import-object-request [session object-spec]
  (doto (new-import-request session)
    (.setObject (clj->js object-spec))))

(defn send-import-request [host req callback]
  (unary game-import-command (clj->js {:request req
                                       :host host
                                       :onEnd callback})))

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
    (if (not (.getHeartbeat msg))
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
  (re-frame/dispatch [::listen]))

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

(re-frame/reg-fx
 ::import-object
 (fn [{:keys [host session spec]}]
   (.log js/console (str "sending import object request with spec: " spec))
   (let [req (new-import-object-request session spec)]
     (send-import-request host req (fn [resp] (.log js/console resp))))))

(re-frame/reg-fx
 ::import-location
 (fn [{:keys [host session spec]}]
   (.log js/console (str "sending import location request with spec: " spec))
   (let [req (new-import-location-request session spec)]
     (send-import-request host req (fn [resp] (.log js/console resp))))))
