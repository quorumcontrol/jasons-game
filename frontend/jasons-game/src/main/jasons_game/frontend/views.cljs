(ns jasons-game.frontend.views
  (:require
   [jasons-game.frontend.components.terminal :as terminal]
   ["react"]
   ["react-dom" :as ReactDOM]
   ["semantic-ui-react" :refer (Container)]))

(defn app-root []
  [:> Container {:text true}
   [:p "hi!"]])
