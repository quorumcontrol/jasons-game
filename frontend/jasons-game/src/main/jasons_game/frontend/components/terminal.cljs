(ns jasons-game.frontend.components.terminal
  (:require [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal.commands :as commands]
            [clojure.string :as string]
            [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm :refer [EmulatorState Outputs OutputFactory]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn new-state []
  (.create EmulatorState (clj->js {:commandMapping commands/default-mapping})))

(defn text->output [txt]
  (.makeTextOutput OutputFactory txt))

(defn text->error-output [txt]
  (.makeErrorOutput OutputFactory
                    (clj->js {:source "game"
                              :type txt})))

(defn msg->output* [msg]
  (-> msg
      :message
      (str "\n\n")))

(defn msg->output [msg]
  (-> msg
      msg->output*
      text->output))

(defn msg->error-output [msg]
  (-> msg
      msg->output*
      text->error-output))

(defn add-output-record [outputs new-output]
  (.addRecord Outputs outputs new-output))

(defn add-output [state output]
  (-> state
      .getOutputs
      (add-output-record output)
      (as-> new-outputs (.setOutputs state new-outputs))))

(defn add-text-message [state msg]
  (let [msg-output (msg->output msg)]
    (add-output state msg-output)))

(defn add-error-message [state msg]
  (let [err-output (msg->error-output msg)]
    (add-output state err-output)))

(defn update-commands [state commands]
  (let [new-mapping (->> commands
                         (map #(string/split % #" "))
                         (map first)
                         (commands/add-all (commands/empty-mapping)))]
    (.setCommandMapping state new-mapping)))

(defn show []
  (let [current-input (r/atom "")]
    (fn [state read-only?]
      [:> ReactTerminalStateless {:emulatorState state
                                  :acceptInput (not read-only?)
                                  :inputStr @current-input
                                  :theme  {:background "#141313"
                                           :promptSymbolColor "#6effe6"
                                           :commandColor "#fcfcfc"
                                           :outputColor "#fcfcfc"
                                           :errorOutputColor "#ff89bd"
                                           :fontSize "1.1rem"
                                           :spacing "1%"
                                           :fontFamily "monospace"
                                           :height "100vh"
                                           :width "100vw"}
                                  :promptSymbol "$ >"
                                  :clickToFocus true
                                  :onInputChange (fn [new-input]
                                                   (reset! current-input new-input))
                                  :onStateChange (fn [new-state]
                                                   (re-frame/dispatch [::disable-input])
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
