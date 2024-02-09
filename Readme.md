Crud application
TODO API using Golang and ScyllaDB that supports basic CRUD operations and
includes pagination functionality for the list endpoint.


Endpoints:

[.] router.HandleFunc("/todos", createTodo).Methods("POST")

[.] router.HandleFunc("/todos/{id}", getTodo).Methods("GET")

[.] router.HandleFunc("/todos/{id}", updateTodo).Methods("PUT")

[.] router.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE")

[.] router.HandleFunc("/todos", listTodos).Methods("GET")


Requirements

● Set up a Golang project and integrate ScyllaDB as the database for storing TODO items.
Ensure that items in the database are stored user-wise.

● Implement endpoints for creating, reading, updating, and deleting TODO items for a
single user at a time. Each TODO item should have at least the following properties: id, user_id, title description, status, created, updated.

● Implement a paginated list endpoint to retrieve TODO items.

● Provide support for filtering based on TODO item status (e.g., pending, completed).