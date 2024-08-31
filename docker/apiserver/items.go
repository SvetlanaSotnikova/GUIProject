package apiserver

import (
	"fmt"
	"github.com/USER/go-and-compose/storage"
	"net/http"
)

func (s *APIServer) createItem(w http.ResponseWriter, req *http.Request) error {
	item, err := s.storage.CreateItem(req.Context(), storage.CreateItemRequest{
		Name: req.PostFormValue("name"),
	})

	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(fmt.Sprintf("New Item ID: %s", item.ID)))
	return err

}

func (s *APIServer) listItems(w http.ResponseWriter, req *http.Request) error {
	items, err := s.storage.ListItems(req.Context())
	if err != nil {
		return err
	}

	for _, item := range items {
		w.Write([]byte(fmt.Sprintf("%s - %s\n", item.ID, item.Name)))
	}

	return nil
}
