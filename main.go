package main

import (
	"encoding/json"
	// "errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	// "strconv"

	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
)

// Todo represents a TODO item
type Todo struct {
    ID          gocql.UUID `json:"id"`
    UserID      string     `json:"user_id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    Status      string     `json:"status"`
    Created     int64      `json:"created"`
    Updated     int64      `json:"updated"`
}

var session *gocql.Session

func init() {
    // Connect to ScyllaDB
    cluster := gocql.NewCluster("go-scylla1-1")
    cluster.Keyspace = "todo_app"
    var err error
    session, err = cluster.CreateSession()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Connected to ScyllaDB")
}

func main() {
    defer session.Close()

    router := mux.NewRouter()

    // Define API endpoints
    router.HandleFunc("/todos", createTodo).Methods("POST")
    router.HandleFunc("/todos/{id}", getTodo).Methods("GET")
    router.HandleFunc("/todos/{id}", updateTodo).Methods("PUT")
    router.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE")
    router.HandleFunc("/todos", listTodos).Methods("GET")

    log.Fatal(http.ListenAndServe(":8080", router))
}

func createTodo(w http.ResponseWriter, r *http.Request) {
    var todo Todo
    err := json.NewDecoder(r.Body).Decode(&todo)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate todo fields
    if todo.Title == "" {
        http.Error(w, "Title is required", http.StatusBadRequest)
        return
    }
    if todo.UserID == "" {
        http.Error(w, "UserID is required", http.StatusBadRequest)
        return
    }

    // Generate a UUID for the new todo item
    todo.ID = gocql.TimeUUID()

    // Set created and updated timestamps
    currentTime := time.Now().Unix()
    todo.Created = currentTime
    todo.Updated = currentTime

    // Insert the todo item into the database
    err = session.Query(
        "INSERT INTO todos (id, user_id, title, description, status, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?)",
        todo.ID, todo.UserID, todo.Title, todo.Description, todo.Status, todo.Created, todo.Updated,
    ).Exec()
    if err != nil {
        log.Println("Error inserting todo item:", err)
        http.Error(w, "Failed to create TODO item", http.StatusInternalServerError)
        return
    }

    // Respond with the newly created todo item
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(todo)
}

func getTodo(w http.ResponseWriter, r *http.Request) {
    // Extract todo ID from URL parameters
    vars := mux.Vars(r)
    todoID := vars["id"]

    // Parse todo ID into UUID
    id, err := gocql.ParseUUID(todoID)
    if err != nil {
        http.Error(w, "Invalid todo ID", http.StatusBadRequest)
        return
    }

    // Query the database to retrieve the todo item by ID
    var todo Todo
    err = session.Query("SELECT id, user_id, title, description, status, created, updated FROM todos WHERE id = ?", id).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Status, &todo.Created, &todo.Updated)
    if err != nil {
        if err == gocql.ErrNotFound {
            http.Error(w, "Todo item not found", http.StatusNotFound)
        } else {
            log.Println("Error retrieving todo item:", err)
            http.Error(w, "Failed to retrieve TODO item", http.StatusInternalServerError)
        }
        return
    }

    // Respond with the retrieved todo item
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(todo)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
    // Extract todo ID from URL parameters
    vars := mux.Vars(r)
    todoID := vars["id"]

    // Parse todo ID into UUID
    id, err := gocql.ParseUUID(todoID)
    if err != nil {
        http.Error(w, "Invalid todo ID", http.StatusBadRequest)
        return
    }

    // Decode the request body into a Todo struct
    var updatedTodo Todo
    err = json.NewDecoder(r.Body).Decode(&updatedTodo)
    if err != nil {
        http.Error(w, "Failed to decode request body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    // Validate updated fields
    if updatedTodo.ID != id {
        http.Error(w, "Todo ID mismatch", http.StatusBadRequest)
        return
    }

    // Query the database to update the todo item by ID
    query := "UPDATE todos SET title = ?, description = ?, status = ?, updated = ? WHERE id = ?"
    err = session.Query(query, updatedTodo.Title, updatedTodo.Description, updatedTodo.Status, time.Now(), id).Exec()
    if err != nil {
        log.Println("Error updating todo item:", err)
        http.Error(w, "Failed to update TODO item", http.StatusInternalServerError)
        return
    }

    // Respond with a success message
    w.WriteHeader(http.StatusNoContent)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
    // Extract todo ID from URL parameters
    vars := mux.Vars(r)
    todoID := vars["id"]

    // Parse todo ID into UUID
    id, err := gocql.ParseUUID(todoID)
    if err != nil {
        http.Error(w, "Invalid todo ID", http.StatusBadRequest)
        return
    }

    // Query the database to delete the todo item by ID
    query := "DELETE FROM todos WHERE id = ?"
    err = session.Query(query, id).Exec()
    if err != nil {
        log.Println("Error deleting todo item:", err)
        http.Error(w, "Failed to delete TODO item", http.StatusInternalServerError)
        return
    }

    // Respond with a success message
    w.WriteHeader(http.StatusNoContent)
}

func listTodos(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    queryParams := r.URL.Query()
    pageStr := queryParams.Get("page")
    sizeStr := queryParams.Get("size")
    status := queryParams.Get("status")
    sort := queryParams.Get("sort")

    // Default values for pagination
    page := 1
    size := 10

    // Parse page and size parameters
    if pageStr != "" {
        parsedPage, err := strconv.Atoi(pageStr)
        if err != nil {
            http.Error(w, "Invalid page number", http.StatusBadRequest)
            return
        }
        page = parsedPage
    }
    if sizeStr != "" {
        parsedSize, err := strconv.Atoi(sizeStr)
        if err != nil {
            http.Error(w, "Invalid page size", http.StatusBadRequest)
            return
        }
        size = parsedSize
    }

    // Construct the CQL query based on the query parameters
    query := "SELECT * FROM todos"
    if status != "" {
        query += " WHERE status = '" + status + "'"
    }
    if sort != "" {
        // Implement sorting based on the provided sort parameter
        sort="created_at_asc"
    }
    query += " LIMIT ? OFFSET ?"

    // Execute the query against the database
    var todoList []Todo
    err := session.Query(query, size, (page-1)*size).Scan(&todoList)
    if err != nil {
        log.Println("Error querying todo items:", err)
        http.Error(w, "Failed to fetch TODO items", http.StatusInternalServerError)
        return
    }

    // Marshal the result into JSON and send it in the response
    jsonResponse, err := json.Marshal(todoList)
    if err != nil {
        log.Println("Error marshalling JSON response:", err)
        http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(jsonResponse)
}