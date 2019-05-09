(ns jasons-game.frontend.views
  (:require
   [reagent.core :as r]
   [re-frame.core :as re-frame :refer [subscribe dispatch]]
   [jasons-game.frontend.components.terminal :as terminal]
   ["react"]
   ["react-dom" :as ReactDOM]
   ["semantic-ui-react" :refer [Container Input Button Menu Form]]))

(defn user-message [idx msg]
  (let [prefix (if (:user msg) "$ " ">>> ")
        loc (:location msg)]
    [:div {:key idx}
     (if loc
       [:p
        (str "[" (:did loc) ", (" (:x loc) "," (:y loc) ")" " tip: " (:tip loc) "] ")])
     [:p (str prefix (:message msg))]]))

(defn scrolling-container [messages]
  (let [bottom-el (atom nil)]
    (r/create-class
     {:display-name "scrolling-container"

      :component-did-mount
      (fn [_]
        (.scrollIntoView @bottom-el (clj->js {:behavior "smooth"})))

      :component-did-update
      (fn [_]
        (.scrollIntoView @bottom-el (clj->js {:behavior "smooth"})))

      :reagent-render
      (fn [messages]
        [:> Container {:style {:overflow "auto" :maxHeight "50vh"}}
         (map-indexed user-message messages)
         [:div {:ref (fn [el] (reset! bottom-el el))}]])})))


(defn app-root []
  (let [input-state (r/atom "")]
    (fn []
      [:div
       [:> Menu {:fixed "bottom"}
        [:> Form {:onSubmit (fn [evt]
                              (.log js/console "submission" evt)
                              (dispatch [:user-input @input-state])
                              (reset! input-state ""))}
         [:> Input {:onChange (fn [evt] (reset! input-state (-> evt .-target .-value)))
                    :action {:labelPosition "right"
                             :content "Send"
                             :type "submit"}
                    :size "big"
                    :value @input-state
                    :placeholder "What do you want to do?"}]]]
       (let [messages (subscribe [:game-messages])]
         [scrolling-container @messages])])))

