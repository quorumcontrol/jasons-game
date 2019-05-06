(ns jasons-game.frontend.events
  (:require [re-frame.core :as re-frame]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))

(re-frame/reg-event-db
 :initialize-db
 (fn-traced  [_ _]
            ;  (service/get-book "http://localhost:8080" "1234" (fn [res] (.log js/console res)))
             {:page :home}))

; (re-frame/reg-event-db
;  :routes/home
;  (fn-traced  [db _]
;              (-> db
;                  (assoc :page :home))))
