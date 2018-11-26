package main

import (
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
	"log"
)

func allChanges(ch chan interface{}) {
	go func() {
		for {
			cursor, err := r.Table("items").Changes().Run(session)
			if err != nil {
				log.Fatalln(err)
			}
			var response interface{}
			for cursor.Next(&response) {
				ch <- response
			}

			if cursor.Err() != nil {
				log.Println(cursor.Err())
			}
		}
	}()
}

func activeChanges(ch chan interface{}) {
	go func() {
		for {
			cursor, err := r.Table("items").Filter(r.Row.Field("Status").Eq("active")).Changes().Run(session)
			if err != nil {
				log.Fatalln(err)
			}
			var response interface{}
			for cursor.Next(&response) {
				ch <- response
			}

			if cursor.Err() != nil {
				log.Println(cursor.Err())
			}
		}
	}()
}

func completedChanges(ch chan interface{}) {
	go func() {
		for {
			cursor, err := r.Table("items").Filter(r.Row.Field("Status").Eq("complete")).Changes().Run(session)
			if err != nil {
				log.Fatalln(err)
			}
			var response interface{}
			for cursor.Next(&response) {
				ch <- response
			}

			if cursor.Err() != nil {
				log.Println(cursor.Err())
			}
		}
	}()
}
