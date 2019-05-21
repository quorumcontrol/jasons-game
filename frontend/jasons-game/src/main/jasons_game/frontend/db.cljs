(ns jasons-game.frontend.db
  (:require [jasons-game.frontend.remote :as remote]))

(def initial-state {:game/messages []
                    :game/session (remote/new-session "12345")
                    :remote/host remote/default-host
                    :nav/page :home})
