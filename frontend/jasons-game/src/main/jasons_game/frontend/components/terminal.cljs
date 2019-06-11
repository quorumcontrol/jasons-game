(ns jasons-game.frontend.components.terminal
  (:require [jasons-game.frontend.remote :as remote]
            [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm :refer [CommandMapping defaultCommandMapping EmulatorState Outputs OutputFactory]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn build-command [f opts]
  (clj->js {:function f, :optDef opts}))

(defn help-command [_ _]
  (re-frame/dispatch [:user/input "help"])
  (clj->js {}))

(def commands
  (let [commands (.create CommandMapping)]
    (.setCommand CommandMapping commands "help" help-command {})))

(defn new-state []
  (.create EmulatorState (clj->js {:commandMapping commands})))

(defn text->output [txt]
  (.makeTextOutput OutputFactory txt))

(defn msg->output [msg]
  (-> msg
      :message
      (str "\n\n")
      text->output))

(defn add-output [outputs new-output]
  (.addRecord Outputs outputs new-output))

(defn add-text-message [state msg]
  (let [msg-output (msg->output msg)]
    (-> state
        .getOutputs
        (add-output msg-output)
        (as-> new-outputs (.setOutputs state new-outputs)))))

(defn show [state read-only?]
  (let [current-input (r/atom "")]
    (fn [state]
      [:> ReactTerminalStateless {:emulatorState state
                                  :acceptInput (not read-only?)
                                  :inputStr @current-input
                                  :onInputChange (fn [new-input]
                                                   (reset! current-input new-input))
                                  :onStateChange (fn [new-state]
                                                   (reset! current-input "")
                                                   (re-frame/dispatch [::change-state new-state]))}])))

(re-frame/reg-sub
 ::state
 (fn [{::keys [state] :as db} _]
   state))

(re-frame/reg-sub
 ::read-only?
 (fn [{::keys [read-only?] :as db} _]
   read-only?))

(re-frame/reg-event-db
 ::change-state
 (fn [db [_ new-state]]
   (assoc db ::state new-state)))

(re-frame/reg-event-db
 ::disable-input
 (fn [db _]
   (assoc db ::read-only? true)))

(re-frame/reg-event-db
 ::enable-input
 (fn [db _]
   (assoc db ::read-only? false)))
