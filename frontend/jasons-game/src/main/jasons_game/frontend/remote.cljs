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

(defn get-created-did [created created-name]
  (->> created
       (filter #(= (:name %) created-name))
       first
       :did))

(defn new-drop-proto [{:keys [created-objects current] :as status}]
  (let [{:keys [objects] :as current-loc} (second current)
        object-name (first objects)
        did (get-created-did created-objects object-name)]
    (doto (game-lib/PopulateDropObjectsPhase.)
      (.setObjectName object-name)
      (.setObjectDid did))))

(defn new-connect-proto [{:keys [populated-locations unpopulated-locations current] :as status}]
  (let [{:keys [links] :as pop-spec} (second current)
        location-name (-> links first :location)
        link-name (-> links first :name)
        did (get-created-did (concat unpopulated-locations populated-locations)
                              location-name)]
    (doto (game-lib/PopulateDropObjectsPhase.)
      (.setConnectionName link-name)
      (.setToDid did))))

(defn set-phase-proto [spec {:keys [current] :as status} phase]
  (let [pop-loc (second current)]
    (case phase
      :describe (.setDescribe spec (doto (game-lib/PopulateDescribePhase.)
                                     (.setDescription (:description pop-loc))))
      :drop (.setDrop spec (new-drop-proto status))
      :connect (.setConnect spec (new-connect-proto status))
      (.setVisit spec (game-lib/PopulateVisitPhase.))))
  spec)

(defn status->populate-proto [{:keys [current] :as status}]
  (let [{:keys [name did phase]} (second current)]
    (doto (game-lib/PopulateSpec.)
      (.setName name)
      (.setDid did)
      (set-phase-proto status phase))))

(defn new-populate-location-request [session status]
  (let [pop-loc-proto (status->populate-proto status)]
    (doto (new-import-request session)
      (.setPopulate pop-loc-proto))))

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
   (let [req (new-import-object-request session spec)]
     (send-import-request host req (fn [resp] (.log js/console resp))))))

(re-frame/reg-fx
 ::import-location
 (fn [{:keys [host session spec]}]
   (let [req (new-import-location-request session spec)]
     (send-import-request host req (fn [resp] (.log js/console resp))))))

(re-frame/reg-fx
 ::populate-location
 (fn [{:keys [host session status]}]
   (let [req (new-populate-location-request session status)]
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
  (if (:current import-status)
    {::populate-location {:host host, :session session, :status import-status}}
    (when-let [unpopulated-locations (seq (:unpopulated-locations import-status))]
      (let [new-status (-> import-status
                           (update :unpopulated-locations rest)
                           (assoc :current [::populate-location (first unpopulated-locations)]))]
        {:db (assoc db ::import-status new-status)
         ::populate-location {:host host, :session session, :status new-status}}))))

(re-frame/reg-event-fx
 ::import
 (fn [{db :db} _]
   (let [{:keys [pending-objects pending-locations unpopulated-locations]}
         (::import-status db)]
     (cond
       (seq pending-objects) (import-next-object db)
       (seq pending-locations) (import-next-location db)
       :else (populate-next-location db)))))

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

(defn increment-drop-phase [{:keys [objects] :as current-loc}]
  (if-let [other-objects (-> objects rest seq)]
    (-> current-loc
        (assoc :phase :drop)
        (assoc :objects other-objects))
    (-> current-loc
        (assoc :phase :connect)
        (dissoc :objects))))

(defn increment-connect-phase [{:keys [links] :as current-loc}]
  (if (seq links)
    (assoc current-loc :phase :connect)
    (assoc current-loc :phase :done)))

(defn increment-populate-phase [{:keys [phase] :as current-loc}]
  (case phase
    :visit (assoc current-loc :phase :describe)
    :describe (assoc current-loc :phase :drop)
    :drop (increment-drop-phase current-loc)
    :connect (increment-connect-phase current-loc)
    (assoc current-loc :phase :visit)))

(defn finalize-location [{:keys [unpopulated-locations current]
                          :as status}]
  (let [current-location (second current)]
    (if (seq unpopulated-locations)
      (-> status
          (update :populated-locations conj current-location)
          (update :unpopulated-locations rest)
          (assoc :current [::populate-location (first unpopulated-locations)]))
      (-> status
          (update :populated-locations conj current-location)
          (dissoc current)))))


(defn update-populate-location-status [{:keys [populated-locations current] :as status}
                                       {:keys [populate] :as populate-result}]
  (let [current-loc (second current)
        new-loc (increment-populate-phase current-loc)]
    (assoc status :current [::populate-location new-loc])
    (if (= (:phase new-loc) :done)
      (-> status
          (update :populated-locations conj new-loc)
          (dissoc :current))
      (-> (assoc status :current [::populate-location new-loc])))))

(defn update-status [{:keys [current] :as status} import-result]
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
