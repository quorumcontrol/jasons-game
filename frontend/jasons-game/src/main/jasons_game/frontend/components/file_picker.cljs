(ns jasons-game.frontend.components.file-picker
  (:require [re-frame.core :as re-frame]
            [clojure.walk :refer [keywordize-keys]]
            ["yaml" :as yaml]))

(def element-id "open-file")

(defn enable-yaml-parsing [reader]
  (set! (.-onload reader) (fn [evt]
                            (let [contents (-> evt .-target .-result)
                                  parsed (->> contents
                                              (.parse yaml)
                                              js->clj
                                              keywordize-keys)]
                              (re-frame/dispatch [::load parsed])))))

(defn yaml-file-reader []
  (doto (js/FileReader.) enable-yaml-parsing))

(defn read-yaml-file [e]
  (let [file (-> e .-target .-files (aget 0))
        reader (yaml-file-reader)]
    (.readAsText reader file)))

(defn activate []
  (let [el (.getElementById js/document element-id)]
    (.click el)))

(defn element []
  [:input {:id element-id
           :type "file"
           :name element-id
           :style {:display "none"}
           :on-change read-yaml-file}])
