package db

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	environ "github.com/ydb-platform/ydb-go-sdk-auth-environ"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
	"github.com/ydb-platform/ydb-go-sdk/v3/sugar"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system.
type User struct {
	Login             string `sql:"login"`
	PasswordHash      string `sql:"password_hash"`
	ProductInstanceId string `sql:"product_instance_id"`
}

// NewUser creates a new User instance with a hashed password.
// It requires a non-empty login and password.
func NewUser(login string, password string) (*User, error) {
	if login == "" || password == "" {
		return nil, fmt.Errorf("login and password are required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("could not generate password hash: %w", err)
	}
	return &User{
		Login:        login,
		PasswordHash: string(passwordHash),
	}, nil
}

// CheckPassword verifies if the provided password matches the user's hashed password.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// Session represents a user session.
type Session struct {
	ID   string `sql:"id"`
	User User   `sql:"login"` // User holds the associated user data.
}

// NewSession creates a new Session for a given user.
func NewSession(user User) (*Session, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("could not generate session ID: %w", err)
	}
	return &Session{ID: id.String(), User: user}, nil
}

// Repository defines the interface for database operations related to users and sessions.
// This allows for easier mocking and testing.
type Repository interface {
	CreateUser(ctx context.Context, login string, password string) (*User, error)
	GetUser(ctx context.Context, login string) (*User, error)
	UpdateUser(ctx context.Context, login string, productInstanceId string) error
	CreateSession(ctx context.Context, user User) (*Session, error)
	GetSessionWithUser(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	Migrate(ctx context.Context) error
	Close() error // Added a Close method to the interface for proper resource management
}

// Repo handles database operations for users and sessions.
// It implements the Repository interface.
type Repo struct {
	db *ydb.Driver
}

// NewRepo creates a new Repo instance and connects to the YDB database.
// It requires the YDB_CONNECTION_STRING environment variable to be set.
// It returns an implementation of the Repository interface.
func NewRepo() (Repository, error) {
	connectionString, exists := os.LookupEnv("YDB_CONNECTION_STRING")
	if !exists {
		return nil, fmt.Errorf("YDB_CONNECTION_STRING environment variable is not set")
	}
	ctx := context.Background() // Use background context for initial setup
	db, err := ydb.Open(ctx, connectionString,
		environ.WithEnvironCredentials(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not open YDB connection: %w", err)
	}
	return &Repo{db: db}, nil
}

// Close closes the database connection.
func (r *Repo) Close() error {
	if r.db != nil {
		return r.db.Close(context.Background()) // Use a background context for closing
	}
	return nil
}

// CreateUser creates a new user in the database.
func (r *Repo) CreateUser(ctx context.Context, login string, password string) (*User, error) {
	user, err := NewUser(login, password)
	if err != nil {
		return nil, fmt.Errorf("could not prepare user data: %w", err)
	}
	err = r.db.Query().Exec(ctx, fmt.Sprintf(`
		DECLARE $login AS Utf8;
		DECLARE	$password_hash AS Utf8;
		
		INSERT INTO %s (login, password_hash) VALUES($login, $password_hash);`,
		"`users`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$login").Text(user.Login).
			Param("$password_hash").Text(user.PasswordHash).
			Build(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create user in database: %w", err)
	}
	return user, nil
}

// GetUser retrieves a user from the database by login.
func (r *Repo) GetUser(ctx context.Context, login string) (*User, error) {
	row, err := r.db.Query().QueryRow(ctx, fmt.Sprintf(`
		DECLARE $login AS Utf8;
		
		SELECT * FROM %s WHERE login = $login;`,
		"`users`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$login").Text(login).
			Build(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query user from database: %w", err)
	}
	user, err := sugar.UnmarshallRow[User](row)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall user data: %w", err)
	}

	return user, nil
}

// UpdateUser updates the product_instance_id for a given user.
func (r *Repo) UpdateUser(ctx context.Context, login string, productInstanceId string) error {
	err := r.db.Query().Exec(ctx, fmt.Sprintf(`
		DECLARE $login AS Utf8;
		DECLARE	$password_hash AS Utf8;
		
		UPDATE %s SET product_instance_id = $product_instance_id
		WHERE login = $login;`,
		"`users`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$login").Text(login).
			Param("$product_instance_id").Text(productInstanceId).
			Build(),
		),
	)
	if err != nil {
		return fmt.Errorf("could not update user in database: %w", err)
	}
	return nil
}

// CreateSession creates a new session for a user in the database.
func (r *Repo) CreateSession(ctx context.Context, user User) (*Session, error) {
	session, err := NewSession(user)
	if err != nil {
		return nil, fmt.Errorf("could not prepare session data: %w", err)
	}

	err = r.db.Query().Exec(ctx, fmt.Sprintf(`
		DECLARE $id AS Utf8;
		DECLARE $login AS Utf8;
		
		INSERT INTO %s (id, login) VALUES($id, $login);`,
		"`sessions`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$id").Text(session.ID).
			Param("$login").Text(user.Login).
			Build(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create session in database: %w", err)
	}
	return session, nil
}

// GetSessionWithUser retrieves a session and its associated user data by session ID.
func (r *Repo) GetSessionWithUser(ctx context.Context, id string) (*Session, error) {
	row, err := r.db.Query().QueryRow(ctx, fmt.Sprintf(`
		PRAGMA SimpleColumns;
		DECLARE $id AS Utf8;
		
		SELECT * FROM %s AS s
		JOIN %s AS u ON s.login = u.login
		WHERE id = $id;`,
		"`sessions`", "`users`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$id").Text(id).
			Build(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query session with user from database: %w", err)
	}
	session := Session{
		User: User{},
	}
	err = row.ScanNamed(
		query.Named("id", &session.ID),
		query.Named("login", &session.User.Login),
		query.Named("password_hash", &session.User.PasswordHash),
		query.Named("product_instance_id", &session.User.ProductInstanceId),
	)
	if err != nil {
		return nil, fmt.Errorf("could not scan session and user data: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a session from the database by ID.
func (r *Repo) DeleteSession(ctx context.Context, id string) error {
	err := r.db.Query().Exec(ctx, fmt.Sprintf(`
		DECLARE $id AS Utf8;
		
		DELETE FROM %s WHERE id = $id;`,
		"`sessions`"),
		query.WithParameters(ydb.ParamsBuilder().
			Param("$id").Text(id).
			Build(),
		),
	)
	if err != nil {
		return fmt.Errorf("could not delete session from database: %w", err)
	}
	return nil
}

// Migrate creates the necessary database tables (users, sessions) if they don't already exist.
func (r *Repo) Migrate(
	ctx context.Context,
) error {
	err := r.db.Query().Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			login Utf8,
			password_hash Utf8,
			product_instance_id Utf8,
			
			PRIMARY KEY(login)
		)`, "`users`"),
		query.WithTxControl(query.NoTx()),
	)
	if err != nil {
		return fmt.Errorf("could not create table 'users': %w", err)
	}
	err = r.db.Query().Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id Utf8,
			login Utf8,
			
			PRIMARY KEY(id)
		)`, "`sessions`"),
		query.WithTxControl(query.NoTx()),
	)
	if err != nil {
		return fmt.Errorf("could not create table 'sessions': %w", err)
	}
	return nil
}
