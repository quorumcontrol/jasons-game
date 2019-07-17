(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.file-picker :as file-picker]
            [jasons-game.frontend.components.terminal :as terminal]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))

(re-frame/reg-event-fx
 :initialize
 (fn [{{::remote/keys [host session]} :db} _]
   {::remote/listen {:host host, :session session}}))

(re-frame/reg-event-fx
 :user/input
 (fn-traced [{{::remote/keys [host session]} :db} [_ user-command]]
   {::remote/send-input {:host host, :session session, :command user-command}
    :dispatch [::terminal/disable-input]}))

(re-frame/reg-event-fx
 :user/message
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db ::terminal/state terminal/add-text-message message-to-user)
    :dispatch [::terminal/enable-input]}))

(re-frame/reg-event-db
 :command/update
 (fn [db [_ command-update]]
   (update db ::terminal/state terminal/update-commands command-update)))

(re-frame/reg-event-fx
 ::file-picker/load
 (fn [{{::remote/keys [host session] :as db} :db} [_ {:keys [objects locations]}]]
   (let [status {:pending-objects objects
                 :created-objects []
                 :pending-locations locations
                 :unpopulated-locations []
                 :populated-locations []
                 :linked-locations []
                 :current nil}]
     {:db (assoc db
                 ::remote/importing true
                 ::remote/import-status status)
      :dispatch [::remote/import {:host host,
                                  :session session,
                                  :status status}]})))
