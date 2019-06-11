(ns jasons-game.frontend.components.terminal
  (:require [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm :refer [EmulatorState Outputs OutputFactory]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn new-state []
  (.createEmpty EmulatorState))

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
