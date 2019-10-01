(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))

(re-frame/reg-event-fx
 :initialize
 (fn [{{::remote/keys [host session]} :db} _]
   {::remote/listen {:host host, :session session}}))

(re-frame/reg-event-fx
 :user/input
 (fn-traced [{{::remote/keys [host session] ::terminal/keys [submission-val]} :db} [_ _]]
   {::remote/send-input {:host host, :session session, :command submission-val}}))

(re-frame/reg-event-fx
 :user/message
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db ::terminal/state terminal/add-text-message message-to-user)
    :dispatch [::terminal/enable-input]}))

(re-frame/reg-event-fx
  :user/heartbeat
  (fn [_ _]
    {:dispatch [::terminal/enable-input]}))

(re-frame/reg-event-db
 :command/update
 (fn [db [_ command-update]]
   (update db ::terminal/state terminal/update-commands command-update)))
