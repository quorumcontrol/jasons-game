(ns jasons-game.frontend.service
  (:require ["/frontend/books_pb" :as book-lib :refer (Book)]))

(let [my-book (new Book (cljs->js {:isbn 1234}))]
  (.log js/console "book: " (.getIsbn my-book)))