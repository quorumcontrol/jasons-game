(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]
            [day8.re-frame.tracing :refer-macros [fn-traced]]
            [clojure.walk :refer [keywordize-keys]]))

(re-frame/reg-sub
 ::remote/messages
 (fn [{::remote/keys [messages] :as db} _]
   messages))

(re-frame/reg-sub
 ::remote/host
 (fn [{::remote/keys [host] :as db} _]
   host))

(re-frame/reg-sub
 ::terminal/state
 (fn [{::terminal/keys [state] :as db} _]
   state))

(re-frame/reg-sub
 ::terminal/read-only?
 (fn [{::terminal/keys [read-only?] :as db} _]
   read-only?))

(re-frame/reg-event-fx
 :initialize
 (fn [_ _]
   (re-frame/dispatch [::remote/listen])
   {}))

(defn resp->message [resp]
  (-> resp
      .toObject
      js->clj
      keywordize-keys))

(defn handle-game-message [resp]
  (if (not (.getHeartbeat resp))
    (do
      (.log js/console "game message" (.toObject resp))
      (let [game-msg (-> resp
                         resp->message
                         (assoc :user false))]
        (re-frame/dispatch [:game-message game-msg])))))

(defn handle-game-end [resp]
  (.log js/console "game end, redoing" resp)
  (re-frame/dispatch [::remote/listen]))

(re-frame/reg-event-fx
 ::remote/listen
 (fn [{:keys [db]} _]
   (let [req (remote/start-game-listener (::remote/host db) (::remote/session db) handle-game-message handle-game-end)]
     {:db (assoc db :remote/current-listener req)})))

(re-frame/reg-event-db
 ::db/initialize
 (fn-traced [_ _] db/initial-state))

(re-frame/reg-event-db
 :new-host
 (fn-traced [db host]
   (assoc db ::remote/host host)))

(re-frame/reg-fx
 ::remote/input
 (fn [{:keys [host session command]}]
   (remote/send-user-input host session command
                           (fn [resp] (.log js/console resp)))))

(re-frame/reg-event-fx
 ::remote/send-input
 (fn-traced [{{::remote/keys [host session]} :db} [_ user-command]]
   {::remote/input {:host host, :session session, :command user-command}
    :dispatch [::terminal/disable-input]}))

(re-frame/reg-event-fx
 :game-message
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db ::terminal/state terminal/add-text-message message-to-user)
    :dispatch [::terminal/enable-input]}))

(re-frame/reg-event-db
 ::terminal/change-state
 (fn [db [_ new-state]]
   (assoc db ::terminal/state new-state)))

(re-frame/reg-event-db
 ::terminal/disable-input
 (fn [db _]
   (assoc db ::terminal/read-only? true)))

(re-frame/reg-event-db
 ::terminal/enable-input
 (fn [db _]
   (assoc db ::terminal/read-only? false)))
