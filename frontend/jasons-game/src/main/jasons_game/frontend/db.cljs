(ns jasons-game.frontend.db
  (:require [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]))

(def initial-state {::remote/messages []
                    ::remote/session (remote/new-session "12345")
                    ::remote/host remote/default-host
                    :nav/page :home
                    ::terminal/state (terminal/new-state)
                    ::terminal/read-only? false})
