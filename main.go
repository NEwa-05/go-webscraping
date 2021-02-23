package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-github/github"
	gogit "github.com/google/go-github/github"
	"github.com/gorilla/mux"
	githuboauth "golang.org/x/oauth2/github"

	"golang.org/x/oauth2"
)

var (
	oauthConf = &oauth2.Config{
		ClientID:     "",
		ClientSecret: "",
		Scopes:       []string{},
		Endpoint:     oauth2.Endpoint{},
		RedirectURL:  "",
	}
)

func main() {

	getEnvInfo()
	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleMain)
	muxRouter.HandleFunc("/auth/github/login", loginCallback)
	muxRouter.HandleFunc("/auth/github/callback", authCallback)
	muxRouter.HandleFunc("/info", getUserInfo)
	fmt.Println("Listening on localhost:8080")
	err := http.ListenAndServe(":8080", muxRouter)
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on 8080")
}

func generateItem(url string) rssItem {
	//func scrape(url string) (title, description, link string) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	newRSS := rssItem{}
	dataInBytes, err := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)

	titleStartIndex := strings.Index(pageContent, "<title>")
	if titleStartIndex == -1 {
		fmt.Println("No title element found")
		os.Exit(0)
	}
	// The start index of the title is the index of the first
	// character, the < symbol. We don't want to include
	// <title> as part of the final value, so let's offset
	// the index by the number of characers in <title>
	titleStartIndex += 7

	// Find the index of the closing tag
	titleEndIndex := strings.Index(pageContent, "</title>")
	if titleEndIndex == -1 {
		fmt.Println("No closing tag for title found.")
		os.Exit(0)
	}

	pageTitle := []byte(pageContent[titleStartIndex:titleEndIndex])

	// Find a substr
	descriptionStartIndex := strings.Index(pageContent, "window__info-description")
	if descriptionStartIndex == -1 {
		fmt.Println("No description element found")
		os.Exit(0)
	}

	newRSS.Title = string(pageTitle)

	// The start index of the title is the index of the first
	// character, the < symbol. We don't want to include
	// <title> as part of the final value, so let's offset
	// the index by the number of characers in <title>
	descriptionStartIndex += 38

	// Find the index of the closing tag
	descriptionEndIndex := strings.Index(pageContent, "(https://www.patreon.com/electromonkeys)</p>")
	if descriptionEndIndex == -1 {
		fmt.Println("No closing tag for description found.")
		os.Exit(0)
	}
	descriptionEndIndex -= 77

	// (Optional)
	// Copy the substring in to a separate variable so the
	// variables with the full document data can be garbage collected
	pageDescription := []byte(pageContent[descriptionStartIndex:descriptionEndIndex])
	newRSS.Description = string(pageDescription)

	pageLink := url + dnsify(string(pageTitle))
	newRSS.Link = string(pageLink)

	newRSS.Image = "https://storage.buzzsprout.com/variants/k1gf0b0yd0yqkiz0qt1t2zj4smv1/74cb75bab2243992e98fab5156007185827084cf97936f24c0c66a651388df90.jpg"

	return newRSS
}

func generateRSS(item rssItem) {

	episodes := []rssItem{}
	episodes = append(episodes, item)

	rssStruct := &rss{
		Version:       "2.0",
		Title:         "Eletro Monkeys Podcast",
		Link:          "https://electro-monkeys.fr/",
		Description:   "Le podcast pour découvrir et comprendre les technologies et les concepts cloud natifs",
		LastBuildDate: time.Now().Format(time.RFC1123Z),
		Item:          episodes,
	}

	data, err := xml.MarshalIndent(rssStruct, "", "    ")
	if err != nil {
		fmt.Println(err)
	}

	rssFeed := []byte(xml.Header + string(data))
	if err := ioutil.WriteFile(filepath.Join("./", "rss.xml"), rssFeed, 0644); err != nil {
		fmt.Println(err)
	}
}

func getCurrentRSS() rss {
	rssFeed := rss{}
	data, err := ioutil.ReadFile(filepath.Join("./", "oldrss.xml"))
	if err != nil {
		fmt.Println(err)
	}
	if err = xml.Unmarshal(data, &rssFeed); err != nil {
		fmt.Println(err)
	}
	return rssFeed
}

func appendNewRSS(item rssItem, rssFeed rss) {
	rssFeed.Item = append(rssFeed.Item, item)
	rssFeed.LastBuildDate = time.Now().Format(time.RFC1123Z)

	data, err := xml.MarshalIndent(rssFeed, "", "    ")
	if err != nil {
		fmt.Println(err)
	}

	newRSSFeed := []byte(xml.Header + string(data))
	if err := ioutil.WriteFile(filepath.Join("./", "rss.xml"), newRSSFeed, 0644); err != nil {
		fmt.Println(err)
	}
}

type rss struct {
	Version       string `xml:"version,attr"`
	Title         string `xml:"channel>title"`
	Link          string `xml:"channel>link"`
	Description   string `xml:"channel>description"`
	LastBuildDate string `xml:"channel>lastBuildDate"`

	Item []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Image       string `xml:"image"`
}

