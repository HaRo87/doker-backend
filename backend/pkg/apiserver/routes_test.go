package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/genjidb/genji"
	"github.com/haro87/dokerb/pkg/datastore"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"testing"
)

type workPackage struct {
	ID                string  `json:"id"`
	Summary           string  `json:"summary"`
	Effort            float64 `json:"effort"`
	StandardDeviation float64 `json:"standarddeviation"`
}

type estimate struct {
	WorkPackageID  string  `json:"workpackageid"`
	UserName       string  `json:"username"`
	BestCase       float64 `json:"bestcase"`
	MostLikelyCase float64 `json:"mostlikelycase"`
	WorstCase      float64 `json:"worstcase"`
}

type apiResponse struct {
	Message      string        `json:"message"`
	Reason       string        `json:"reason"`
	Route        string        `json:"route"`
	Users        []string      `json:"users"`
	Workpackages []workPackage `json:"workpackages"`
	Estimates    []estimate    `json:"estimates"`
}

type mock struct {
	Method      string
	Call        string
	ReturnValue interface{}
}

type testCase struct {
	Description          string
	Route                string
	Method               string
	Body                 map[string]interface{}
	ExpectedError        bool
	ExpectedErrorMessage string
	ExpectedCode         int
	Mock                 []mock
}

var m *datastore.MockGenjiDB
var db *genji.DB
var td string
var tre *regexp.Regexp

func setupTestCaseForRealDB(t *testing.T) func(t *testing.T) {
	td, _ = ioutil.TempDir("", "db-test")
	db, _ = genji.Open(td + "/my.db")
	db = db.WithContext(context.Background())
	tre, _ = regexp.Compile("/sessions/([\\d|\\w]*)")

	return func(t *testing.T) {
		db.Close()
		os.RemoveAll(td)
		td = ""
	}
}

func setupTestCaseForMock(t *testing.T) func(t *testing.T) {
	m = new(datastore.MockGenjiDB)
	return func(t *testing.T) {
		m = nil
	}
}

