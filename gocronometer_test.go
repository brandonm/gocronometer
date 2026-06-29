package gocronometer_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jrmycanady/gocronometer"
)

// setup performs some basic actions to set up testing. These are live integration
// tests that hit the real Cronometer API, so the test is skipped when the
// GOCRONOMETER_TEST_USERNAME/PASSWORD env vars are not set (e.g. in CI without
// secrets). Set both to run them against your own account. The returned error is
// always nil on success; it is retained for the existing call-site signatures.
func setup(t *testing.T) (username string, password string, client *gocronometer.Client, err error) {
	t.Helper()
	username = os.Getenv("GOCRONOMETER_TEST_USERNAME")
	password = os.Getenv("GOCRONOMETER_TEST_PASSWORD")

	if username == "" || password == "" {
		t.Skip("set GOCRONOMETER_TEST_USERNAME and GOCRONOMETER_TEST_PASSWORD to run live API tests")
	}

	return username, password, gocronometer.NewClient(nil), nil
}

func TestClient_ObtainAntiCSRF(t *testing.T) {
	_, _, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	antiCSRF, err := client.ObtainAntiCSRF(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if antiCSRF == "" {
		t.Fatalf("the anticsrf value was found to be empty")
	}
}

func TestClient_Login(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())
}

func TestClient_Login_BadCreds(t *testing.T) {
	username, _, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, "BAD"); err == nil {
		t.Fatalf("logged in with bad credentials")
	}
}

func TestClient_GenerateAuthToken(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	token, err := client.GenerateAuthToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if token == "" {
		t.Fatalf("GWT auth token was empty")
	}

	if len(token) > len("2f90aabe5493a07a6d9ab4a17b9ea65e") {
		t.Fatalf("token was %s", token)
	}
}

func TestClient_ExportBiometrics(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportBiometrics(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export bio: %s", err)
	}

}

func TestClient_ExportDailyNutrition(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportDailyNutrition(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export bio: %s", err)
	}
}

func TestClient_ExportNotes(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportNotes(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export bio: %s", err)
	}

}

func TestClient_ExportServings(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportServings(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export bio: %s", err)
	}

}

func TestClient_ExportExercises(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportExercises(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export bio: %s", err)
	}
}

func TestClient_ExportServingsParsed(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportServingsParsed(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export servings parsed: %s", err)
	}

}

func TestClient_ExportDailyNutritionParsed(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportDailyNutrition(context.Background(), startTime, endTime)
	if err != nil {
		t.Fatalf("failed to export daily nutrition parsed: %s", err)
	}

}

func TestClient_ExportExercisesParsed(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportExercisesParsedWithLocation(context.Background(), startTime, endTime, time.UTC)
	if err != nil {
		t.Fatalf("failed to export exercises parsed: %s", err)
	}

}

func TestClient_ExportBiometricRecordsParsed(t *testing.T) {
	username, password, client, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Login(context.Background(), username, password); err != nil {
		t.Fatalf("failed to login: %s", err)
	}

	defer client.Logout(context.Background())

	startTime := time.Date(2021, 6, 1, 0, 0, 0, 0, time.Local)
	endTime := time.Date(2021, 6, 10, 0, 0, 0, 0, time.Local)

	_, err = client.ExportBiometricRecordsParsedWithLocation(context.Background(), startTime, endTime, time.UTC)
	if err != nil {
		t.Fatalf("failed to export bio parsed: %s", err)
	}

}
