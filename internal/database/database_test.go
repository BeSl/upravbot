package database

import (
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Test database creation
	db, err := New(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Error("Expected database instance, got nil")
	}
}

func TestNewInvalidPath(t *testing.T) {
	// Test with invalid path
	_, err := New("/invalid/path/test.db")
	if err == nil {
		t.Error("Expected error for invalid path, but got none")
	}
}

func TestCreateOrUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	user := &User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test creating user
	err := db.CreateOrUpdateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test updating user
	user.FirstName = "Updated"
	user.IsAdmin = true
	err = db.CreateOrUpdateUser(user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify user was updated
	retrievedUser, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.FirstName != "Updated" {
		t.Errorf("Expected FirstName 'Updated', got '%s'", retrievedUser.FirstName)
	}
	if !retrievedUser.IsAdmin {
		t.Error("Expected IsAdmin true, got false")
	}
}

func TestGetUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	user := &User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create user first
	err := db.CreateOrUpdateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting existing user
	retrievedUser, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, retrievedUser.ID)
	}
	if retrievedUser.Username != user.Username {
		t.Errorf("Expected Username '%s', got '%s'", user.Username, retrievedUser.Username)
	}

	// Test getting non-existing user
	_, err = db.GetUser(999999999)
	if err == nil {
		t.Error("Expected error for non-existing user, but got none")
	}
}

func TestGetAllUsers(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	// Clear existing data
	db.conn.Exec("DELETE FROM users")

	users := []*User{
		{
			ID:        111111111,
			Username:  "user1",
			FirstName: "User",
			LastName:  "One",
			IsAdmin:   false,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        222222222,
			Username:  "user2",
			FirstName: "User",
			LastName:  "Two",
			IsAdmin:   true,
			IsActive:  true,
			CreatedAt: time.Now().Add(time.Hour),
			UpdatedAt: time.Now().Add(time.Hour),
		},
	}

	// Create users
	for _, user := range users {
		err := db.CreateOrUpdateUser(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Get all users
	allUsers, err := db.GetAllUsers()
	if err != nil {
		t.Fatalf("Failed to get all users: %v", err)
	}

	if len(allUsers) != 2 {
		t.Errorf("Expected 2 users, got %d", len(allUsers))
	}

	// Check if users are ordered by created_at DESC (newest first)
	if len(allUsers) >= 2 && allUsers[0].CreatedAt.Before(allUsers[1].CreatedAt) {
		t.Error("Expected users to be ordered by created_at DESC")
	}
}

// Helper functions
func setupTestDB(t *testing.T) *DB {
	tmpFile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	db, err := New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func teardownTestDB(t *testing.T, db *DB) {
	db.Close()
	// Note: The actual file cleanup is handled by the test framework
}
