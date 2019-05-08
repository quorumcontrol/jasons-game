(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame :refer [dispatch dispatch-sync]]
            [jasons-game.frontend.remote.game :as game]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))


(def host "http://localhost:8080")

(defn game-messages-query
  [db v]         ;; db is current app state, v the query vector
  (:game/messages db))

(re-frame.core/reg-sub  ;; part of the re-frame API
 :game-messages         ;; query id  
 game-messages-query)            ;; query fn

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :initialize                ;; the kind of event
 (fn [coffects _]
   (dispatch [:initialize-game-listener])
   {}))

(defn handle-game-message [resp]
  (if (not (.getHeartbeat resp))
    (do
      (.log js/console "game message" resp)
      (dispatch [:game-message {:user false :message (.getMessage resp) :game resp}]))))

(defn handle-game-end [resp]
  (.log js/console "game end, redoing" resp)
  (dispatch [:initialize-game-listener]))

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :initialize-game-listener                ;; the kind of event
 (fn [{:keys [db]} _]
   (game/start-game-listener host (:game/session db) handle-game-message handle-game-end)
   {}))

(re-frame/reg-event-db
 :initialize-db
 (fn-traced  [_ _]
             {:game/messages []
              :game/session (game/new-session "12345")
              :nav/page :home}))

(defn handle-user-input [{:keys [db]} [_ user-command]]
  (game/send-user-input host (:game/session db) user-command (fn [resp] (.log js/console resp)))
  {:db (update db :game/messages #(conj % {:user true :message user-command}))})

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :user-input                ;; the kind of event
 handle-user-input)

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :game-message                ;; the kind of event
 (fn [{:keys [db]} [_ message-to-user]]
   {:db (update db :game/messages #(conj % message-to-user))}))
