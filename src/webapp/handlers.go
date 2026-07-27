package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

func handleList(w http.ResponseWriter, r *http.Request) {
	cars, err := dbapi.ListCars()
	data := PageData{Title: "Cars", Cars: cars}
	if err != nil {
		log.Printf("list cars: %v", err)
		data.Error = "Unable to load cars from dbapi. Is the API running?"
	}
	render(w, "cars", data)
}

func handleNewForm(w http.ResponseWriter, r *http.Request) {
	render(w, "car_form", PageData{
		Title:      "Add car",
		IsEdit:     false,
		FormAction: "/cars",
		Car:        Car{Year: 2020},
	})
}

func handleEditForm(w http.ResponseWriter, r *http.Request, id int) {
	car, err := dbapi.GetCar(id)
	if err != nil {
		log.Printf("get car %d: %v", id, err)
		render(w, "car_form", PageData{
			Title:      "Edit car",
			IsEdit:     true,
			FormAction: "/cars/" + strconv.Itoa(id),
			Error:      "Unable to load car from dbapi.",
		})
		return
	}
	render(w, "car_form", PageData{
		Title:      "Edit car",
		IsEdit:     true,
		FormAction: "/cars/" + strconv.Itoa(id),
		Car:        car,
	})
}

func carFromForm(r *http.Request) (Car, string) {
	if err := r.ParseForm(); err != nil {
		return Car{}, "Invalid form data."
	}
	year, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("year")))
	car := Car{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Year:        year,
		Condition:   strings.TrimSpace(r.FormValue("condition")),
		Reason:      strings.TrimSpace(r.FormValue("reason")),
		Module:      strings.TrimSpace(r.FormValue("module")),
		Manufacture: strings.TrimSpace(r.FormValue("manufacture")),
	}
	if car.Name == "" || car.Manufacture == "" {
		return car, "Name and manufacture are required."
	}
	return car, ""
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	car, msg := carFromForm(r)
	if msg != "" {
		render(w, "car_form", PageData{
			Title:      "Add car",
			IsEdit:     false,
			FormAction: "/cars",
			Car:        car,
			Error:      msg,
		})
		return
	}
	if _, err := dbapi.CreateCar(car); err != nil {
		log.Printf("create car: %v", err)
		render(w, "car_form", PageData{
			Title:      "Add car",
			IsEdit:     false,
			FormAction: "/cars",
			Car:        car,
			Error:      "Unable to create car via dbapi.",
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleUpdate(w http.ResponseWriter, r *http.Request, id int) {
	car, msg := carFromForm(r)
	car.ID = id
	if msg != "" {
		render(w, "car_form", PageData{
			Title:      "Edit car",
			IsEdit:     true,
			FormAction: "/cars/" + strconv.Itoa(id),
			Car:        car,
			Error:      msg,
		})
		return
	}
	if _, err := dbapi.UpdateCar(id, car); err != nil {
		log.Printf("update car %d: %v", id, err)
		render(w, "car_form", PageData{
			Title:      "Edit car",
			IsEdit:     true,
			FormAction: "/cars/" + strconv.Itoa(id),
			Car:        car,
			Error:      "Unable to update car via dbapi.",
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDelete(w http.ResponseWriter, r *http.Request, id int) {
	if err := dbapi.DeleteCar(id); err != nil {
		log.Printf("delete car %d: %v", id, err)
		cars, _ := dbapi.ListCars()
		render(w, "cars", PageData{
			Title: "Cars",
			Cars:  cars,
			Error: "Unable to delete car via dbapi.",
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
