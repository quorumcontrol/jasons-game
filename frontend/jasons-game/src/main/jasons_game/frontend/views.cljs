(ns jasons-game.frontend.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame :refer [subscribe dispatch]]
   [jasons-game.frontend.components.terminal :as terminal]
   ["react"]
   ["react-dom" :as ReactDOM]
   ["semantic-ui-react" :refer [Container Input Button Menu Form]]))

(defn user-message [idx msg]
  [:p {:key idx} msg])

(defn app-root []
  [:div
   [:> Menu {:fixed "bottom"}
    (let [input-state (r/atom "")]
      [:> Form {:onSubmit (fn [evt]
                            (.log js/console "submission" evt)
                            (dispatch [:user-input @input-state])
                            (reset! input-state ""))}
       [:> Input {:onChange (fn [evt] (reset! input-state (-> evt .-target .-value)))
                  :action {:labelPosition "right"
                           :content "Send"
                           :type "submit"}
                  :actionPosition "right"
                  :size "big"
                  :placeholder "What do you want to do?"}]])]
   [:> Container
    (let [messages (subscribe [:game-messages])]
      (map-indexed user-message @messages))]])

