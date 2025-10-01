package immich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAllPeople(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/people" {
			t.Errorf("Expected path /api/people, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got %s", query.Get("page"))
		}
		if query.Get("size") != "500" {
			t.Errorf("Expected size=500, got %s", query.Get("size"))
		}

		// Mock response
		response := PeopleResponseDto{
			HasNextPage: false,
			Hidden:      0,
			Total:       2,
			People: []PersonResponseDto{
				{
					ID:            "person-id-1",
					Name:          "John Doe",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-1.jpg",
				},
				{
					ID:            "person-id-2",
					Name:          "Jane Smith",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-2.jpg",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetAllPeople
	ctx := context.Background()
	result, err := client.GetAllPeople(ctx)
	if err != nil {
		t.Fatalf("GetAllPeople failed: %v", err)
	}

	// Verify response
	if result.Total != 2 {
		t.Errorf("Expected total=2, got %d", result.Total)
	}

	if len(result.People) != 2 {
		t.Errorf("Expected 2 people, got %d", len(result.People))
	}

	if result.People[0].Name != "John Doe" {
		t.Errorf("Expected first person name 'John Doe', got '%s'", result.People[0].Name)
	}

	if result.People[1].Name != "Jane Smith" {
		t.Errorf("Expected second person name 'Jane Smith', got '%s'", result.People[1].Name)
	}
}

func TestGetAllPeopleWithOptions(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		// Check custom parameters
		if query.Get("page") != "2" {
			t.Errorf("Expected page=2, got %s", query.Get("page"))
		}
		if query.Get("size") != "100" {
			t.Errorf("Expected size=100, got %s", query.Get("size"))
		}
		if query.Get("withHidden") != "true" {
			t.Errorf("Expected withHidden=true, got %s", query.Get("withHidden"))
		}
		if query.Get("closestAssetId") != "asset-123" {
			t.Errorf("Expected closestAssetId=asset-123, got %s", query.Get("closestAssetId"))
		}

		// Mock response
		response := PeopleResponseDto{
			HasNextPage: true,
			Hidden:      1,
			Total:       5,
			People: []PersonResponseDto{
				{
					ID:            "person-id-3",
					Name:          "Hidden Person",
					IsHidden:      true,
					ThumbnailPath: "/thumbnails/person-3.jpg",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetAllPeople with options
	ctx := context.Background()
	withHidden := true
	options := GetAllPeopleOptions{
		Page:           2,
		Size:           100,
		WithHidden:     &withHidden,
		ClosestAssetId: "asset-123",
	}

	result, err := client.GetAllPeople(ctx, options)
	if err != nil {
		t.Fatalf("GetAllPeople with options failed: %v", err)
	}

	// Verify response
	if result.Total != 5 {
		t.Errorf("Expected total=5, got %d", result.Total)
	}

	if result.Hidden != 1 {
		t.Errorf("Expected hidden=1, got %d", result.Hidden)
	}

	if !result.HasNextPage {
		t.Error("Expected HasNextPage=true")
	}

	if len(result.People) != 1 {
		t.Errorf("Expected 1 person, got %d", len(result.People))
	}

	if result.People[0].Name != "Hidden Person" {
		t.Errorf("Expected person name 'Hidden Person', got '%s'", result.People[0].Name)
	}

	if !result.People[0].IsHidden {
		t.Error("Expected person to be hidden")
	}
}

func TestGetPersonByName(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response with multiple people
		response := PeopleResponseDto{
			HasNextPage: false,
			Hidden:      0,
			Total:       3,
			People: []PersonResponseDto{
				{
					ID:            "person-id-1",
					Name:          "John Doe",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-1.jpg",
				},
				{
					ID:            "person-id-2",
					Name:          "Jane Smith",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-2.jpg",
				},
				{
					ID:            "person-id-3",
					Name:          "Bob Johnson",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-3.jpg",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetPersonByName - found
	ctx := context.Background()
	person, err := client.GetPersonByName(ctx, "Jane Smith")
	if err != nil {
		t.Fatalf("GetPersonByName failed: %v", err)
	}

	if person == nil {
		t.Fatal("Expected to find person, got nil")
	}

	if person.Name != "Jane Smith" {
		t.Errorf("Expected name 'Jane Smith', got '%s'", person.Name)
	}

	if person.ID != "person-id-2" {
		t.Errorf("Expected ID 'person-id-2', got '%s'", person.ID)
	}

	// Test GetPersonByName - not found
	person, err = client.GetPersonByName(ctx, "Nonexistent Person")
	if err != nil {
		t.Fatalf("GetPersonByName should not fail for missing person: %v", err)
	}

	if person != nil {
		t.Error("Expected nil for nonexistent person")
	}
}

func TestGetPeopleByNames(t *testing.T) {
	// Mock server setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response with multiple people
		response := PeopleResponseDto{
			HasNextPage: false,
			Hidden:      0,
			Total:       3,
			People: []PersonResponseDto{
				{
					ID:            "person-id-1",
					Name:          "John Doe",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-1.jpg",
				},
				{
					ID:            "person-id-2",
					Name:          "Jane Smith",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-2.jpg",
				},
				{
					ID:            "person-id-3",
					Name:          "Bob Johnson",
					IsHidden:      false,
					ThumbnailPath: "/thumbnails/person-3.jpg",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetPeopleByNames
	ctx := context.Background()
	names := []string{"John Doe", "Bob Johnson", "Nonexistent Person"}
	result, err := client.GetPeopleByNames(ctx, names)
	if err != nil {
		t.Fatalf("GetPeopleByNames failed: %v", err)
	}

	// Should find 2 out of 3 people
	if len(result) != 2 {
		t.Errorf("Expected 2 people found, got %d", len(result))
	}

	if result["John Doe"] == nil {
		t.Error("Expected to find 'John Doe'")
	} else if result["John Doe"].ID != "person-id-1" {
		t.Errorf("Expected John Doe ID 'person-id-1', got '%s'", result["John Doe"].ID)
	}

	if result["Bob Johnson"] == nil {
		t.Error("Expected to find 'Bob Johnson'")
	} else if result["Bob Johnson"].ID != "person-id-3" {
		t.Errorf("Expected Bob Johnson ID 'person-id-3', got '%s'", result["Bob Johnson"].ID)
	}

	if result["Nonexistent Person"] != nil {
		t.Error("Did not expect to find 'Nonexistent Person'")
	}

	// Test with empty names
	result, err = client.GetPeopleByNames(ctx, []string{})
	if err != nil {
		t.Fatalf("GetPeopleByNames with empty names failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty names, got %d", len(result))
	}
}

func TestGetAllPeopleValidation(t *testing.T) {
	// Create client (no server needed for validation tests)
	client, err := NewImmichClient("http://localhost", "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test invalid page
	_, err = client.GetAllPeople(ctx, GetAllPeopleOptions{Page: 0})
	if err == nil {
		t.Error("Expected error for page=0")
	}

	// Test invalid size (too small)
	_, err = client.GetAllPeople(ctx, GetAllPeopleOptions{Size: 0})
	if err == nil {
		t.Error("Expected error for size=0")
	}

	// Test invalid size (too large)
	_, err = client.GetAllPeople(ctx, GetAllPeopleOptions{Size: 1001})
	if err == nil {
		t.Error("Expected error for size=1001")
	}
}

func TestGetAllPeopleIterator(t *testing.T) {
	callCount := 0
	// Mock server setup with pagination
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		query := r.URL.Query()
		page := query.Get("page")

		var response PeopleResponseDto

		switch page {
		case "1":
			// First page
			response = PeopleResponseDto{
				HasNextPage: true,
				Hidden:      0,
				Total:       3,
				People: []PersonResponseDto{
					{
						ID:            "person-id-1",
						Name:          "John Doe",
						IsHidden:      false,
						ThumbnailPath: "/thumbnails/person-1.jpg",
					},
					{
						ID:            "person-id-2",
						Name:          "Jane Smith",
						IsHidden:      false,
						ThumbnailPath: "/thumbnails/person-2.jpg",
					},
				},
			}
		case "2":
			// Second page
			response = PeopleResponseDto{
				HasNextPage: false,
				Hidden:      0,
				Total:       3,
				People: []PersonResponseDto{
					{
						ID:            "person-id-3",
						Name:          "Bob Johnson",
						IsHidden:      false,
						ThumbnailPath: "/thumbnails/person-3.jpg",
					},
				},
			}
		default:
			t.Errorf("Unexpected page: %s", page)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client, err := NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test GetAllPeopleIterator
	ctx := context.Background()
	var people []PersonResponseDto

	err = client.GetAllPeopleIterator(ctx, func(person *PersonResponseDto) error {
		people = append(people, *person)
		return nil
	})
	if err != nil {
		t.Fatalf("GetAllPeopleIterator failed: %v", err)
	}

	// Should have made 2 API calls (2 pages)
	if callCount != 2 {
		t.Errorf("Expected 2 API calls, got %d", callCount)
	}

	// Should have collected all 3 people
	if len(people) != 3 {
		t.Errorf("Expected 3 people, got %d", len(people))
	}

	expectedNames := []string{"John Doe", "Jane Smith", "Bob Johnson"}
	for i, person := range people {
		if person.Name != expectedNames[i] {
			t.Errorf("Expected person %d name '%s', got '%s'", i, expectedNames[i], person.Name)
		}
	}
}
