(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))

(re-frame/reg-event-fx
 :user/input
 (fn-traced [{{::remote/keys [host session]} :db} [_ user-command]]
   {::remote/send-input {:host host, :session session, :command user-command}
    :dispatch [::terminal/disable-input]}))

(re-frame/reg-event-db
 :command/update
 (fn [db [_ command-update]]
   (update db ::terminal/state terminal/update-commands command-update)))
