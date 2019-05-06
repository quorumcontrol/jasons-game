(ns jasons-game.frontend.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame :refer [subscribe dispatch]]
   [jasons-game.frontend.components.terminal :as terminal]
   ["react"]
   ["react-dom" :as ReactDOM]
   ["semantic-ui-react" :refer [Container Input Button]]))

(defn user-message [msg]
  [:p msg])

(defn app-root []
  [:> Container {:text true}
   [:> Container {:text true}
    (let [messages (subscribe [:game-messages])]
      (map user-message @messages))]
    ;; TODO: put the text here]
   (let [input-state (r/atom "")]
     [:div
       [:> Input {:onChange (fn [evt] (reset! input-state (-> evt .-target .-value))) :placeholder "What do you want to do?"}]
       [:> Button {:onClick #(dispatch [:user-input @input-state])} "Send"]])])
