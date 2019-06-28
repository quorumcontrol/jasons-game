(ns jasons-game.frontend.components.terminal.commands
  (:require [jasons-game.frontend.remote :as remote]
            [clojure.string :as string]
            [clojure.walk :refer [keywordize-keys]]
            [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm
             :refer [CommandMapping OptionParser]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn parse-args [arg-str option-schema]
  (let [js-schema (clj->js option-schema)]
    (-> OptionParser
        (.parseOptions arg-str js-schema)
        js->clj
        keywordize-keys)))

(defmulti dispatch-command
  (fn [command-name args] command-name))

(defmethod dispatch-command :default [command-name arg-string]
  (let [command (->> [command-name arg-string]
                     (string/join " ")
                     string/trim)]
    (re-frame/dispatch [:user/input command])
    {}))

(defn add-command [commands new-command-name]
  (let [command-fn (fn [_ arg-string]
                     (dispatch-command new-command-name arg-string))]
    (.setCommand CommandMapping
                 commands new-command-name command-fn {})))

(def default-mapping
  (-> (.create CommandMapping)
      (add-command "call")
      (add-command "create")
      (add-command "connect")
      (add-command "set")
      (add-command "go")
      (add-command "build")
      (add-command "exit")
      (add-command "say")
      (add-command "shout")
      (add-command "drop")
      (add-command "pick")
      (add-command "look")
      (add-command "help")
      (add-command "open")
      (add-command "refresh")))
