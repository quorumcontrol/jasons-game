(ns jasons-game.frontend.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame :refer [subscribe dispatch]]
   [jasons-game.frontend.components.terminal :as terminal]
   [jasons-game.frontend.remote :as remote]
   ["react"]
   ["react-dom" :as ReactDOM]
   ["semantic-ui-react" :refer [Container Input Button Menu Form]]))

(defn app-root []
  (let [state (subscribe [::terminal/state])
        read-only? (subscribe [::terminal/read-only?])
        submission-val (subscribe [::terminal/submission-val])]
    [terminal/show @state @submission-val @read-only?]))