func dnsify(title string) (mp3link string) {

	// Il manque des exception, genre "ê"
	replaceSpace := strings.ReplaceAll(title, " ", "-")
	toLower := strings.ToLower(replaceSpace)
	replaceEAcute := strings.ReplaceAll(toLower, "é", "e")
	replaceEGrave := strings.ReplaceAll(replaceEAcute, "è", "e")
	replaceAAcute := strings.ReplaceAll(replaceEGrave, "à", "a")
	replaceApostrophe := strings.ReplaceAll(replaceAAcute, "'", "-")
	mp3Link := "-" + replaceApostrophe + ".mp3"

	return mp3Link
}

func getUserInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET": //check if you only connect on the page
		tpl, err := template.ParseFiles("html_template/userinfo.html") //get template file
		if err != nil {
			log.Print("template parsing error: ", err)
		}
		err = tpl.Execute(w, nil) //present template file with variables
		if err != nil {
			log.Print("template executing error: ", err)
		}
	case "POST":
		r.ParseForm() //get form variables

		tpl, err := template.ParseFiles("html_template/startconv.html") //get template file
		if err != nil {
			log.Print("template parsing error: ", err)
		}

		gistID := r.Form.Get("gistID")
		gistFileName := r.Form.Get("gistFileName")

		//getGistFile(gistID, gistFileName)
		updateGistFile(gistID, gistFileName)

		err = tpl.Execute(w, nil) //present template file with variables
		if err != nil {
			log.Print("template executing error: ", err)
		}

	default:
		fmt.Printf("Unknown HTTP method : %s", r.Method)
	}
}

func getGistFile(gistID, gistFileName string) {
	client := github.NewClient(nil)
	gistres, _, err := client.Gists.Get(context.Background(), gistID)
	if err != nil {
		log.Printf("can't get gist: %s", err)
	}

	getFileMap := gistres.Files[github.GistFilename(gistFileName)]
	var gistFile GistFile
	gistByte, err := json.Marshal(getFileMap)
	if err != nil {
		fmt.Print("cannot marshal new slot")
	}
	err = json.Unmarshal(gistByte, &gistFile)
	if err != nil {
		fmt.Print("cannot unmarshal new value")
	}

	downfile, err := os.Create(*gistFile.Filename)
	if err != nil {
		fmt.Printf("cannot create file: %s", err)
	}

	defer downfile.Close()

	httpGet, err := http.Get(*gistFile.RawURL)
	if err != nil {
		fmt.Printf("can't do http get: %s", err)
	}

	defer httpGet.Body.Close()

	_, err = io.Copy(downfile, httpGet.Body)
	if err != nil {
		fmt.Printf("cannot write to file: %s", err)
	}

}

func updateGistFile(gistID, gistFileName string) {
	fileByteContent, err := ioutil.ReadFile(gistFileName)
	if err != nil {
		fmt.Printf("cannot read file: %s", err)
	}
	fileStringContent := string(fileByteContent)

	input := &gogit.Gist{
		Files: map[gogit.GistFilename]gogit.GistFile{
			gogit.GistFilename(gistFileName): {Content: &fileStringContent},
		},
	}

	cookieToken, err := ioutil.ReadFile("./token")
	if err != nil {
		fmt.Printf("cannot read token from file: %s", err)
	}
	var token *oauth2.Token
	err = json.Unmarshal(cookieToken, &token)
	if err != nil {
		fmt.Printf("cannot unmarshal token: %s", err)
	}
	ts := oauth2.StaticTokenSource(token)
	oauthClient := oauth2.NewClient(oauth2.NoContext, ts)
	client := gogit.NewClient(oauthClient)
	_, resp, err := client.Gists.Edit(context.Background(), gistID, input)
	if err != nil {
		log.Printf("can't get gist: %s", err)
	}

	fmt.Printf("test: %v", resp)
}

type clientInfo struct {
	ClientID     string `json:"ClientID"`
	ClientSecret string `json:"ClientSecret"`
}

func getEnvInfo() {

	var clientEnv clientInfo
	readFile, err := ioutil.ReadFile("./.env")
	if err != nil {
		fmt.Printf("Can't read file %s", err)
	}

	err = json.Unmarshal(readFile, &clientEnv)
	if err != nil {
		fmt.Printf("can't unmarshall json: %s", err)
	}

	oauthConf = &oauth2.Config{
		ClientID:     clientEnv.ClientID,
		ClientSecret: clientEnv.ClientSecret,
		Scopes:       []string{"user:email", "gist"},
		RedirectURL:  "http://localhost:8080/auth/github/callback",
		Endpoint:     githuboauth.Endpoint,
	}

}

func handleMain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	tpl, err := template.ParseFiles("html_template/home.html") //get template file
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = tpl.Execute(w, nil) //present template file with variables
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

// login
func loginCallback(w http.ResponseWriter, r *http.Request) {
	oauthState := generateStateOauthCookie(w)
	url := oauthConf.AuthCodeURL(oauthState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(20 * time.Minute)
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)
	return state
}

// /authCallback Called by github after authorization is granted
func authCallback(w http.ResponseWriter, r *http.Request) {

	oauthState, _ := r.Cookie("oauthstate")

	if r.FormValue("state") != oauthState.Value {
		log.Println("invalid oauth github state")
		http.Redirect(w, r, "/tutu", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)

	if err != nil {
		log.Println("cannot craete token")
		return
	}
	bytesToken, err := json.Marshal(token)
	if err != nil {
		fmt.Printf("cannot marshal token: %s", err)
	}
	ioutil.WriteFile("./token", bytesToken, 0755)

	http.Redirect(w, r, "/info", http.StatusTemporaryRedirect)

}
