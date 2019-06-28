(ns jasons-game.frontend.app
 (:require [reagent.core :as reagent]
           [re-frame.core :as re-frame]
           [jasons-game.frontend.db :as db]
           [jasons-game.frontend.views :as views]
           [jasons-game.frontend.events :as events]))

(defn mount-root []
  (re-frame/clear-subscription-cache!)
  (reagent/render [views/app-root]
                  (.getElementById js/document "app")))

(defn ^:export init []
  (re-frame/dispatch-sync [::db/initialize])
  (re-frame/dispatch [:initialize])
  (mount-root))
