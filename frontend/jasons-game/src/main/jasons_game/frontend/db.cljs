(ns jasons-game.frontend.db
  (:require [jasons-game.frontend.remote.game :as game]))

(def initial-state {:game/messages []
                    :game/session (game/new-session "12345")
                    :remote/host default-host
                    :nav/page :home})
