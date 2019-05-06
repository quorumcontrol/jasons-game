(ns jasons-game.frontend.remote.game
  (:require
   ["@improbable-eng/grpc-web" :as grpc-lib :refer (grpc)]
   ["/frontend/remote/jasonsgame_pb" :as game-lib]
   ["/frontend/remote/jasonsgame_pb_service" :as game-service]))
