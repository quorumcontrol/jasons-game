(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame :refer [dispatch dispatch-sync]]
            [jasons-game.frontend.remote.game :as game]
            [day8.re-frame.tracing :refer-macros [fn-traced]]
            [clojure.walk :refer [keywordize-keys]]))

(goog-define dev-host false)
  
(defonce default-host (if dev-host dev-host (-> js/window (.-location) (.-origin))))

(re-frame.core/reg-sub  
 :game-messages        
 (fn game-messages-sub [db _]
  (:game/messages db)))

(re-frame.core/reg-sub  
 :remote/host         
 (fn set-remote-host [db _]
   (:remote/host db)))

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :initialize                ;; the kind of event
 (fn initialize [_ _]
   (dispatch [:initialize-game-listener])
   {}))

(defn handle-game-message [resp]
  (if (not (.getHeartbeat resp))
    (do
      (.log js/console "game message" (.toObject resp))
      (let [clj-msg (keywordize-keys (js->clj (.toObject resp)))]
        (dispatch [:game-message (conj {:user false} clj-msg)])))))

(defn handle-game-end [resp]
  (.log js/console "game end, redoing" resp)
  (dispatch [:initialize-game-listener]))

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :initialize-game-listener                ;; the kind of event
 (fn initialize-game-listener [{:keys [db]} _]
   (let [req (game/start-game-listener (:remote/host db) (:game/session db) handle-game-message handle-game-end)]
     {:db (conj db {:remote/current-listener req})})))

(re-frame/reg-event-db
 :initialize-db
 (fn-traced  [_ _]
             {:game/messages []
              :game/session (game/new-session "12345")
              :remote/host default-host
              :nav/page :home}))

(re-frame/reg-event-db
 :new-host
 (fn-traced  [db host]
   (conj db {:remote/host host})))
 

(defn handle-user-input [{:keys [db]} [_ user-command]]
  (game/send-user-input (:remote/host db) (:game/session db) user-command (fn [resp] (.log js/console resp)))
  {:db (update db :game/messages #(conj % {:user true :message user-command}))})

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :user-input                ;; the kind of event
 handle-user-input)

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :game-message                ;; the kind of event
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db :game/messages #(conj % message-to-user))}))
