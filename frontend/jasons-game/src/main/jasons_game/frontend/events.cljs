(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]
            [day8.re-frame.tracing :refer-macros [fn-traced]]
            [clojure.walk :refer [keywordize-keys]]))

(re-frame/reg-sub
 ::remote/messages
 (fn [db _]
  (::remote/messages db)))

(re-frame/reg-sub
 ::remote/host
 (fn [db _]
   (::remote/host db)))

(re-frame/reg-sub
 ::terminal/state
 (fn [db _]
   (::terminal/state db)))

(re-frame/reg-event-fx
 :initialize
 (fn [_ _]
   (re-frame/dispatch [::remote/listen])
   {}))

(defn handle-game-message [resp]
  (if (not (.getHeartbeat resp))
    (do
      (.log js/console "game message" (.toObject resp))
      (let [clj-msg (keywordize-keys (js->clj (.toObject resp)))]
        (re-frame/dispatch [:game-message (conj {:user false} clj-msg)])))))

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

(defn handle-user-input [{:keys [db]} [_ user-command]]
  (remote/send-user-input (::remote/host db) (::remote/session db) user-command (fn [resp] (.log js/console resp)))
  {:db (update db ::remote/messages #(conj % {:user true :message user-command}))})

(re-frame/reg-event-fx
 :user-input
 handle-user-input)

(re-frame/reg-event-fx
 :game-message
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db ::remote/messages #(conj % message-to-user))}))

(re-frame/reg-event-db
 ::terminal/change-state
 (fn [db new-state]
   (assoc db ::terminal/state new-state)))
