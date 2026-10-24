package handler

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lmmendes/attic/internal/domain"
	"github.com/lmmendes/attic/internal/repository"
	"github.com/lmmendes/attic/internal/testutil"
)

type locationPhotoStorage struct {
	objects               map[string][]byte
	next                  int
	failUpload, failURL   bool
	filename, contentType string
	onUpload              func()
}

func (s *locationPhotoStorage) Upload(_ context.Context, name, contentType string, body io.Reader) (string, error) {
	if s.failUpload {
		return "", errors.New("upload unavailable")
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	s.next++
	key := fmt.Sprintf("photo-%d", s.next)
	s.objects[key] = data
	s.filename = name
	s.contentType = contentType
	if s.onUpload != nil {
		s.onUpload()
	}
	return key, nil
}
func (s *locationPhotoStorage) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	if s.failURL {
		return "", errors.New("signing unavailable")
	}
	return "https://storage.test/" + key, nil
}
func (s *locationPhotoStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

func TestLocationPhotoHTTPIntegration(t *testing.T) {
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	db, err := testutil.NewTestDB(ctx)
	must(err)
	defer db.Close(ctx)
	fixtures := testutil.NewFixtures(db.Pool)
	org, err := fixtures.CreateOrganization(ctx, "Photo locations")
	must(err)
	otherOrg, err := fixtures.CreateOrganization(ctx, "Other organization")
	must(err)
	parent, err := fixtures.CreateLocation(ctx, org.ID, "Garage", nil)
	must(err)
	loc, err := fixtures.CreateLocation(ctx, org.ID, "Box B3", &parent.ID)
	must(err)
	foreign, err := fixtures.CreateLocation(ctx, otherOrg.ID, "Foreign box", nil)
	must(err)
	repo := repository.NewLocationRepository(db.Pool)
	store := &locationPhotoStorage{objects: map[string][]byte{}}
	h := New(nil, &Repositories{Locations: repo, Organizations: repository.NewOrganizationRepository(db.Pool)}, store, org.ID)
	router := chi.NewRouter()
	router.Use(h.RequireFeature("locations"))
	router.Get("/api/locations", h.ListLocations)
	router.Get("/api/locations/{id}", h.GetLocation)
	router.Delete("/api/locations/{id}", h.DeleteLocation)
	router.Get("/api/locations/{id}/photos", h.ListLocationPhotos)
	router.Post("/api/locations/{id}/photos", h.UploadLocationPhoto)
	router.Delete("/api/locations/{id}/photos/{photoId}", h.DeleteLocationPhoto)
	router.Put("/api/locations/{id}/photos/{photoId}/cover", h.SetLocationCoverPhoto)
	router.Delete("/api/locations/{id}/photos/{photoId}/cover", h.UnsetLocationCoverPhoto)
	path := "/api/locations/" + loc.ID.String()
	request := func(method, path string, body io.Reader, contentType string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, body)
		r.Header.Set("Content-Type", contentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body)
		}
		return w
	}
	var buffer bytes.Buffer
	must(png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	imageData := buffer.Bytes()
	upload := func(id string, data []byte, want int) domain.LocationPhoto {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "../../box.svg")
		must(err)
		_, err = part.Write(data)
		must(err)
		must(writer.Close())
		response := request("POST", "/api/locations/"+id+"/photos", &body, writer.FormDataContentType(), want)
		var photo domain.LocationPhoto
		if want == 201 {
			must(json.Unmarshal(response.Body.Bytes(), &photo))
			if strings.Contains(response.Body.String(), "file_key") {
				t.Fatal("storage key exposed")
			}
		}
		return photo
	}
	list := func() []domain.LocationPhoto {
		t.Helper()
		var photos []domain.LocationPhoto
		must(json.Unmarshal(request("GET", path+"/photos", nil, "", 200).Body.Bytes(), &photos))
		return photos
	}
	photoPath := func(id uuid.UUID) string { return path + "/photos/" + id.String() }
	migration := func(direction string) error {
		sql, err := os.ReadFile("../../migrations/000033_location_photo_gallery." + direction + ".sql")
		if err != nil {
			return err
		}
		_, err = db.Pool.Exec(ctx, string(sql))
		return err
	}
	// Exercise migration of the previously applied single-photo feature in place.
	must(migration("down"))
	_, err = db.Pool.Exec(ctx, "UPDATE locations SET photo_key='legacy-photo' WHERE id=$1", loc.ID)
	must(err)
	store.objects["legacy-photo"] = imageData
	must(migration("up"))
	legacy := list()
	if len(legacy) != 1 || !legacy[0].IsCover || legacy[0].URL != "https://storage.test/legacy-photo" {
		t.Fatalf("legacy photo lost: %+v", legacy)
	}
	request("DELETE", photoPath(legacy[0].ID), nil, "", 204)
	if len(store.objects) != 0 {
		t.Fatal("legacy file not cleaned up")
	}
	a := upload(loc.ID.String(), imageData, 201)
	b := upload(loc.ID.String(), imageData, 201)
	c := upload(loc.ID.String(), imageData, 201)
	if a.IsCover || b.IsCover || c.IsCover || store.filename != "location-photo.png" || store.contentType != "image/png" {
		t.Fatal("upload selected a cover or used incorrect storage metadata")
	}
	photos := list()
	if len(photos) != 3 || photos[0].ID != a.ID {
		t.Fatal("uploads replaced existing photos or changed order")
	}
	if err := migration("down"); err == nil {
		t.Fatal("rollback silently discarded multiple photos")
	}
	if len(list()) != 3 {
		t.Fatal("refused rollback changed gallery")
	}
	assertNoCover := func() {
		t.Helper()
		for _, photo := range list() {
			if photo.IsCover {
				t.Fatal("unexpected automatic cover")
			}
		}
		var location domain.Location
		must(json.Unmarshal(request("GET", path, nil, "", 200).Body.Bytes(), &location))
		if location.PhotoURL != nil {
			t.Fatal("location icon replaced without explicit cover")
		}
	}
	assertNoCover()
	request("PUT", photoPath(c.ID)+"/cover", nil, "", 204)
	photos = list()
	if photos[0].ID != c.ID || !photos[0].IsCover {
		t.Fatal("cover not selected")
	}
	for _, url := range []string{path, "/api/locations", "/api/locations?tree=true"} {
		response := request("GET", url, nil, "", 200)
		if !strings.Contains(response.Body.String(), c.URL) || strings.Contains(response.Body.String(), "photo_key") {
			t.Fatal("cover missing or key exposed", response.Body)
		}
	}
	request("DELETE", photoPath(c.ID)+"/cover", nil, "", 204)
	assertNoCover()
	request("DELETE", photoPath(c.ID)+"/cover", nil, "", 204)
	if len(list()) != 3 || len(store.objects) != 3 {
		t.Fatal("removing cover deleted a photo")
	}
	request("PUT", photoPath(a.ID)+"/cover", nil, "", 204)
	request("PUT", photoPath(uuid.New())+"/cover", nil, "", 404)
	request("DELETE", photoPath(uuid.New())+"/cover", nil, "", 404)
	if list()[0].ID != a.ID {
		t.Fatal("failed cover selection cleared existing cover")
	}
	request("DELETE", photoPath(b.ID)+"/cover", nil, "", 204)
	if !list()[0].IsCover || list()[0].ID != a.ID {
		t.Fatal("unsetting another photo cleared the selected cover")
	}
	request("DELETE", path+"/photos/invalid/cover", nil, "", 400)
	request("DELETE", photoPath(uuid.New()), nil, "", 404)
	request("DELETE", path+"/photos/invalid", nil, "", 400)
	// Photo IDs from another location, even within this workspace, cannot be used.
	foreignPhoto := &domain.LocationPhoto{LocationID: foreign.ID, FileKey: "foreign-file", FileName: "Foreign"}
	must(repo.AddPhoto(ctx, otherOrg.ID, foreignPhoto))
	request("PUT", photoPath(foreignPhoto.ID)+"/cover", nil, "", 404)
	request("DELETE", photoPath(foreignPhoto.ID)+"/cover", nil, "", 404)
	request("DELETE", photoPath(foreignPhoto.ID), nil, "", 404)
	parentPhoto := &domain.LocationPhoto{LocationID: parent.ID, FileKey: "parent-file", FileName: "Parent"}
	must(repo.AddPhoto(ctx, org.ID, parentPhoto))
	request("PUT", photoPath(parentPhoto.ID)+"/cover", nil, "", 404)
	request("DELETE", photoPath(parentPhoto.ID)+"/cover", nil, "", 404)
	request("DELETE", photoPath(parentPhoto.ID), nil, "", 404)
	upload(foreign.ID.String(), imageData, 404)
	upload("invalid", imageData, 400)
	upload(uuid.New().String(), imageData, 404)
	request("GET", "/api/locations/"+foreign.ID.String()+"/photos", nil, "", 404)
	for _, data := range [][]byte{nil, []byte("<svg></svg>"), []byte("not an image"), make([]byte, maxLocationPhotoSize+1)} {
		upload(loc.ID.String(), data, 400)
	}
	huge := append([]byte(nil), imageData...)
	binary.BigEndian.PutUint32(huge[16:20], maxLocationPhotoPixels)
	binary.BigEndian.PutUint32(huge[20:24], 2)
	binary.BigEndian.PutUint32(huge[29:33], crc32.ChecksumIEEE(huge[12:29]))
	upload(loc.ID.String(), huge, 400)
	store.failUpload = true
	upload(loc.ID.String(), imageData, 500)
	store.failUpload = false
	store.failURL = true
	upload(loc.ID.String(), imageData, 500)
	store.failURL = false
	if len(store.objects) != 3 || len(list()) != 3 {
		t.Fatal("failed uploads removed existing photos or leaked new files")
	}
	request("PUT", photoPath(c.ID)+"/cover", nil, "", 204)
	request("DELETE", photoPath(c.ID), nil, "", 204)
	assertNoCover()
	request("DELETE", photoPath(b.ID), nil, "", 204)
	request("DELETE", photoPath(a.ID), nil, "", 204)
	if len(list()) != 0 || len(store.objects) != 0 {
		t.Fatal("last photo removal left data behind")
	}
	var emptyLocation domain.Location
	must(json.Unmarshal(request("GET", path, nil, "", 200).Body.Bytes(), &emptyLocation))
	if emptyLocation.PhotoURL != nil {
		t.Fatal("empty location retained cover")
	}
	// Concurrent uploads must not select a cover.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			photo := &domain.LocationPhoto{LocationID: loc.ID, FileKey: fmt.Sprintf("concurrent-%d", index), FileName: "Concurrent"}
			errs <- repo.AddPhoto(ctx, org.ID, photo)
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(err)
	}
	concurrent, err := repo.ListPhotos(ctx, org.ID, loc.ID)
	must(err)
	coverCount := 0
	for _, photo := range concurrent {
		if photo.IsCover {
			coverCount++
		}
	}
	if len(concurrent) != 8 || coverCount != 0 {
		t.Fatalf("concurrent cover invariant violated: %d photos, %d covers", len(concurrent), coverCount)
	}
	upload(loc.ID.String(), imageData, 201)
	request("DELETE", path, nil, "", 204)
	if len(store.objects) != 0 {
		t.Fatal("deleting location leaked gallery files")
	}
	remaining, err := repo.ListPhotos(ctx, org.ID, loc.ID)
	must(err)
	if len(remaining) != 0 {
		t.Fatal("location deletion retained gallery rows")
	}
	upload(loc.ID.String(), imageData, 404)
	lost, err := fixtures.CreateLocation(ctx, org.ID, "Concurrent deletion", nil)
	must(err)
	store.onUpload = func() {
		_, err := db.Pool.Exec(ctx, "UPDATE locations SET deleted_at=now() WHERE id=$1", lost.ID)
		must(err)
	}
	upload(lost.ID.String(), imageData, 404)
	store.onUpload = nil
	if len(store.objects) != 0 {
		t.Fatal("failed persistence leaked uploaded file")
	}
	newLoc, err := fixtures.CreateLocation(ctx, org.ID, "Storage unavailable", nil)
	must(err)
	direct := chi.NewRouter()
	direct.Post("/{id}/photos", New(nil, h.repos, nil, org.ID).UploadLocationPhoto)
	response := httptest.NewRecorder()
	direct.ServeHTTP(response, httptest.NewRequest("POST", "/"+newLoc.ID.String()+"/photos", nil))
	if response.Code != 503 {
		t.Fatal("missing storage should return 503")
	}
	features, err := h.repos.Organizations.GetFeatures(ctx, org.ID)
	must(err)
	features.Locations = false
	must(h.repos.Organizations.UpdateFeatures(ctx, org.ID, features))
	upload(newLoc.ID.String(), imageData, 404)
	// Single-photo galleries still support a lossless down/up round trip.
	must(migration("down"))
	must(migration("up"))
}