func TestAPIRoutesForErrors(t *testing.T) {
	setupAndTearDown := setupTestCaseForMock(t)
	defer setupAndTearDown(t)

	testCases := []testCase{
		{
			Description:          "Create a new Session fails",
			Route:                "/api/sessions",
			Method:               "POST",
			ExpectedError:        false,
			ExpectedErrorMessage: "Unable to store session token",
			ExpectedCode:         500,
			Mock: []mock{
				mock{
					Method:      "Exec",
					Call:        "INSERT INTO sessions VALUES ?",
					ReturnValue: fmt.Errorf("Ooops, something went wrong"),
				},
			},
		},
		{
			Description:          "Deleting a sessions fails due to wrong token length",
			Route:                "/api/sessions/12345",
			Method:               "DELETE",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description: "Adding a user to a session fails due to wrong token length",
			Route:       "/api/sessions/12345/users",
			Method:      "POST",
			Body: map[string]interface{}{
				"name": "Tigger",
			},
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Getting users fails due to wrong token length",
			Route:                "/api/sessions/12345/users",
			Method:               "GET",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Deleting a user fails due to wrong token length",
			Route:                "/api/sessions/12345/users/Tigger",
			Method:               "DELETE",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Getting work packages fails due to wrong token length",
			Route:                "/api/sessions/12345/workpackages",
			Method:               "GET",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description: "Adding a work package to a session fails due to wrong token length",
			Route:       "/api/sessions/12345/workpackages",
			Method:      "POST",
			Body: map[string]interface{}{
				"id": "TEST01",
			},
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Deleting a work package fails due to wrong token length",
			Route:                "/api/sessions/12345/workpackages/TEST01",
			Method:               "DELETE",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description: "Adding a estimates to a work package fails due to wrong token length",
			Route:       "/api/sessions/12345/workpackages/TEST01",
			Method:      "PUT",
			Body: map[string]interface{}{
				"effort": 0.2,
			},
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Deleting a work package estimate fails due to wrong token length",
			Route:                "/api/sessions/12345/workpackages/TEST01/estimate",
			Method:               "DELETE",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description: "Adding a estimates to a session fails due to wrong token length",
			Route:       "/api/sessions/12345/estimates",
			Method:      "POST",
			Body: map[string]interface{}{
				"id":   "TEST01",
				"user": "Tigger",
				"b":    0.5,
				"m":    1.0,
				"w":    2.0,
			},
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Deleting a estimate fails due to wrong token length",
			Route:                "/api/sessions/12345/estimates",
			Method:               "GET",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
		{
			Description:          "Getting estimates fails due to wrong token length",
			Route:                "/api/sessions/12345/estimates/Tigger/TEST01",
			Method:               "DELETE",
			ExpectedError:        false,
			ExpectedErrorMessage: "Session token does not match desired length",
			ExpectedCode:         500,
		},
	}

	m.On("Exec", "CREATE TABLE sessions").Return(nil)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, m).Start()

	for _, test := range testCases {
		var req *http.Request

		if len(test.Body) > 0 {
			body, me := json.Marshal(test.Body)
			assert.NoError(t, me)
			req, _ = http.NewRequest(
				test.Method,
				test.Route,
				bytes.NewBuffer(body),
			)
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, _ = http.NewRequest(
				test.Method,
				test.Route,
				nil,
			)
		}

		if len(test.Mock) > 0 {
			for _, moc := range test.Mock {
				m.On(moc.Method, moc.Call).Return(moc.ReturnValue)
			}
		}

		res, err := app.Test(req, -1)

		assert.Equalf(t, test.ExpectedError, err != nil, test.Description)

		if test.ExpectedErrorMessage != "" {
			var ar apiResponse
			decoder := json.NewDecoder(res.Body)
			derr := decoder.Decode(&ar)
			assert.NoError(t, derr)
			assert.Equal(t, "error", ar.Message)
			assert.Equal(t, test.ExpectedErrorMessage, ar.Reason)
		}

		if test.ExpectedError {
			continue
		}

		assert.Equalf(t, test.ExpectedCode, res.StatusCode, test.Description)
	}
}

func TestSessionGetsCreatedSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]
	assert.Len(t, token, 32)
	assert.Equal(t, "/sessions/"+token, ar.Route)
}

func TestDeleteSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	req, _ = http.NewRequest(
		"DELETE",
		"/api/sessions/"+token,
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
}

func TestAddUserToSessionFailsDueToMissingHeader(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payloadf := map[string]interface{}{
		"name": "Tigger",
	}
	body, me := json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "error", ar.Message)
	assert.Equal(t, "Unprocessable Entity", ar.Reason)
	assert.Equal(t, 400, res.StatusCode)
}

func TestAddUserToSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"name": "Tigger",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "/sessions/"+token+"/users/Tigger", ar.Route)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/users",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Contains(t, ar.Users, "Tigger")
	assert.Len(t, ar.Users, 1)
}

func TestAddUserToSessionFailsDueToUserExists(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"name": "Tigger",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "error", ar.Message)
	assert.Equal(t, "User with name: Tigger already part of session", ar.Reason)
}

func TestAddMultipleUsersToSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"name": "Tigger",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"name": "Rabbit",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/users",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Contains(t, ar.Users, "Tigger")
	assert.Contains(t, ar.Users, "Rabbit")
	assert.Len(t, ar.Users, 2)
}

func TestRemoveUserFromSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"name": "Tigger",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"name": "Rabbit",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/users",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Contains(t, ar.Users, "Tigger")
	assert.Contains(t, ar.Users, "Rabbit")
	assert.Len(t, ar.Users, 2)

	req, _ = http.NewRequest(
		"DELETE",
		"/api/sessions/"+token+"/users/Rabbit",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/users",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Contains(t, ar.Users, "Tigger")
	assert.Len(t, ar.Users, 1)
}

func TestAddWorkPackagesToSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"id":      "TEST02",
		"summary": "some test",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Workpackages[0].ID)
	assert.Empty(t, ar.Workpackages[0].Summary)
	assert.Equal(t, "TEST02", ar.Workpackages[1].ID)
	assert.Equal(t, "some test", ar.Workpackages[1].Summary)
	assert.Len(t, ar.Workpackages, 2)
}

func TestAddWorkPackageToSessionFailsDueToMissingHeader(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "error", ar.Message)
	assert.Equal(t, "Unprocessable Entity", ar.Reason)
	assert.Equal(t, 400, res.StatusCode)
}

func TestRemoveWorkPackageFromSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"id":      "TEST02",
		"summary": "some test",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Workpackages[0].ID)
	assert.Empty(t, ar.Workpackages[0].Summary)
	assert.Equal(t, "TEST02", ar.Workpackages[1].ID)
	assert.Equal(t, "some test", ar.Workpackages[1].Summary)
	assert.Len(t, ar.Workpackages, 2)

	req, _ = http.NewRequest(
		"DELETE",
		"/api/sessions/"+token+"/workpackages/TEST01",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST02", ar.Workpackages[0].ID)
	assert.Equal(t, "some test", ar.Workpackages[0].Summary)
	assert.Len(t, ar.Workpackages, 1)
}

func TestAddAndRemoveEstimateToFromWorkPackageSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"id":      "TEST02",
		"summary": "some test",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Workpackages[0].ID)
	assert.Empty(t, ar.Workpackages[0].Summary)
	assert.Equal(t, 0.0, ar.Workpackages[0].Effort)
	assert.Equal(t, 0.0, ar.Workpackages[0].StandardDeviation)
	assert.Equal(t, "TEST02", ar.Workpackages[1].ID)
	assert.Equal(t, "some test", ar.Workpackages[1].Summary)
	assert.Equal(t, 0.0, ar.Workpackages[1].Effort)
	assert.Equal(t, 0.0, ar.Workpackages[1].StandardDeviation)
	assert.Len(t, ar.Workpackages, 2)

	payloadf := map[string]float64{
		"effort": 1.5,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"PUT",
		"/api/sessions/"+token+"/workpackages/TEST01",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payloadf = map[string]float64{
		"effort":            3.7,
		"standarddeviation": 0.2,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"PUT",
		"/api/sessions/"+token+"/workpackages/TEST02",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Workpackages[0].ID)
	assert.Empty(t, ar.Workpackages[0].Summary)
	assert.Equal(t, 1.5, ar.Workpackages[0].Effort)
	assert.Equal(t, 0.0, ar.Workpackages[0].StandardDeviation)
	assert.Equal(t, "TEST02", ar.Workpackages[1].ID)
	assert.Equal(t, "some test", ar.Workpackages[1].Summary)
	assert.Equal(t, 3.7, ar.Workpackages[1].Effort)
	assert.Equal(t, 0.2, ar.Workpackages[1].StandardDeviation)
	assert.Len(t, ar.Workpackages, 2)

	req, _ = http.NewRequest(
		"DELETE",
		"/api/sessions/"+token+"/workpackages/TEST02/estimate",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/workpackages",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Workpackages[0].ID)
	assert.Empty(t, ar.Workpackages[0].Summary)
	assert.Equal(t, 1.5, ar.Workpackages[0].Effort)
	assert.Equal(t, 0.0, ar.Workpackages[0].StandardDeviation)
	assert.Equal(t, "TEST02", ar.Workpackages[1].ID)
	assert.Equal(t, "some test", ar.Workpackages[1].Summary)
	assert.Equal(t, 0.0, ar.Workpackages[1].Effort)
	assert.Equal(t, 0.0, ar.Workpackages[1].StandardDeviation)
	assert.Len(t, ar.Workpackages, 2)
}

func TestAddEstimateToWorkPackageFailsDueToMissingHeader(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payloadf := map[string]float64{
		"effort": 1.5,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"PUT",
		"/api/sessions/"+token+"/workpackages/TEST01",
		bytes.NewBuffer(body),
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "error", ar.Message)
	assert.Equal(t, "Unprocessable Entity", ar.Reason)
	assert.Equal(t, 400, res.StatusCode)
}

func TestAddEstimateToSessionFailsDueToMissingHeader(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payloadf := map[string]interface{}{
		"id":   "TEST01",
		"user": "Tigger",
		"b":    0.5,
		"m":    1.0,
		"w":    2.0,
	}
	body, me := json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/estimates",
		bytes.NewBuffer(body),
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "error", ar.Message)
	assert.Equal(t, "Unprocessable Entity", ar.Reason)
	assert.Equal(t, 400, res.StatusCode)
}

