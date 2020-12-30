package datastore

import (
	"context"
	"fmt"
	"github.com/genjidb/genji"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var m *MockGenjiDB
var db *genji.DB
var td string

func setupTestCaseForMock(t *testing.T) func(t *testing.T) {
	m = new(MockGenjiDB)
	return func(t *testing.T) {
		m = nil
	}
}

func setupTestCaseForRealDB(t *testing.T) func(t *testing.T) {
	td, _ = ioutil.TempDir("", "db-test")
	db, _ = genji.Open(td + "/my.db")
	db = db.WithContext(context.Background())

	return func(t *testing.T) {
		db.Close()
		os.RemoveAll(td)
		td = ""
	}
}

func TestNilDB(t *testing.T) {
	_, err := NewGenjiDatastore(nil)
	assert.Errorf(t, err, "Proper DB must be provided and not nil")
}

func TestErrorOnDBSetupCreateTable(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)
	m.On("Exec", "CREATE TABLE sessions").Return(fmt.Errorf("Ooops, something went wrong"))
	_, err := NewGenjiDatastore(m)
	assert.Errorf(t, err, "Unable to create sessions table")

	m.MethodCalled("Exec", "CREATE TABLE sessions")
}

func TestSingletonPatternWorks(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)
	m.On("Exec", "CREATE TABLE sessions").Return(nil)
	g1, err := NewGenjiDatastore(m)
	g2, err2 := NewGenjiDatastore(m)
	assert.NoError(t, err)
	assert.NoError(t, err2)

	assert.Equal(t, g1, g2)
}

func TestGenerateTokenWrongLength(t *testing.T) {
	_, err := generateToken(-20)
	assert.Errorf(t, err, "Invalid token length provided: -20, should be >= 20")
}

func TestGenerateTokenWithMinimalSuggestedLength(t *testing.T) {
	token, err := generateToken(20)
	assert.NoError(t, err)
	assert.Equal(t, 20, len(token))
}

func TestGenerateTokenWithProperLength(t *testing.T) {
	token, err := generateToken(32)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(token))
}

func TestCorrectDBSetup(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)
	m.On("Exec", "CREATE TABLE sessions").Return(nil)
	_, err := NewGenjiDatastore(m)
	assert.NoError(t, err)

	m.MethodCalled("Exec", "CREATE TABLE sessions")
}

func TestCorrectDBSetupWithGenji(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)
	_, err := NewGenjiDatastore(db)
	assert.NoError(t, err)
}

func TestCreateSessionFailsDueToExecError(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)
	m.On("Exec", "CREATE TABLE sessions").Return(nil)
	ds, err := NewGenjiDatastore(m)
	assert.NoError(t, err)
	m.On("Exec", "INSERT INTO sessions VALUES ?").Return(fmt.Errorf("Ooops, something went wrong"))
	token, err2 := ds.CreateSession()
	assert.Empty(t, token)
	assert.Errorf(t, err2, "Unable to store session token")
	m.MethodCalled("Exec", "INSERT INTO sessions VALUES ?")
}

func TestCreateSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)
	m.On("Exec", "CREATE TABLE sessions").Return(nil)
	ds, err := NewGenjiDatastore(m)
	assert.NoError(t, err)
	m.On("Exec", "INSERT INTO sessions VALUES ?").Return(nil)
	token, err2 := ds.CreateSession()
	assert.Equal(t, 32, len(token))
	assert.NoError(t, err2)
	m.MethodCalled("Exec", "INSERT INTO sessions VALUES ?")
}

func TestCreateSessionSuccessWithRealDB(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)
	gds, err := NewGenjiDatastore(db)
	assert.NoError(t, err)
	token, err2 := gds.CreateSession()
	assert.NoError(t, err2)
	assert.Equal(t, 32, len(token))
}

func TestJoinSessionSuccessWithRealDB(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)
	gds, err := NewGenjiDatastore(db)
	assert.NoError(t, err)
	token, err2 := gds.CreateSession()
	assert.NoError(t, err2)
	err3 := gds.JoinSession(token, "Bob")
	assert.NoError(t, err3)
}

func TestJoinSessionErrorWhileTryingToAddUserTwiceWithRealDB(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)
	gds, err := NewGenjiDatastore(db)
	assert.NoError(t, err)
	token, err2 := gds.CreateSession()
	assert.NoError(t, err2)
	err3 := gds.JoinSession(token, "Bob")
	assert.NoError(t, err3)
	err4 := gds.JoinSession(token, "Bob")
	assert.Errorf(t, err4, "User with name: Bob already part of session")
}