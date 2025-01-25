package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type Car struct {
	ID          int     `json:"Идентификатор"`
	Brand       string  `json:"Марка"`
	Model       string  `json:"Модель"`
	Mileage     float64 `json:"Пробег"`
	OwnersCount int     `json:"Владельцы"`
}

var (
	cars     []Car
	nextID   = 1
	mu       sync.Mutex
	filename = "cars.json"
)

func loadCars() error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cars)
	if err != nil {
		return err
	}
	if len(cars) > 0 {
		nextID = cars[len(cars)-1].ID + 1
	}
	return nil
}

func saveCars() error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(cars)
}

func getCars(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cars)
}

func createCar(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	var car Car
	if err := json.NewDecoder(r.Body).Decode(&car); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	car.ID = nextID
	nextID++
	cars = append(cars, car)
	if err := saveCars(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(car)
}

func getCarByID(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	idStr := r.URL.Path[len("/cars/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}
	for _, car := range cars {
		if car.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(car)
			return
		}
	}
	http.Error(w, "Авто не найден", http.StatusNotFound)
}

func updateCar(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	idStr := r.URL.Path[len("/cars/"):]
	id, err := strconv.Atoi(idStr)

	if err != nil || id <= 0 {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var updatedCar Car
	if err := json.NewDecoder(r.Body).Decode(&updatedCar); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i, car := range cars {
		if car.ID == id {
			updatedCar.ID = id
			isPartialUpdate := true
			if updatedCar.Brand != "" && updatedCar.Model != "" && updatedCar.Mileage > 0 && updatedCar.OwnersCount >= 0 {
				isPartialUpdate = false
			}

			cars[i] = updatedCar

			if err := saveCars(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if isPartialUpdate {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cars[i])
			return
		}
	}

	http.Error(w, "Авто не найден", http.StatusNotFound)
}

func deleteCar(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	idStr := r.URL.Path[len("/cars/"):]
	id, err := strconv.Atoi(idStr)

	if err != nil || id <= 0 {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	for i, car := range cars {
		if car.ID == id {
			cars = append(cars[:i], cars[i+1:]...)
			if err := saveCars(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "Авто не найден", http.StatusNotFound)
}

func deleteAllCars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	cars = []Car{}
	nextID = 1
	if err := saveCars(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	if err := loadCars(); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cars", getCars)
	mux.HandleFunc("/cars/create", createCar)
	mux.HandleFunc("/cars/delete_all", deleteAllCars)

	mux.HandleFunc("/cars/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCarByID(w, r)
		case http.MethodPatch:
			updateCar(w, r)
		case http.MethodPut:
			updateCar(w, r)
		case http.MethodDelete:
			deleteCar(w, r)
		default:
			http.Error(w, "Недопустимый метод", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Сервер работает на: 0228")

	if err := http.ListenAndServe(":0228", mux); err != nil {
		log.Fatal(err)
	}
}