func TestAddAndRemoveEstimateToFromSessionSuccess(t *testing.T) {
	setupAndTearDown := setupTestCaseForRealDB(t)
	defer setupAndTearDown(t)

	app := NewServer(&Config{
		Static: static{Prefix: "/public", Path: "../../static"},
	}, db).Start()

	req, _ := http.NewRequest(
		"POST",
		"/api/sessions",
		nil,
	)

	res, err := app.Test(req, -1)

	assert.NoError(t, err)

	var ar apiResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	token := tre.FindStringSubmatch(ar.Route)[1]

	payload := map[string]string{
		"id": "TEST01",
	}
	body, me := json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"id":      "TEST02",
		"summary": "some test",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/workpackages",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"name": "Tigger",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payload = map[string]string{
		"name": "Rabbit",
	}
	body, me = json.Marshal(payload)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/users",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payloadf := map[string]interface{}{
		"id":   "TEST01",
		"user": "Tigger",
		"b":    0.5,
		"m":    1.0,
		"w":    2.0,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/estimates",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payloadf = map[string]interface{}{
		"id":   "TEST02",
		"user": "Tigger",
		"b":    0.2,
		"m":    1.2,
		"w":    1.5,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/estimates",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	payloadf = map[string]interface{}{
		"id":   "TEST01",
		"user": "Rabbit",
		"b":    1.0,
		"m":    1.2,
		"w":    2.0,
	}
	body, me = json.Marshal(payloadf)

	assert.NoError(t, me)

	req, _ = http.NewRequest(
		"POST",
		"/api/sessions/"+token+"/estimates",
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/estimates",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Estimates[0].WorkPackageID)
	assert.Equal(t, "Tigger", ar.Estimates[0].UserName)
	assert.Equal(t, 0.5, ar.Estimates[0].BestCase)
	assert.Equal(t, 1.0, ar.Estimates[0].MostLikelyCase)
	assert.Equal(t, 2.0, ar.Estimates[0].WorstCase)
	assert.Equal(t, "TEST02", ar.Estimates[1].WorkPackageID)
	assert.Equal(t, "Tigger", ar.Estimates[1].UserName)
	assert.Equal(t, "TEST01", ar.Estimates[2].WorkPackageID)
	assert.Equal(t, "Rabbit", ar.Estimates[2].UserName)
	assert.Len(t, ar.Estimates, 3)

	req, _ = http.NewRequest(
		"DELETE",
		"/api/sessions/"+token+"/estimates/Rabbit/TEST01",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)

	req, _ = http.NewRequest(
		"GET",
		"/api/sessions/"+token+"/estimates",
		nil,
	)

	res, err = app.Test(req, -1)

	assert.NoError(t, err)

	decoder = json.NewDecoder(res.Body)
	err = decoder.Decode(&ar)
	assert.NoError(t, err)
	assert.Equal(t, "ok", ar.Message)
	assert.Equal(t, "TEST01", ar.Estimates[0].WorkPackageID)
	assert.Equal(t, "Tigger", ar.Estimates[0].UserName)
	assert.Equal(t, 0.5, ar.Estimates[0].BestCase)
	assert.Equal(t, 1.0, ar.Estimates[0].MostLikelyCase)
	assert.Equal(t, 2.0, ar.Estimates[0].WorstCase)
	assert.Equal(t, "TEST02", ar.Estimates[1].WorkPackageID)
	assert.Equal(t, "Tigger", ar.Estimates[1].UserName)
	assert.Len(t, ar.Estimates, 2)
}
