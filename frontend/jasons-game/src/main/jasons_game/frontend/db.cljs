(ns jasons-game.frontend.db
  (:require [jasons-game.frontend.remote :as remote]))

(def initial-state {::remote/messages []
                    ::remote/session (remote/new-session "12345")
                    ::remote/host remote/default-host
                    :nav/page :home})
