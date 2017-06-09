package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/stianeikeland/go-rpio"
)

func IsPi() bool {
	return runtime.GOARCH == "arm"
}

var templates = template.Must(template.ParseGlob("templates/*.html"))

// PersonData is ...
type PersonData struct {
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	Name      string   `json:"name"`
	Slack     string   `json:"slack"` // Slack username
	Photo     string   `json:"photo"` // Photo filename
	Title     string   `json:"title"` // Job title
	Notes     string   `json:"notes"` // Extra notes (freeform HTML).
	Hours     string   `json:"hours"` // What time they are in the office (or "normal").
	Phone     string   `json:"phone"`
	Hostnames []string `json:"hostnames"` // Hostnames for their phone, laptop etc.
}

// PrinterData is ...
type PrinterData struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Name  string  `json:"name"`  // Printer friendly name
	URL   string  `json:"url"`   // Slack username
	Photo string  `json:"photo"` // Photo filename
	Notes string  `json:"notes"` // Extra notes (freeform HTML).
}

// RoomData is ...
type RoomData struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Name  string  `json:"name"`
	Notes string  `json:"notes"` // Extra notes (freeform HTML).
	Photo string  `json:"photo"`
}

// ThingData is for everything else.
type ThingData struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Icon  string  `json:"icon"`
	Label string  `json:"label"` // Freeform HTML
}

// AccessPoint is ...
type AccessPoint struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Name string  `json:"name"`
	Mac  string  `json:"mac"`
}

// IndexData is the data sent to the index template.
type IndexData struct {
	People      []PersonData  `json:"people"`
	Printers    []PrinterData `json:"printers"`
	Rooms       []RoomData    `json:"rooms"`
	Things      []ThingData   `json:"things"`
	AccessPoint []AccessPoint `json:"access_points"`
}

type WapData struct {
	Hostname string `json:"hostname"`
	ApMac    string `json:"ap_mac"`
	Signal   int    `json:"signal"`
}

type WapResponse struct {
	Data []WapData `json:"data"`
}

var indexData IndexData

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", indexData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var hornPin rpio.Pin

func confettiHandler(w http.ResponseWriter, r *http.Request) {
	/*	fmt.Fprintf(w, "%s: Horn BARRRAP", r.URL.Path)

		go func() {
			if !IsPi() {
				return
			}
			hornPin.High()
			time.Sleep(1000 * time.Millisecond)
			hornPin.Low()
		}()
	*/

	resp, err := http.Get("http://map.local:8080/api/confetti")

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	w.Write(body)

}

func presence1Handler(w http.ResponseWriter, r *http.Request) {

	resp, err := http.Get("http://map.local:8080/api/presence1")

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	w.Write(body)
}

func trackerApiQuery(client *http.Client, query string, data string) []byte {
	baseUrl := "https://10.0.1.10:8443/api/"
	body := strings.NewReader(data)
	req, err := http.NewRequest("POST", baseUrl+query, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes()
}

// trackerGetData returns a list of hostnames, and the access point.
func trackerGetData() []WapData {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Transport: tr,
		Jar:       cookieJar,
	}

	trackerApiQuery(client, "login", "{'username':'maphack', 'password':'showmetheway'}")
	wapBytes := trackerApiQuery(client, "s/default/stat/sta", "")
	trackerApiQuery(client, "logout", "")

	var response WapResponse
	err := json.Unmarshal(wapBytes, &response)
	if err != nil {
		log.Print(err)
	}

	return response.Data
}

func findPersonHandler(w http.ResponseWriter, r *http.Request) {
	// Get the name
	name := r.FormValue("name")

	// Find it in the list of people.
	idx := -1
	for i, n := range indexData.People {
		if n.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		// Person not found.
		http.Error(w, "Person not found", http.StatusBadRequest)
		return
	}

	// Get the tracking data.
	track := trackerGetData()

	// See if any of their hostnames are in it.
	for _, hostname := range indexData.People[idx].Hostnames {

		// See if we can find their hostname.
		for _, trackedHostname := range track {
			if trackedHostname.Hostname == hostname {
				// Yeay. See if we can find the access point for this hostname.

				for _, ap := range indexData.AccessPoint {
					if strings.ToLower(trackedHostname.ApMac) == strings.ToLower(ap.Mac) {
						// Yeay, location found!
						fmt.Fprintf(w, `{ "x": %f, "y": %f }`, ap.X, ap.Y)
						return
					}
				}
			}
		}
	}
	// Fail.
	fmt.Fprintf(w, `{}`)
	return
}

func hornHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "%s: Horn BARRRAP", r.URL.Path)

	cmd := exec.Command("afplay", "horn.mp3")
	cmd.Start()

}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	apiPath := strings.Replace(r.URL.Path, "/api/", "", 1)

	device := apiPath[0:strings.Index(apiPath, "/")]

	log.Println(device)

	devicePath := apiPath[len(device)+1:]

	log.Println(devicePath)
}

func main() {

	if IsPi() {
		log.Println("Starting GPIO...")
		rpio.Open()
		hornPin = rpio.Pin(4)
		hornPin.Mode(rpio.Output)
	}

	log.Println("Loading markers...")
	data, err := ioutil.ReadFile("markers.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(data, &indexData)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Downloading wap data...")
	hostnameToZoneMap := trackerGetData()
	for _, track := range hostnameToZoneMap {
		fmt.Printf("%s: %s\n", track.ApMac, track.Hostname)
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static", fs))
	http.HandleFunc("/api/confetti", confettiHandler)
	http.HandleFunc("/api/presence1", presence1Handler)
	http.HandleFunc("/api/horn", hornHandler)
	http.HandleFunc("/api/find_person", findPersonHandler)
	http.HandleFunc("/", indexHandler)
	var port string
	port = os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Println("Starting map server on port " + port + "...")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
