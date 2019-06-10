(ns jasons-game.frontend.components.terminal
  (:require [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm :refer [EmulatorState Outputs OutputFactory]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn new-state []
  (.createEmpty EmulatorState))

(defn msg->output [msg]
  (.makeTextOutput OutputFactory (:message msg)))

(defn add-output [outputs new-output]
  (.addRecord Outputs outputs new-output))

(defn add-text-message [state msg]
  (let [msg-output (msg->output msg)]
    (-> state
        .getOutputs
        (add-output msg-output)
        (as-> new-outputs (.setOutputs state new-outputs)))))

(defn show [state]
  (let [current-input (r/atom "")]
    (fn [state]
      [:> ReactTerminalStateless {:emulatorState state
                                  :inputStr @current-input
                                  :onInputChange (fn [new-input]
                                                   (reset! current-input new-input))
                                  :onStateChange (fn [new-state]
                                                   (re-frame/dispatch [::change-state new-state]))}])))
