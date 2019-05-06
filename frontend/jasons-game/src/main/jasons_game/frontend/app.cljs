(ns jasons-game.frontend.app
 (:require [reagent.core :as reagent]
           [re-frame.core :as re-frame]
           [jasons-game.frontend.views :as views]
           [jasons-game.frontend.events :as events]
           [jasons-game.frontend.service :as service]))

(defn mount-root []
  (re-frame/clear-subscription-cache!)
  (reagent/render [views/app-root]
                  (.getElementById js/document "app")))

(defn ^:export init []
  (re-frame/dispatch-sync [:initialize-db])
  (mount-root))