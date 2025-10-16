package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Структуры данных
type Employee struct {
	ID         int       `json:"id" db:"id"`
	LastName   string    `json:"last_name" db:"last_name"`
	FirstName  string    `json:"first_name" db:"first_name"`
	MiddleName string    `json:"middle_name" db:"middle_name"`
	Position   string    `json:"position" db:"position"`
	Phone      string    `json:"phone" db:"phone"`
	Email      string    `json:"email" db:"email"`
	Building   string    `json:"building" db:"building"`
	Comments   string    `json:"comments" db:"comments"`
	Status     string    `json:"status" db:"status"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBTable    string
	DBSSLMode  string
	ServerPort string
}

// Глобальные переменные
var (
	db  *sql.DB
	cfg Config
)

func loadConfig() Config {
	// Загружаем .env файл если существует
	godotenv.Load()

	return Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "phonebook"),
		DBTable:    getEnv("DB_TABLE", "employees"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDB() error {
	// Сначала подключаемся к postgres базе для создания БД если нужно
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)

	adminDb, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}
	defer adminDb.Close()

	// Проверяем существование базы данных
	var dbExists bool
	err = adminDb.QueryRow(`
		SELECT EXISTS(
			SELECT FROM pg_database WHERE datname = $1
		)
	`, cfg.DBName).Scan(&dbExists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %v", err)
	}

	// Создаем базу данных если не существует
	if !dbExists {
		_, err = adminDb.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
		if err != nil {
			return fmt.Errorf("failed to create database: %v", err)
		}
		log.Printf("Database '%s' created successfully", cfg.DBName)
	}

	// Теперь подключаемся к конкретной базе данных
	connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Проверяем соединение
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	// Создаем таблицу если не существует
	err = createTableIfNotExists()
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

func createTableIfNotExists() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			last_name VARCHAR(100) NOT NULL,
			first_name VARCHAR(100) NOT NULL,
			middle_name VARCHAR(100),
			position VARCHAR(200) NOT NULL,
			phone VARCHAR(20) NOT NULL,
			email VARCHAR(150),
			building VARCHAR(50) CHECK (building IN ('Цветоносная', 'Феофанова', 'Везде', 'Удаленный')),
			comments TEXT,
			status VARCHAR(20) CHECK (status IN ('работает', 'уволен', 'внешний')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, cfg.DBTable)

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	// Проверяем есть ли данные в таблице, если нет - добавляем тестовые
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", cfg.DBTable)).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		err = insertSampleData()
		if err != nil {
			return err
		}
		log.Printf("Inserted sample records")
	}

	return nil
}

func insertSampleData() error {
	query := fmt.Sprintf(`
		INSERT INTO %s (last_name, first_name, middle_name, position, phone, email, building, comments, status) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9),
		($10, $11, $12, $13, $14, $15, $16, $17, $18),
		($19, $20, $21, $22, $23, $24, $25, $26, $27),
		($28, $29, $30, $31, $32, $33, $34, $35, $36)
	`, cfg.DBTable)

	_, err := db.Exec(query,
		// Первая запись
		"Иванов", "Иван", "Иванович", "Старший разработчик",
		"+7-999-123-45-67", "ivanov@company.com", "Цветоносная",
		"Team lead backend team", "работает",

		// Вторая запись
		"Петрова", "Мария", "Сергеевна", "Менеджер проектов",
		"+7-999-123-45-68", "petrova@company.com", "Феофанова",
		"PMO department", "работает",

		// Третья запись
		"Сидоров", "Алексей", "Петрович", "Бизнес-аналитик",
		"+7-999-123-45-69", "sidorov@company.com", "Удаленный",
		"Внешний консультант", "внешний",

		// Четвертая запись
		"Козлова", "Ольга", "Владимировна", "Дизайнер",
		"+7-999-123-45-70", "kozlova@company.com", "Везде",
		"UI/UX designer", "работает",
	)

	return err
}

func getEmployees() ([]Employee, error) {
	query := fmt.Sprintf(`
		SELECT id, last_name, first_name, middle_name, position, 
			   phone, email, building, comments, status, created_at 
		FROM %s 
		ORDER BY last_name, first_name
	`, cfg.DBTable)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var emp Employee
		err := rows.Scan(
			&emp.ID, &emp.LastName, &emp.FirstName, &emp.MiddleName,
			&emp.Position, &emp.Phone, &emp.Email, &emp.Building,
			&emp.Comments, &emp.Status, &emp.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		employees = append(employees, emp)
	}

	return employees, nil
}

func searchEmployees(query string) ([]Employee, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT id, last_name, first_name, middle_name, position, 
			   phone, email, building, comments, status, created_at 
		FROM %s 
		WHERE last_name ILIKE $1 OR first_name ILIKE $1 OR 
			  middle_name ILIKE $1 OR position ILIKE $1 OR 
			  phone ILIKE $1 OR email ILIKE $1
		ORDER BY last_name, first_name
	`, cfg.DBTable)

	rows, err := db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var emp Employee
		err := rows.Scan(
			&emp.ID, &emp.LastName, &emp.FirstName, &emp.MiddleName,
			&emp.Position, &emp.Phone, &emp.Email, &emp.Building,
			&emp.Comments, &emp.Status, &emp.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		employees = append(employees, emp)
	}

	return employees, nil
}

func indexHandler(c *gin.Context) {
	employees, err := getEmployees()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Подсчитываем активных сотрудников
	activeCount := 0
	for _, emp := range employees {
		if emp.Status == "работает" {
			activeCount++
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Employees":   employees,
		"ActiveCount": activeCount,
	})
}

func searchHandler(c *gin.Context) {
	query := c.Query("q")

	if query == "" {
		indexHandler(c)
		return
	}

	employees, err := searchEmployees(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Подсчитываем активных сотрудников
	activeCount := 0
	for _, emp := range employees {
		if emp.Status == "работает" {
			activeCount++
		}
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Employees":   employees,
		"SearchQuery": query,
		"ActiveCount": activeCount,
	})
}

func apiEmployeesHandler(c *gin.Context) {
	employees, err := getEmployees()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, employees)
}

func getEmployeeHandler(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	query := fmt.Sprintf(`
		SELECT id, last_name, first_name, middle_name, position, 
			   phone, email, building, comments, status, created_at 
		FROM %s 
		WHERE id = $1
	`, cfg.DBTable)

	var emp Employee
	err = db.QueryRow(query, id).Scan(
		&emp.ID, &emp.LastName, &emp.FirstName, &emp.MiddleName,
		&emp.Position, &emp.Phone, &emp.Email, &emp.Building,
		&emp.Comments, &emp.Status, &emp.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	c.JSON(http.StatusOK, emp)
}

func main() {
	// Загрузка конфигурации
	cfg = loadConfig()

	// Инициализация базы данных
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Настройка Gin в release mode
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Добавляем функцию add в шаблоны
	router.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})

	// Загружаем HTML шаблоны из папки templates
	router.LoadHTMLGlob("templates/*")

	// Статические файлы
	router.Static("/static", "./static")

	// Маршруты
	router.GET("/", indexHandler)
	router.GET("/search", searchHandler)
	router.GET("/api/employees", apiEmployeesHandler)
	router.GET("/api/employees/:id", getEmployeeHandler)

	log.Printf("Server starting on http://localhost:%s", cfg.ServerPort)
	log.Printf("Database: %s, Table: %s", cfg.DBName, cfg.DBTable)
	log.Fatal(router.Run(":" + cfg.ServerPort))
}
