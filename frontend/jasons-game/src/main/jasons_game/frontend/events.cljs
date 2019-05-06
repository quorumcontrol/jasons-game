(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))


(defn game-messages-query
  [db v]         ;; db is current app state, v the query vector
  (:game/messages db))

(re-frame.core/reg-sub  ;; part of the re-frame API
 :game-messages         ;; query id  
 game-messages-query)            ;; query fn

(re-frame/reg-event-db
 :initialize-db
 (fn-traced  [_ _]
             {:game/messages []
              :nav/page :home}))

(defn handle-user-input [{:keys [db]} [_ item]]
  {:db (update db :game/messages #(conj % item))})

(re-frame.core/reg-event-fx   ;; a part of the re-frame API
 :user-input                ;; the kind of event
 handle-user-input)

; (re-frame/reg-event-db
;  :routes/home
;  (fn-traced  [db _]
;              (-> db
;                  (assoc :page :home))))
