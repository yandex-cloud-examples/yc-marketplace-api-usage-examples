package db

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	environ "github.com/ydb-platform/ydb-go-sdk-auth-environ"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/query"
	"github.com/ydb-platform/ydb-go-sdk/v3/sugar"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Login             string `sql:"login"`
	PasswordHash      string `sql:"password_hash"`
	ProductInstanceId string `sql:"product_instance_id"`
}

func NewUser(login string, password string) (*User, error) {
	if login == "" || password == "" {
		return nil, fmt.Errorf("Login and password are required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &User{
		Login:        login,
		PasswordHash: string(passwordHash),
	}, nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

type Session struct {
	ID   string `sql:"id"`
	User User   `sql:"login"`
}

func NewSession(user User) (*Session, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	return &Session{ID: id.String(), User: user}, nil
}

type Repo struct {
	db *ydb.Driver
}

func NewRepo() (*Repo, error) {

	connectionString, exists := os.LookupEnv("YDB_CONNECTION_STRING")
	if !exists {
		return nil, fmt.Errorf("YDB_CONNECTION_STRING environment variable is not set")
	}
	ctx := context.Background()
	db, err := ydb.Open(ctx, connectionString,
		environ.WithEnvironCredentials(),
	)
	if err != nil {
		return nil, err
	}
	return &Repo{db: db}, nil
}

func (r *Repo) CreateUser(ctx context.Context, login string, password string) (*User, error) {
	user, err := NewUser(login, password)
	if err != nil {
		return nil, err
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
	return user, err
}

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
		return nil, err
	}
	user, err := sugar.UnmarshallRow[User](row)
	if err != nil {
		return nil, err
	}

	return user, nil
}

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
	return err
}

func (r *Repo) CreateSession(ctx context.Context, user User) (*Session, error) {

	session, err := NewSession(user)
	if err != nil {
		return nil, err
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
	return session, err
}

func (r *Repo) GetSessionWithUser(id string) (*Session, error) {
	row, err := r.db.Query().QueryRow(context.Background(), fmt.Sprintf(`
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
		return nil, err
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
		return nil, err
	}

	return &session, err
}

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
	return err
}

func (r *Repo) Migrate(
	ctx context.Context,
) {
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
		log.Fatalf("Could not create table: %v", err)
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
		log.Fatalf("Could not create table: %v", err)
	}
}
