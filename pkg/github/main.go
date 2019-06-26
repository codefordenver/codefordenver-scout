package github

import (
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"io/ioutil"
	"net/http"
)

type Repository struct {
	Name string `json:"name"`
	Owner struct {
		Name string `json:"name"`
	} `json:"owner"`
}

type RepositoryEvent struct {
	Action string `json:"action"`
	EventRepository Repository `json:"repository"`
}

func Create() (*github.Client, error) {
	tr := http.DefaultTransport
	itr, err := ghinstallation.NewKeyFromFile(tr, 31388, 1101679, "/Users/Zaden/Downloads/cfd-scout.2019-05-23.private-key.pem")
	if err != nil {
		fmt.Println("error creating github key", err)
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: itr})

	return client, nil
}

func HandleRepositoryEvent(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Unmarshal
	var event RepositoryEvent
	err = json.Unmarshal(b, &event)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if event.Action == "create" {
		//handleRepositoryCreate(event.EventRepository)
	}

}
