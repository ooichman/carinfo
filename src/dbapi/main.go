package main

import(
	"fmt"
	"os"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"log"
	"io/ioutil"
	"time"
	"encoding/json"
)

type CarInfoRequest struct {

	Module string `json:module`
	Manufacture string `json:manufacture`
	
}

type CarInfoAnswer struct {

	Name string `json:name`
	Year int `json:year`
	Condition string `json:condition`
	Reason string `json:reason`
	Module string `json:module`
	Manufacture string `json:manufacture`

}

func getEnv(key, fallback string) string {
	value , exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}

	return value
}



func sqlHandler(w http.ResponseWriter, r *http.Request) {

    resDelay := getEnv("SET_DELAY","no")

	dbuser := getEnv("DB_USER","carinfo")
	dbpassword := getEnv("DB_PASSWORD", "CarInfoPass")
	dbhost := getEnv("DB_HOST", "localhost")
	dbname := getEnv("DB_NAME", "carinfo")

	// id:password@tcp(your-mariadb-uri.com:3306)/dbname
	connstring := dbuser + ":" + dbpassword + "@tcp(" + dbhost + ":3306)/" + dbname 

	fmt.Fprintf(os.Stdout, "Starting dbapi with string: %s\n", connstring)
	// creating a database handler , confirm the driver is present 
	db , _ := sql.Open("mysql", connstring)
	defer db.Close()

	// Connect and check the server version

	var Body []byte

	if r.Method != "POST" {
		fmt.Fprintf(os.Stderr, "Serving only a POST request\n")
		http.Error(w, "Serving on;u a POST request\n", http.StatusBadRequest)
	}

	if r.URL.Path != "/query" {
		http.Error(w, "the Provided URL is invalid\n", http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "The provided URL is invalid\n")

	}

	if r.Body != nil {
		if data , err := ioutil.ReadAll(r.Body); err == nil {
			Body = data
		} else {

			fmt.Fprintf(os.Stderr, "Unable to Copy the Body\n")
			http.Error(w,"Unable to copy the Body\n", http.StatusBadRequest)
		}
	}

	var requestdata CarInfoRequest
	if err := json.Unmarshal(Body, &requestdata ); err != nil {
		
		fmt.Fprintf(os.Stderr, "Unable to Unmarshal the body\n")
		http.Error(w, "Unable to Unmarshal the body\n", http.StatusBadRequest)
		return

	} else {
		fmt.Fprintf(os.Stdout,"%+v\n", requestdata)
	}	

	module := requestdata.Module
	manufacture := requestdata.Manufacture

	fmt.Fprintf(os.Stdout,"the module is : %s and the Manufacture is : %s\n", module , manufacture)
	//var carAnswer CarInfoAnswer

	select_q := "select cars.car_name,cars.car_year,cars.sell_condition,cars.reason from cars,cars_vendors where cars.vendor_id = (select cars_vendors.vendor_id where cars_vendors.vendor_name = '" + manufacture + "')"
	
	if resDelay == "yes" {
			time.Sleep(11 * time.Second)
	}

	results , err := db.Query(select_q)

	if err != nil {
		fmt.Fprintf(os.Stdout,"Unable to retrived select reponse\n")
		http.Error(w, "Unable to retrived select reponse\n", http.StatusBadRequest)
	}

	for results.Next() {
		var carAnswer CarInfoAnswer

		err = results.Scan(&carAnswer.Name, &carAnswer.Year,&carAnswer.Condition,&carAnswer.Reason)
		carAnswer.Module = module
		carAnswer.Manufacture = manufacture		
		fmt.Fprintf(os.Stdout, "the details are : %s , %d , %s , %s , %s", carAnswer.Name, carAnswer.Year,carAnswer.Condition,carAnswer.Reason )
	

		sendBody, _ := json.Marshal(carAnswer)	

		

		if _ , werr := w.Write(sendBody); werr != nil {
    		fmt.Fprintf(os.Stderr, "Can't write a response\n")
   			http.Error(w, "Can't write a response\n", http.StatusBadRequest)
    	}
    }
}

func sqlinit() {
	dbuser := getEnv("DB_USER", "carinfo")
	dbpassword := getEnv("DB_PASSWORD", "CarInfoPass")
	dbhost := getEnv("DB_HOST", "localhost")
	dbname := getEnv("DB_NAME", "carinfo")
	connstring := dbuser + ":" + dbpassword + "@tcp(" + dbhost + ":3306)/" + dbname + "?multiStatements=true"

	fmt.Fprintf(os.Stdout, "Starting dbapi with string: %s\n", connstring)

	var db *sql.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = sql.Open("mysql", connstring)
		if err == nil {
			var version string
			err = db.QueryRow("SELECT VERSION()").Scan(&version)
			if err == nil {
				fmt.Fprintf(os.Stdout, "Connected to : %v\n", version)
				break
			}
			db.Close()
		}
		fmt.Fprintf(os.Stderr, "waiting for mariadb (attempt %d/30): %v\n", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("unable to connect to mariadb: %v", err)
	}
	defer db.Close()

	var tablecheck int
	err = db.QueryRow("SELECT COUNT(*) FROM cars_vendors WHERE vendor_id = 1").Scan(&tablecheck)
	if err == nil && tablecheck == 1 {
		fmt.Fprintf(os.Stdout, "schema already initialized\n")
		return
	}

	fmt.Fprintf(os.Stdout, "initializing schema and seed data\n")
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS cars_vendors (
			vendor_id INT AUTO_INCREMENT,
			vendor_name VARCHAR(255) NOT NULL,
			car_module VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY(vendor_id)
		)`,
		`CREATE TABLE IF NOT EXISTS cars (
			car_id INT AUTO_INCREMENT,
			vendor_id INT,
			car_name VARCHAR(255) NOT NULL,
			sell_condition VARCHAR(255),
			reason TEXT,
			inventory_date DATE,
			car_year INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(vendor_id) REFERENCES cars_vendors(vendor_id),
			PRIMARY KEY(car_id)
		)`,
		`INSERT INTO cars_vendors(vendor_name, car_module) VALUES
			('Volvo','B5252S'),('Ford','escort'),('Alfa Romeo','Julietta'),
			('Subaru','impreza'),('Tesla','2012'),('Toyota','Corolla')`,
		`INSERT INTO cars(vendor_id, car_name, sell_condition, reason, inventory_date, car_year) VALUES
			(1,'nency','new','out of the factory',CURDATE(),1983),
			(3,'Jhonny','mid condition','Bad gear',CURDATE(),1985),
			(6,'Barbara','Old','the Engine is not working',CURDATE(),2011)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("schema init failed: %v\nSQL: %s", err, stmt)
		}
	}
	fmt.Fprintf(os.Stdout, "schema initialization complete\n")
}

func main() {

	sqlinit()
	portnum := getEnv("PORT", "8080")
	http.HandleFunc("/query", sqlHandler)
	http.HandleFunc("/cars", carsHandler)
	http.HandleFunc("/cars/", carsHandler)
	log.Printf("Staring HTTP Service on port %v", portnum)
	log.Fatal(http.ListenAndServe(":"+portnum, nil))
}
