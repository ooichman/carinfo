package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Car struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Year         int    `json:"year"`
	Condition    string `json:"condition"`
	Reason       string `json:"reason"`
	Module       string `json:"module"`
	Manufacture  string `json:"manufacture"`
}

func openDB() (*sql.DB, error) {
	dbuser := getEnv("DB_USER", "carinfo")
	dbpassword := getEnv("DB_PASSWORD", "CarInfoPass")
	dbhost := getEnv("DB_HOST", "localhost")
	dbname := getEnv("DB_NAME", "carinfo")
	connstring := dbuser + ":" + dbpassword + "@tcp(" + dbhost + ":3306)/" + dbname
	return sql.Open("mysql", connstring)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
	}
}

func scanCar(scanner interface {
	Scan(dest ...interface{}) error
}) (Car, error) {
	var c Car
	err := scanner.Scan(&c.ID, &c.Name, &c.Year, &c.Condition, &c.Reason, &c.Module, &c.Manufacture)
	return c, err
}

const carSelect = `SELECT c.car_id, c.car_name, c.car_year, c.sell_condition, c.reason,
	COALESCE(v.car_module, ''), v.vendor_name
	FROM cars c
	JOIN cars_vendors v ON c.vendor_id = v.vendor_id`

func getCarByID(db *sql.DB, id int) (Car, error) {
	row := db.QueryRow(carSelect+` WHERE c.car_id = ?`, id)
	return scanCar(row)
}

func findOrCreateVendor(db *sql.DB, manufacture, module string) (int64, error) {
	var vendorID int64
	err := db.QueryRow(`SELECT vendor_id FROM cars_vendors WHERE vendor_name = ?`, manufacture).Scan(&vendorID)
	if err == nil {
		if module != "" {
			_, _ = db.Exec(`UPDATE cars_vendors SET car_module = ? WHERE vendor_id = ?`, module, vendorID)
		}
		return vendorID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO cars_vendors (vendor_name, car_module) VALUES (?, ?)`, manufacture, module)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func listCars(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusBadGateway)
		return
	}
	defer db.Close()

	rows, err := db.Query(carSelect + ` ORDER BY c.car_id`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list cars: %v\n", err)
		http.Error(w, "unable to list cars", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cars := []Car{}
	for rows.Next() {
		c, err := scanCar(rows)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan car: %v\n", err)
			http.Error(w, "unable to list cars", http.StatusInternalServerError)
			return
		}
		cars = append(cars, c)
	}
	writeJSON(w, http.StatusOK, cars)
}

func createCar(w http.ResponseWriter, r *http.Request) {
	var in Car
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Manufacture) == "" {
		http.Error(w, "name and manufacture are required", http.StatusBadRequest)
		return
	}

	db, err := openDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusBadGateway)
		return
	}
	defer db.Close()

	vendorID, err := findOrCreateVendor(db, in.Manufacture, in.Module)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vendor: %v\n", err)
		http.Error(w, "unable to resolve vendor", http.StatusInternalServerError)
		return
	}

	res, err := db.Exec(
		`INSERT INTO cars (vendor_id, car_name, sell_condition, reason, inventory_date, car_year)
		 VALUES (?, ?, ?, ?, CURDATE(), ?)`,
		vendorID, in.Name, in.Condition, in.Reason, in.Year,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "insert car: %v\n", err)
		http.Error(w, "unable to create car", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	car, err := getCarByID(db, int(id))
	if err != nil {
		http.Error(w, "created but unable to reload car", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, car)
}

func getCar(w http.ResponseWriter, r *http.Request, id int) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusBadGateway)
		return
	}
	defer db.Close()

	car, err := getCarByID(db, id)
	if err == sql.ErrNoRows {
		http.Error(w, "car not found", http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "get car: %v\n", err)
		http.Error(w, "unable to get car", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, car)
}

func updateCar(w http.ResponseWriter, r *http.Request, id int) {
	var in Car
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Manufacture) == "" {
		http.Error(w, "name and manufacture are required", http.StatusBadRequest)
		return
	}

	db, err := openDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusBadGateway)
		return
	}
	defer db.Close()

	if _, err := getCarByID(db, id); err == sql.ErrNoRows {
		http.Error(w, "car not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "unable to get car", http.StatusInternalServerError)
		return
	}

	vendorID, err := findOrCreateVendor(db, in.Manufacture, in.Module)
	if err != nil {
		http.Error(w, "unable to resolve vendor", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		`UPDATE cars SET vendor_id = ?, car_name = ?, sell_condition = ?, reason = ?, car_year = ? WHERE car_id = ?`,
		vendorID, in.Name, in.Condition, in.Reason, in.Year, id,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "update car: %v\n", err)
		http.Error(w, "unable to update car", http.StatusInternalServerError)
		return
	}

	car, err := getCarByID(db, id)
	if err != nil {
		http.Error(w, "updated but unable to reload car", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, car)
}

func deleteCar(w http.ResponseWriter, r *http.Request, id int) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "database unavailable", http.StatusBadGateway)
		return
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM cars WHERE car_id = ?`, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delete car: %v\n", err)
		http.Error(w, "unable to delete car", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "car not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func carsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cars")
	path = strings.Trim(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			listCars(w, r)
		case http.MethodPost:
			createCar(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "invalid car id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getCar(w, r, id)
	case http.MethodPut:
		updateCar(w, r, id)
	case http.MethodDelete:
		deleteCar(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
