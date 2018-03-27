package main

import (
	"log"
	"os"

	"gopkg.in/mgo.v2"
)

func GetCollection() *mgo.Collection {
	session, err := CreateDatabaseSession()
	if err != nil {
		log.Fatal(err)
	}
	c := session.DB(os.Getenv("MONGO_DB")).C(os.Getenv("MONGO_COLLECTION"))
	return c
}

func CreateDatabaseSession() (*mgo.Session, error) {
	session, err := mgo.Dial(os.Getenv("MONGO_HOST"))
	if err != nil {
		return nil, err
	}
	session.SetMode(mgo.Eventual, true)
	creds := &mgo.Credential{
		Username: os.Getenv("MONGO_USER"),
		Password: os.Getenv("MONGO_PASSWD"),
		Source:   os.Getenv("MONGO_DB"),
	}
	err = session.Login(creds)
	if err != nil {
		return nil, err
	}
	return session, nil
}
