(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [jasons-game.frontend.db :as db]
            [jasons-game.frontend.remote.game :as game]
            [day8.re-frame.tracing :refer-macros [fn-traced]]
            [clojure.walk :refer [keywordize-keys]]))

(goog-define dev-host false)

(defonce default-host
  (if dev-host
    dev-host
    (-> js/window (.-location) (.-origin))))

(re-frame/reg-sub
 :game-messages
 (fn game-messages-sub [db _]
  (:game/messages db)))

(re-frame/reg-sub
 :remote/host
 (fn set-remote-host [db _]
   (:remote/host db)))

(re-frame/reg-event-fx
 :initialize
 (fn initialize [_ _]
   (re-frame/dispatch [:initialize-game-listener])
   {}))

(defn handle-game-message [resp]
  (if (not (.getHeartbeat resp))
    (do
      (.log js/console "game message" (.toObject resp))
      (let [clj-msg (keywordize-keys (js->clj (.toObject resp)))]
        (re-frame/dispatch [:game-message (conj {:user false} clj-msg)])))))

(defn handle-game-end [resp]
  (.log js/console "game end, redoing" resp)
  (re-frame/dispatch [:initialize-game-listener]))

(re-frame/reg-event-fx
 :initialize-game-listener
 (fn initialize-game-listener [{:keys [db]} _]
   (let [req (game/start-game-listener (:remote/host db) (:game/session db) handle-game-message handle-game-end)]
     {:db (conj db {:remote/current-listener req})})))

(re-frame/reg-event-db
 :initialize-db
 (fn-traced [_ _] db/initial-state))

(re-frame/reg-event-db
 :new-host
 (fn-traced  [db host]
   (conj db {:remote/host host})))


(defn handle-user-input [{:keys [db]} [_ user-command]]
  (game/send-user-input (:remote/host db) (:game/session db) user-command (fn [resp] (.log js/console resp)))
  {:db (update db :game/messages #(conj % {:user true :message user-command}))})

(re-frame/reg-event-fx
 :user-input
 handle-user-input)

(re-frame/reg-event-fx
 :game-message
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db :game/messages #(conj % message-to-user))}))
