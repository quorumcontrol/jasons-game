(ns jasons-game.frontend.components.terminal.commands
  (:require [jasons-game.frontend.remote :as remote]
            [reagent.core :as r]
            [re-frame.core :as re-frame]
            ["javascript-terminal" :as jsterm :refer [CommandMapping]]
            ["react-terminal-component" :refer [ReactTerminalStateless]]))

(defn build-command [f opts]
  (clj->js {:function f, :optDef opts}))

(defn add-command
  ([commands new-command-name new-command-fn]
   (add-command commands new-command-name new-command-fn {}))
  ([commands new-command-name new-command-fn new-command-opts]
   (.setCommand CommandMapping commands
                new-command-name new-command-fn new-command-opts)))

(defn help-command [_ _]
  (re-frame/dispatch [:user/input "help"])
  (clj->js {}))

(defn refresh-command [_ _]
  (re-frame/dispatch []))

(def mapping
  (-> (.create CommandMapping)
      (add-command "help" help-command)))
