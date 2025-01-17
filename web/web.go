package web

import (
	"fmt"
	"log"
	"net/http"
)

func StartWebUI(port int, globalRecords map[string]string, zones map[string]map[string]string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h1>DNS Records</h1>")

		fmt.Fprint(w, "<h2>Global Records</h2><ul>")
		for name, ip := range globalRecords {
			fmt.Fprintf(w, "<li>%s -> %s</li>", name, ip)
		}
		fmt.Fprint(w, "</ul>")

		for zoneName, records := range zones {
			fmt.Fprintf(w, "<h2>Zone: %s</h2><ul>", zoneName)
			for name, ip := range records {
				fmt.Fprintf(w, "<li>%s -> %s</li>", name, ip)
			}
			fmt.Fprint(w, "</ul>")
		}
	})

	log.Printf("Starting Web UI on port %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
