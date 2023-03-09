package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/datastore"
)

type WorkItem struct {
	Id           *datastore.Key `datastore:"__key__"`
	UserId       int
	WorkdateTime time.Time
	TimeType     string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.Handle("/views/", http.StripPrefix("/views/", http.FileServer(http.Dir("../views/"))))

	http.HandleFunc("/", Index)
	http.HandleFunc("/create", WorkItemCreate)
	http.HandleFunc("/edit", WorkItemEdit)
	http.HandleFunc("/update", WorkItemUpdate)
	http.ListenAndServe(":"+port, nil)
}

func dbConn() (*datastore.Client, error) {
	// Datastore用のコンテキストとクライアントを作成する
	ctx := context.Background()

	projectId := os.Getenv("DATASTORE_PROJECT_ID")

	client, err := datastore.NewClient(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("datastore.NewClient: %v", err)
	}
	return client, nil
}

func Index(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("../templates/index.html")
	if err != nil {
		http.Error(w, "Could not connect to index.html", http.StatusNotFound)
	}

	ctx := context.Background()
	client, err := dbConn()
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to connect to dbConn(): %v", err), http.StatusInternalServerError)
		return
	}

	query := datastore.NewQuery("WorkItem")

	var records []WorkItem

	_, err = client.GetAll(ctx, query, &records)
	if err != nil {
		http.Error(w, fmt.Sprintf("Data stored in datastore cannot be retrieved: %v", err), http.StatusInternalServerError)
		return
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")
	for i := range records {
		records[i].WorkdateTime = records[i].WorkdateTime.In(loc)
	}

	tmpl.Execute(w, records)
}

func WorkItemCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

		// フォームに入力された値を取得する
		userid, _ := strconv.Atoi(r.FormValue("userid"))
		timetype := r.FormValue("timetype")

		ctx := context.Background()
		client, err := dbConn()
		if err != nil {
			http.Error(w, fmt.Sprintf("Unable to connect to dbConn(): %v", err), http.StatusInternalServerError)
			return
		}

		// WorkItem構造体のキーを作成する
		key := datastore.IncompleteKey("WorkItem", nil)

		// フォームで入力された値で新しいWorkItem構造体を作成する
		workitem := &WorkItem{
			UserId:       userid,
			WorkdateTime: time.Now(),
			TimeType:     timetype,
		}

		// データストアにユーザ構造体を保存する
		_, err = client.Put(ctx, key, workitem)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error saving workitem to datastore: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", 301)

	}
}

func WorkItemEdit(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	client, err := dbConn()
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to connect to dbConn(): %v", err), http.StatusInternalServerError)
		return
	}

	Id := r.URL.Query().Get("id")
	parts := strings.Split(Id, ",")
	id := parts[1]

	intId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid id provided: %v", err), http.StatusBadRequest)
	}

	key := datastore.IDKey("WorkItem", intId, nil)

	var EditRecord WorkItem
	if err := client.Get(ctx, key, &EditRecord); err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve EditRecord: %v", err), http.StatusInternalServerError)
		return
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")
	EditRecord.WorkdateTime = EditRecord.WorkdateTime.In(loc)

	tmpl, err := template.ParseFiles("../templates/edit.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse edit.html: %b", err), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, EditRecord)
}

func WorkItemUpdate(w http.ResponseWriter, r *http.Request) {
	_, err := template.ParseFiles("../templates/edit.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not connect to edit.html: %v", err), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	client, err := dbConn()
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to connect to dbConn(): %v", err), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {

		keyID := r.FormValue("id")
		workdatetime := r.FormValue("workdatetime")
		fmt.Println(workdatetime)
		timetype := r.FormValue("timetype")

		parts := strings.Split(keyID, ",")
		id := parts[1]

		intId, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid id provided: %v", err), http.StatusBadRequest)
			return
		}

		key := datastore.IDKey("WorkItem", intId, nil)

		var EditRecord WorkItem
		if err := client.Get(ctx, key, &EditRecord); err != nil {
			http.Error(w, fmt.Sprintf("Failed to retrieve EditRecord: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Println(EditRecord)

		location, _ := time.LoadLocation("Asia/Tokyo")
		EditRecord.WorkdateTime, _ = time.ParseInLocation("2006-01-02T15:04", workdatetime, location)
		EditRecord.TimeType = timetype

		fmt.Println(EditRecord)

		_, err = client.Put(ctx, key, &EditRecord)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update EditRecord: %v", err), http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/", 301)
}
