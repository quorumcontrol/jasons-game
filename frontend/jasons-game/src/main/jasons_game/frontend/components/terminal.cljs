(ns jasons-game.frontend.components.terminal
  (:require
   [reagent.core :as reagent]
   ["react-terminal-component" :as react-terminal-component :refer (ReactTerminal)]))

(defn terminal []
  [:> ReactTerminal])
