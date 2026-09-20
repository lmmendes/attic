package repository

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/testutil"
)

func TestSavedFilterPersistence(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(testDB.TruncateAll(ctx))
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Home")
	must(err)
	otherOrg, err := fixtures.CreateOrganization(ctx, "Other")
	must(err)
	owner, err := fixtures.CreateUser(ctx, org.ID, "owner@example.com")
	must(err)
	otherUser, err := fixtures.CreateUser(ctx, org.ID, "other@example.com")
	must(err)
	foreignUser, err := fixtures.CreateUser(ctx, otherOrg.ID, "foreign@example.com")
	must(err)
	repo := NewSavedFilterRepository(testDB.Pool)
	var criteria domain.FilterCriteria
	must(json.Unmarshal([]byte(`{"version":1,"q":"retro","attribute_q":"Commodore","expression":{"kind":"group","match":"and","children":[{"kind":"rule","field":"attribute","attribute_id":"f78bff63-88bd-454b-b228-83b07e9cd417","operator":"equals","value":"Commodore"},{"kind":"group","match":"or","children":[{"kind":"rule","field":"collection","operator":"any","values":["b7e50fc4-92ba-4377-909d-1eb0a7735621","0396dd40-c60e-4855-8be8-d53c80653682"]}]}]}}`), &criteria))
	filter := &domain.SavedFilter{OrganizationID: org.ID, UserID: owner.ID, Name: "Favorites", Pinned: true, Criteria: criteria}
	must(repo.Create(ctx, filter))
	if filter.ID == uuid.Nil || filter.CreatedAt.IsZero() || !filter.CreatedAt.Equal(filter.UpdatedAt) {
		t.Fatalf("invalid generated identity or timestamps: %+v", filter)
	}
	loaded, err := repo.GetByID(ctx, org.ID, owner.ID, filter.ID)
	must(err)
	if loaded == nil || !loaded.Pinned || !reflect.DeepEqual(loaded.Criteria, criteria) || loaded.Name != filter.Name || loaded.OrganizationID != org.ID || loaded.UserID != owner.ID || !loaded.CreatedAt.Equal(filter.CreatedAt) || !loaded.UpdatedAt.Equal(filter.UpdatedAt) {
		t.Fatalf("round trip mismatch: %+v", loaded)
	}
	pinned, err := repo.CountPinned(ctx, org.ID, owner.ID)
	must(err)
	if pinned != 1 {
		t.Fatalf("expected one pinned filter, got %d", pinned)
	}
	var storedJSON []byte
	must(testDB.Pool.QueryRow(ctx, `SELECT criteria FROM saved_filters WHERE id = $1`, filter.ID).Scan(&storedJSON))
	var stored map[string]any
	must(json.Unmarshal(storedJSON, &stored))
	if stored["version"] != float64(1) {
		t.Fatalf("version not persisted: %s", storedJSON)
	}
	for _, scope := range []struct {
		name      string
		org, user uuid.UUID
	}{
		{"other user", org.ID, otherUser.ID},
		{"other organization with owner", otherOrg.ID, owner.ID},
		{"foreign user", otherOrg.ID, foreignUser.ID},
	} {
		t.Run(scope.name, func(t *testing.T) {
			list, err := repo.List(ctx, scope.org, scope.user)
			must(err)
			if list == nil || len(list) != 0 {
				t.Fatalf("expected non-nil empty private list: %+v", list)
			}
			hidden, err := repo.GetByID(ctx, scope.org, scope.user, filter.ID)
			must(err)
			if hidden != nil {
				t.Fatal("private filter exposed")
			}
			attempt := *filter
			attempt.OrganizationID, attempt.UserID, attempt.Name = scope.org, scope.user, "Unauthorized"
			if !errors.Is(repo.Update(ctx, &attempt), pgx.ErrNoRows) {
				t.Fatal("private filter update should return ErrNoRows")
			}
			if !errors.Is(repo.Delete(ctx, scope.org, scope.user, filter.ID), pgx.ErrNoRows) {
				t.Fatal("private filter delete should return ErrNoRows")
			}
		})
	}
	loaded, err = repo.GetByID(ctx, org.ID, owner.ID, filter.ID)
	must(err)
	if loaded == nil || loaded.Name != "Favorites" || !loaded.UpdatedAt.Equal(filter.UpdatedAt) {
		t.Fatal("unauthorized writes changed the filter")
	}
	// Backdate creation deterministically so no timing sleeps are needed.
	_, err = testDB.Pool.Exec(ctx, `UPDATE saved_filters SET created_at = NOW() - INTERVAL '1 day' WHERE id = $1`, filter.ID)
	must(err)
	loaded, err = repo.GetByID(ctx, org.ID, owner.ID, filter.ID)
	must(err)
	createdAt, updatedAt := loaded.CreatedAt, loaded.UpdatedAt
	loaded.Name = "Renamed"
	loaded.Pinned = false
	must(json.Unmarshal([]byte(`{"version":2}`), &loaded.Criteria))
	must(repo.Update(ctx, loaded))
	if !loaded.UpdatedAt.After(updatedAt) {
		t.Fatal("update timestamp did not advance")
	}
	reloaded, err := repo.GetByID(ctx, org.ID, owner.ID, filter.ID)
	must(err)
	if reloaded == nil || reloaded.Pinned || reloaded.Name != "Renamed" || !reflect.DeepEqual(reloaded.Criteria, loaded.Criteria) || !reloaded.CreatedAt.Equal(createdAt) || !reloaded.UpdatedAt.Equal(loaded.UpdatedAt) {
		t.Fatalf("update not persisted correctly: %+v", reloaded)
	}
	duplicate := &domain.SavedFilter{ID: uuid.New(), OrganizationID: org.ID, UserID: owner.ID, Name: "Renamed", Criteria: criteria}
	explicitID := duplicate.ID
	must(repo.Create(ctx, duplicate))
	if duplicate.ID != explicitID {
		t.Fatal("explicit ID was replaced")
	}
	list, err := repo.List(ctx, org.ID, owner.ID)
	must(err)
	if len(list) != 2 {
		t.Fatalf("duplicate names should be allowed: %+v", list)
	}
	must(repo.Delete(ctx, org.ID, owner.ID, filter.ID))
	missing, err := repo.GetByID(ctx, org.ID, owner.ID, filter.ID)
	must(err)
	if missing != nil {
		t.Fatal("deleted filter still visible")
	}
	if !errors.Is(repo.Update(ctx, loaded), pgx.ErrNoRows) || !errors.Is(repo.Delete(ctx, org.ID, owner.ID, filter.ID), pgx.ErrNoRows) {
		t.Fatal("missing mutation should return ErrNoRows")
	}
	list, err = repo.List(ctx, org.ID, owner.ID)
	must(err)
	if len(list) != 1 || list[0].ID != duplicate.ID {
		t.Fatal("deletion affected another saved filter")
	}
	for _, name := range []string{"", "   ", strings.Repeat("a", 101)} {
		invalid := &domain.SavedFilter{OrganizationID: org.ID, UserID: owner.ID, Name: name, Criteria: criteria}
		if err := repo.Create(ctx, invalid); err == nil {
			t.Fatalf("invalid name accepted: %q", name)
		}
		attempt := *duplicate
		attempt.Name = name
		if err := repo.Update(ctx, &attempt); err == nil {
			t.Fatalf("invalid name accepted on update: %q", name)
		}
	}
	boundary := &domain.SavedFilter{OrganizationID: org.ID, UserID: owner.ID, Name: strings.Repeat("é", 100), Criteria: criteria}
	must(repo.Create(ctx, boundary))
	for _, scope := range [][2]uuid.UUID{{uuid.New(), owner.ID}, {org.ID, uuid.New()}} {
		invalid := &domain.SavedFilter{OrganizationID: scope[0], UserID: scope[1], Name: "Invalid owner", Criteria: criteria}
		if err := repo.Create(ctx, invalid); err == nil {
			t.Fatal("missing organization/user foreign key accepted")
		}
	}
}

