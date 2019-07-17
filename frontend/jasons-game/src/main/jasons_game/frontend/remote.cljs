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

(defn import-result? [msg]
  (-> msg
      .getUiMessageCase
      (= (:IMPORT_RESULT ui-message-types))))

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

(defn new-location-spec [{:keys [description links objects]}]
  (doto (game-lib/LocationSpec.)
    (.setDescription description)
    (.setObjectsList (clj->js objects))))

(defn add-locations [import-req locations]
  (let [loc-map (.getLocationsMap import-req)]
    (reduce (fn [m loc]
              (.set m (:name loc) (new-location-spec loc)))
            loc-map locations)))

(defn new-import-request [session]
  (doto (game-lib/ImportRequest.)
    (.setSession session)))

(defn increment-drop-phase [{:keys [objects] :as pop-spec}]
  (if (seq objects)
    (-> pop-spec
        (assoc :phase :drop)
        (assoc :objects (rest objects)))
    (assoc pop-spec :phase :connect)))

(defn increment-connect-phase [{:keys [links] :as pop-spec}]
  (if (seq links)
    (-> pop-spec
        (assoc :phase :connect)
        (assoc :phase :done))))

(defn set-populate-phase [pop-spec]
  (let [old-phase (:phase pop-spec)]
    (case old-phase
      :visit (assoc pop-spec :phase :describe)
      :describe (assoc pop-spec :phase :drop)
      :drop (increment-drop-phase pop-spec)
      :connect (increment-connect-phase pop-spec)
      (assoc pop-spec :phase :visit))))

(defn set-phase [{:keys [unpopulated-locations] :as status}]
  (let [{:keys [phase]} unpopulated-locations
        new-spec (if (nil? phase)
                   (assoc unpopulated-locations :phase :visit)
                   (set-populate-phase unpopulated-locations))]
    (assoc status :unpopulated-locations new-spec)))

(defn new-populate-phase [{:keys [phase] :as pop-spec}]
  (case phase
    :visit (game-lib/PopulateVisitPhase.)
    :describe (doto (game-lib/PopulateDescribePhase.)
                (.setDescription (:description pop-spec)))
    :drop (doto (game-lib/PopulateDropObjectsPhase.))))

