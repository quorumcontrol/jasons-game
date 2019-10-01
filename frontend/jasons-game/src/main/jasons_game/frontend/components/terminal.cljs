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

(defn update-commands [state commands]
  (let [new-mapping (->> commands
                         (map #(string/split % #" "))
                         (map first)
                         (commands/add-all (commands/empty-mapping)))]
    (.setCommandMapping state new-mapping)))

(def password-command "password: ")

(defn handle-on-input-change [new-input current-input submission-val]
  (if (string/starts-with? new-input password-command)
    (let [input-length-diff (- (count new-input) (count submission-val))
      password-val (cond
        (= input-length-diff 1) (str submission-val (subs new-input (count submission-val)))
        (= input-length-diff -1) (subs submission-val 0 (count new-input))
        :default password-command)]
      (re-frame/dispatch [::update-submission-val password-val])
      (reset! current-input (str password-command (apply str (repeat (- (count password-val) (count password-command)) "*")))))
    (do (re-frame/dispatch [::update-submission-val new-input])
        (reset! current-input new-input))))

(defn show []
  (let [current-input (r/atom "")]
    (fn [state submission-val read-only?]
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
                                                   (handle-on-input-change new-input current-input submission-val))
                                  :onStateChange (fn [new-state]
                                                   (.log js/console "submitting " state)
                                                   (re-frame/dispatch [::disable-input])
                                                   (reset! current-input "")
                                                   (re-frame/dispatch [::update-submission-val ""])
                                                   (re-frame/dispatch [::change-state new-state]))}])))

(re-frame/reg-sub
 ::state
 (fn [{::keys [state] :as db} _]
   state))

(re-frame/reg-sub
 ::submission-val
 (fn [{::keys [submission-val] :as db} _]
   submission-val))

(re-frame/reg-sub
 ::read-only?
 (fn [{::keys [read-only?] :as db} _]
   read-only?))

(re-frame/reg-event-db
 ::change-state
 (fn [db [_ new-state]]
   (assoc db ::state new-state)))

(re-frame/reg-event-db
 ::update-submission-val
 (fn [db [_ new-val]]
   (assoc db ::submission-val new-val)))

(re-frame/reg-event-db
 ::disable-input
 (fn [db _]
   (assoc db ::read-only? true)))

(re-frame/reg-event-db
 ::enable-input
 (fn [db _]
   (assoc db ::read-only? false)))
