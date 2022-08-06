package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/manishmeganathan/go-moibit-client"
	mDocstore "github.com/manishmeganathan/go-moibit-docstore"
	"github.com/rs/cors"
	"log"
	"net/http"
)

type CreateProperty struct {
	PropertyName string `json:"propertyName"`
	UserID       string `json:"userID"`
}

type UpdateStatus struct {
	PropertyName string `json:"propertyName"`
	Status       uint8  `json:"updatedStatus"`
	ProvHash     string `json:"provHash"`
}

type API struct {
	mobClient *moibit.Client
	Ds        *mDocstore.DocStore
	tarDB     *mDocstore.Collection
}

func main() {
	r := mux.NewRouter()
	moibitClient, err := moibit.NewClient(
		"0xadd1ea8849d678b25717f69d6f3dc2db4e7eb539977986ad9049cdbf0"+
			"b18235050c79b1ae95a0aa3a049695c20a1299aa232407f117f8d76f3a2e3e119153bed1b",
		"1659698955350")
	if err != nil {
		log.Fatal("Error initiating the moibit client")
		return
	}
	log.Println("MOI Bit Client Initiated")

	moibitDocstore, err := mDocstore.NewDocStore(moibitClient)
	if err != nil {
		log.Fatal("Error initiating the moibit client")
		return
	}
	log.Println("MOI Bit Docstore Setup")

	tarCollection, err := moibitDocstore.GetCollection("naalebaa")
	if err != nil {
		log.Fatal("Error creating the Database for this project")
		return
	}

	log.Println("Naalebaa Database Created")

	conn := API{
		Ds:        moibitDocstore,
		tarDB:     tarCollection,
		mobClient: moibitClient,
	}

	r.HandleFunc("/", func(resp http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprint(resp, "Hey there! Welcome to Naalebaa Service")
		if err != nil {
			fmt.Println(err)
			http.Error(resp, "Error in accessing root API", http.StatusBadRequest)
			return
		}
	})

	r.HandleFunc("/property/create", func(resp http.ResponseWriter, req *http.Request) {
		createPropertyPayload := CreateProperty{}

		if err := json.NewDecoder(req.Body).Decode(&createPropertyPayload); err != nil {
			fmt.Println(err)
			http.Error(resp, "Error decoding request payload", http.StatusBadRequest)
			return
		}
		fmt.Printf("Property Name: %s\n", createPropertyPayload.PropertyName)

		propertyDocRef, err := conn.tarDB.GetDocument(createPropertyPayload.PropertyName, true)
		if err != nil {
			http.Error(resp, "Error creating the collection docstore", http.StatusBadRequest)
			return
		}

		if err = propertyDocRef.Set(mDocstore.Document{
			"propertyName":  createPropertyPayload.PropertyName,
			"propertyOwner": createPropertyPayload.UserID,
			"activeStep":    0,
		}); err != nil {
			http.Error(resp, "Error in creating the property", http.StatusBadRequest)
			return
		}

		resp.Header().Add("Content-Type", "application/json")
		resp.WriteHeader(http.StatusCreated)
		_, _ = resp.Write([]byte("Property created successfully"))
	}).Methods("POST")

	r.HandleFunc("/property/get", func(resp http.ResponseWriter, req *http.Request) {
		propertyName := req.URL.Query().Get("propertyName")

		propertyDocRef, err := conn.tarDB.GetDocument(propertyName, false)
		if err != nil {
			http.Error(resp, "Error fetching the collection", http.StatusBadRequest)
			return
		}
		propertyDoc, err := propertyDocRef.Get()
		if err != nil {
			http.Error(resp, "Error fetching the property, does not exist", http.StatusBadRequest)
			return
		}

		resp.Header().Add("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
		propertyDetails, _ := propertyDoc.GetJSON()
		_, _ = resp.Write(propertyDetails)
	}).Methods(http.MethodGet)

	r.HandleFunc("/property/updatestatus", func(resp http.ResponseWriter, req *http.Request) {
		upStatusPayload := UpdateStatus{}
		if err := json.NewDecoder(req.Body).Decode(&upStatusPayload); err != nil {
			fmt.Println(err)
			http.Error(resp, "Error decoding request payload", http.StatusBadRequest)
			return
		}

		propertyDocRef, err := conn.tarDB.GetDocument(upStatusPayload.PropertyName, false)
		if err != nil {
			http.Error(resp, "Error fetching the collection", http.StatusBadRequest)
			return
		}
		propertyDoc, err := propertyDocRef.Get()
		if err != nil {
			http.Error(resp, "Error fetching the property, does not exist", http.StatusBadRequest)
			return
		}

		propertyDoc.SetKey("activeStep", upStatusPayload.Status)
		propertyDoc.SetKey("latestProvenanceHash", upStatusPayload.ProvHash)
		resp.Header().Add("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
		err = propertyDocRef.Set(propertyDoc)
		if err != nil {
			http.Error(resp, "Error updating the property details", http.StatusBadRequest)
			return
		}

		_, _ = resp.Write([]byte("Property details updated"))
	}).Methods(http.MethodPut)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)
	log.Println("Listening on 8060 ...")
	log.Fatal(http.ListenAndServe(":8060", handler))
}