(defn get-created-did [created created-name]
  (->> created
       (filter #(= (:name %) created-name))
       first
       :did))

(defn new-drop-proto [{:keys [created-objects unpopulated-locations] :as status}]
  (let [{:keys [objects] :as pop-spec} (first unpopulated-locations)
        object-name (first objects)
        did (get-created-did created-objects object-name)]
    (doto (game-lib/PopulateDropObjectsPhase.)
      (.setObjectName object-name)
      (.setObjectDid did))))

(defn new-connect-proto [{:keys [populated-locations unpopulated-locations] :as status}]
  (let [{:keys [links] :as pop-spec} (first unpopulated-locations)
        location-name (-> links first :location)
        link-name (-> links first :name)
        did (get-created-did (concat unpopulated-locations populated-locations)
                              location-name)]
    (doto (game-lib/PopulateDropObjectsPhase.)
      (.setConnectionName link-name)
      (.setToDid did))))

(defn set-phase-proto [spec {:keys [current] :as status} phase]
  (.log js/console (str "current:  " current))
  (.log js/console (str "spec: " spec))
  (let [pop-spec (second current)]
    (case phase
      :describe (.setDescribe spec (doto (game-lib/PopulateDescribePhase.)
                                     (.setDescription (:description pop-spec))))
      :drop (.setDrop spec (new-drop-proto pop-spec))
      :connect (.setConnect spec (new-connect-proto pop-spec))
      (.setVisit spec (game-lib/PopulateVisitPhase.))))
  spec)

(defn status->populate-proto [{:keys [pop-spec] :as status}]
  (let [{:keys [name did phase]} pop-spec]
    (doto (game-lib/PopulateSpec.)
      (.setName name)
      (.setDid did)
      (set-phase-proto status phase))))

(defn new-populate-location-request [session status]
  (let [new-status (set-phase status)
        pop-spec-proto (status->populate-proto new-status)]
    (doto (new-import-request session)
      (.setPopulate pop-spec-proto))))

(defn loc-spec->proto [{:keys [name] :as loc-spec}]
  (doto (game-lib/LocationSpec.)
    (.setName name)))

(defn new-import-location-request [session location-spec]
  (let [loc-spec-proto (loc-spec->proto location-spec)]
    (doto (new-import-request session)
      (.setLocation loc-spec-proto))))

(defn obj-spec->proto [{:keys [name description] :as obj-spec}]
  (doto (game-lib/ObjectSpec.)
    (.setName name)
    (.setDescription description)))

(defn new-import-object-request [session object-spec]
  (let [obj-spec-proto (obj-spec->proto object-spec)]
    (doto (new-import-request session)
      (.setObject obj-spec-proto))))

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

(defn handle-import-result [resp]
  (let [msg (.getImportResult resp)]
    (let [import-result (-> msg
                            resp->message)]
      (.log js/console (str "import-result: " import-result))
      (re-frame/dispatch [::import-result import-result]))))

(defn handle-game-message [resp]
  (let [msg-type (.getUiMessageCase resp)]
    (cond
      (user-message? resp) (handle-user-message resp)
      (command-update? resp) (handle-command-update resp)
      (import-result? resp) (handle-import-result resp)
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

(re-frame/reg-fx
 ::populate-location
 (fn [{:keys [host session spec]}]
   (.log js/console (str "sending populate location request with spec: " spec))
   (let [req (new-populate-location-request session spec)]
     (send-import-request host req (fn [resp] (.log js/console resp))))))

(defn import-next-object [{::keys [host session import-status] :as db}]
  (let [{:keys [pending-objects]} import-status
        next (first pending-objects)
        current [::import-object next]
        new-pending-objects (rest pending-objects)
        new-status (-> import-status
                       (assoc :pending-objects new-pending-objects)
                       (assoc :current current))]
    {:db (assoc db ::import-status new-status)
     ::import-object {:host host, :session session, :spec next}}))

(defn import-next-location [{::keys [host session import-status] :as db}]
  (let [{:keys [pending-locations]} import-status
        next (first pending-locations)
        current [::import-location next]
        new-pending-locations (rest pending-locations)
        new-status (-> import-status
                       (assoc :pending-locations new-pending-locations)
                       (assoc :current current))]
    {:db (assoc db ::import-status new-status)
     ::import-location {:host host, :session session, :spec next}}))

(defn populate-next-location [{::keys [host session import-status] :as db}]
  (let [{:keys [unpopulated-locations]} import-status
        next (first unpopulated-locations)
        current [::populate-location next]
        new-unpopulated-locations (rest unpopulated-locations)
        new-status (-> import-status
                       (assoc :unpopulated-locations new-unpopulated-locations)
                       (assoc :current current))]
    {:db (assoc db ::import-status new-status)
     ::populate-location {:host host, :session session, :spec next}}))

(defn handle-import [{{:keys [pending-objects pending-locations
                              unpopulated-locations]} ::import-status
                      :as db}]
  (cond
    (seq pending-objects) (import-next-object db)
    (seq pending-locations) (import-next-location db)
    (seq unpopulated-locations) (populate-next-location db)))

(re-frame/reg-event-fx
 ::import
 (fn [{db :db} _]
   (.log js/console (str "import status: " (::import-status db)))
   (handle-import db)))

(defn update-import-object-status [{:keys [created-objects current] :as status}
                                   {:keys [object] :as import-result}]
  (let [create-spec (second current)
        new-created-object (merge create-spec object)
        new-created-objects (conj created-objects object)]
    (-> status
        (assoc :created-objects new-created-objects)
        (dissoc :current))))

(defn update-import-location-status [{:keys [unpopulated-locations current]:as status}
                                   {:keys [location] :as import-result}]
  (let [create-spec (second current)
        new-created-location (merge create-spec location)
        new-unpopulated-locations (conj unpopulated-locations new-created-location)]
    (-> status
        (assoc :unpopulated-locations new-unpopulated-locations)
        (dissoc :current))))

(defn update-populate-location-status [{:keys [populated-locations current]:as status}
                                       {:keys [populate] :as populate-result}]
  (let [populate-spec (second current)
        populated-location (populate populate-spec)]
    (if-not (= status :done)
      (-> status
          (assoc :current [::populate-location populated-location]))
      (let [new-populated-locations (conj populated-locations populated-location)]
        (-> status
            (assoc :populated-locations new-populated-locations)
            (dissoc :current))))))

(defn update-status [{:keys [created-objects unpopulated-locations current]
                      :as status}
                     {:keys [object location] :as import-result}]
  (let [step (first current)]
    (cond
      (= step ::import-object) (update-import-object-status status import-result)
      (= step ::import-location) (update-import-location-status status import-result)
      (= step ::populate-location) (update-populate-location-status status import-result)
      :default status)))

(re-frame/reg-event-fx
 ::import-result
 (fn [{:keys [db] :as cofx} [_ result]]
   (let [{::keys [host session import-status]} db
         new-status (update-status import-status result)]
     (.log js/console (str "new status: " new-status))
     {:db (assoc db ::import-status new-status)
      :dispatch [::import {:host host,
                           :session session,
                           :status new-status}]})))