func TestSavedFilterPinnedLimitConcurrent(t *testing.T) {
	ctx := context.Background()
	if err := testDB.TruncateAll(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures := testutil.NewFixtures(testDB.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Concurrent pins")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := fixtures.CreateUser(ctx, org.ID, "pins@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := NewSavedFilterRepository(testDB.Pool)
	start := make(chan struct{})
	errs := make(chan error, 6)
	var ready sync.WaitGroup
	ready.Add(6)
	for i := 0; i < 6; i++ {
		go func(i int) {
			ready.Done()
			<-start
			errs <- repo.Create(ctx, &domain.SavedFilter{
				OrganizationID: org.ID,
				UserID:         owner.ID,
				Name:           "Pinned " + string(rune('A'+i)),
				Pinned:         true,
				Criteria:       domain.FilterCriteria{Version: 1},
			})
		}(i)
	}
	ready.Wait()
	close(start)
	var created, rejected int
	for range 6 {
		switch err := <-errs; {
		case err == nil:
			created++
		case errors.Is(err, ErrPinnedFilterLimit):
			rejected++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if created != 5 || rejected != 1 {
		t.Fatalf("created %d and rejected %d; want 5 and 1", created, rejected)
	}
	pinned, err := repo.CountPinned(ctx, org.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pinned != 5 {
		t.Fatalf("persisted %d pinned filters; want 5", pinned)
	}
}
