(ns jasons-game.frontend.db
  (:require [jasons-game.frontend.remote :as remote]
            [jasons-game.frontend.components.terminal :as terminal]
            [re-frame.core :as re-frame]
            [day8.re-frame.tracing :refer-macros [fn-traced]]))

(def initial-state {::remote/session (remote/new-session "12345")
                    ::remote/host remote/default-host
                    ::terminal/state (terminal/new-state)
                    ::terminal/read-only? false})

(re-frame/reg-event-db
 ::initialize
 (fn-traced [_ _] initial-state))
