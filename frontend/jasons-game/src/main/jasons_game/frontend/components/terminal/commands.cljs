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

(defmethod dispatch-command :default [command-name arg-list]
  (let [command (->> arg-list
                     js->clj
                     (concat [command-name])
                     (string/join " ")
                     string/trim)]
    (re-frame/dispatch [:user/input command])
    {}))

(defn add-command [commands new-command-name]
  (let [command-fn (fn [_ arg-list]
                     (dispatch-command new-command-name arg-list))]
    (.setCommand CommandMapping
                 commands new-command-name command-fn {})))

(defn empty-mapping []
  (.create CommandMapping))

(defn add-all [mapping commands]
  (if (empty? commands)
    mapping
    (recur (add-command mapping (first commands))
           (rest commands))))

(def default-mapping
  (empty-mapping))
