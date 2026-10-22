package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Mazik-kun/mini-core/services/document/internal/core/domain"
)

const testMaxSize = 1024

func TestInitUpload_Happy(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	uc := NewInitUploadUseCase(repo, st, 15*time.Minute)

	out, err := uc.InitUpload(clientCtx(cust), InitUploadInput{CustomerID: cust, Type: domain.TypePassport})
	if err != nil {
		t.Fatal(err)
	}
	if out.UploadURL == "" {
		t.Error("empty upload url")
	}
	doc, err := repo.GetByID(context.Background(), out.DocumentID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Status != domain.StatusPending || doc.CustomerID != cust ||
		doc.ObjectKey != domain.ObjectKey(cust, domain.TypePassport, out.DocumentID) {
		t.Errorf("unexpected doc: %+v", doc)
	}
	if out.DocumentID.Version() != 7 {
		t.Errorf("document id must be UUID v7, got v%d", out.DocumentID.Version())
	}
}

func TestInitUpload_InvalidType(t *testing.T) {
	cust := uuid.New()
	uc := NewInitUploadUseCase(newFakeRepo(), newFakeStorage(), time.Minute)
	_, err := uc.InitUpload(clientCtx(cust), InitUploadInput{CustomerID: cust, Type: "VISA"})
	if !errors.Is(err, domain.ErrInvalidDocumentType) {
		t.Errorf("got %v", err)
	}
}

func TestInitUpload_IDOR(t *testing.T) {
	uc := NewInitUploadUseCase(newFakeRepo(), newFakeStorage(), time.Minute)
	_, err := uc.InitUpload(clientCtx(uuid.New()), InitUploadInput{CustomerID: uuid.New(), Type: domain.TypeSelfie})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("got %v", err)
	}
}

func TestInitUpload_OfficerCanActForAnyCustomer(t *testing.T) {
	uc := NewInitUploadUseCase(newFakeRepo(), newFakeStorage(), time.Minute)
	if _, err := uc.InitUpload(officerCtx(), InitUploadInput{CustomerID: uuid.New(), Type: domain.TypeSelfie}); err != nil {
		t.Errorf("officer must have access: %v", err)
	}
}

func TestConfirmUpload_Happy(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	doc := pendingDoc(cust, domain.TypePassport)
	repo.seed(doc)
	body := pngBytes(100)
	st.objects[doc.ObjectKey] = body

	uc := NewConfirmUploadUseCase(repo, st, testMaxSize)
	got, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusConfirmed || got.ContentType != "image/png" ||
		got.SizeBytes != int64(len(body)) || len(got.SHA256) != 64 || got.ConfirmedAt == nil {
		t.Errorf("unexpected doc: %+v", got)
	}
}

func TestConfirmUpload_TooLarge(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	doc := pendingDoc(cust, domain.TypePassport)
	repo.seed(doc)
	st.objects[doc.ObjectKey] = pngBytes(testMaxSize + 1)

	uc := NewConfirmUploadUseCase(repo, st, testMaxSize)
	_, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if !errors.Is(err, domain.ErrFileTooLarge) {
		t.Fatalf("got %v", err)
	}
	assertRejected(t, repo, st, doc)
}

func TestConfirmUpload_WrongMagicBytes(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	doc := pendingDoc(cust, domain.TypeSelfie)
	repo.seed(doc)
	st.objects[doc.ObjectKey] = []byte("just some text, not an image")

	uc := NewConfirmUploadUseCase(repo, st, testMaxSize)
	_, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if !errors.Is(err, domain.ErrUnsupportedContentType) {
		t.Fatalf("got %v", err)
	}
	assertRejected(t, repo, st, doc)
}

func assertRejected(t *testing.T, repo *fakeRepo, st *fakeStorage, doc domain.Document) {
	t.Helper()
	got, _ := repo.GetByID(context.Background(), doc.ID)
	if got.Status != domain.StatusRejected {
		t.Errorf("status = %s, want REJECTED", got.Status)
	}
	if _, ok := st.objects[doc.ObjectKey]; ok || len(st.removed) != 1 {
		t.Error("rejected object must be removed from storage")
	}
}

func TestConfirmUpload_AlreadyConfirmed_Idempotent(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	doc := pendingDoc(cust, domain.TypePassport)
	doc.Status, doc.ContentType, doc.SizeBytes, doc.SHA256 = domain.StatusConfirmed, "image/png", 10, "abc"
	repo.seed(doc)

	uc := NewConfirmUploadUseCase(repo, st, testMaxSize)
	got, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != doc.ID || got.SHA256 != "abc" {
		t.Errorf("must return the same document, got %+v", got)
	}
	if st.statCalls != 0 || len(st.removed) != 0 {
		t.Error("storage must not be touched for already confirmed document")
	}
}

func TestConfirmUpload_NotUploaded(t *testing.T) {
	cust := uuid.New()
	repo := newFakeRepo()
	doc := pendingDoc(cust, domain.TypePassport)
	repo.seed(doc)

	uc := NewConfirmUploadUseCase(repo, newFakeStorage(), testMaxSize)
	_, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if !errors.Is(err, domain.ErrObjectNotUploaded) {
		t.Errorf("got %v", err)
	}
	if got, _ := repo.GetByID(context.Background(), doc.ID); got.Status != domain.StatusPending {
		t.Error("document must stay PENDING")
	}
}

func TestConfirmUpload_RejectedCannotBeConfirmed(t *testing.T) {
	cust := uuid.New()
	repo := newFakeRepo()
	doc := pendingDoc(cust, domain.TypePassport)
	doc.Status = domain.StatusRejected
	repo.seed(doc)

	uc := NewConfirmUploadUseCase(repo, newFakeStorage(), testMaxSize)
	_, err := uc.ConfirmUpload(clientCtx(cust), ConfirmUploadInput{DocumentID: doc.ID})
	if !errors.Is(err, domain.ErrInvalidStatusTransition) {
		t.Errorf("got %v", err)
	}
}

func TestConfirmUpload_IDOR(t *testing.T) {
	owner := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	doc := pendingDoc(owner, domain.TypePassport)
	repo.seed(doc)
	st.objects[doc.ObjectKey] = pngBytes(10)

	uc := NewConfirmUploadUseCase(repo, st, testMaxSize)
	_, err := uc.ConfirmUpload(clientCtx(uuid.New()), ConfirmUploadInput{DocumentID: doc.ID})
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("got %v", err)
	}
}

func TestGetDownloadURL(t *testing.T) {
	cust := uuid.New()
	repo, st := newFakeRepo(), newFakeStorage()
	uc := NewGetDownloadURLUseCase(repo, st, 5*time.Minute)

	pending := pendingDoc(cust, domain.TypePassport)
	repo.seed(pending)
	if _, err := uc.GetDownloadURL(clientCtx(cust), GetDownloadURLInput{DocumentID: pending.ID}); !errors.Is(err, domain.ErrDocumentNotConfirmed) {
		t.Errorf("pending: got %v", err)
	}

	confirmed := pendingDoc(cust, domain.TypeSelfie)
	confirmed.Status = domain.StatusConfirmed
	repo.seed(confirmed)
	out, err := uc.GetDownloadURL(clientCtx(cust), GetDownloadURLInput{DocumentID: confirmed.ID})
	if err != nil || out.URL == "" || !out.ExpiresAt.After(time.Now()) {
		t.Errorf("confirmed: %+v %v", out, err)
	}

	if _, err := uc.GetDownloadURL(clientCtx(uuid.New()), GetDownloadURLInput{DocumentID: confirmed.ID}); !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("idor: got %v", err)
	}
	if _, err := uc.GetDownloadURL(officerCtx(), GetDownloadURLInput{DocumentID: confirmed.ID}); err != nil {
		t.Errorf("officer: %v", err)
	}
}

func TestHasRequiredDocuments(t *testing.T) {
	cust := uuid.New()
	repo := newFakeRepo()
	required := []domain.Type{domain.TypePassport, domain.TypeSelfie}
	uc := NewHasRequiredDocumentsUseCase(repo, required)

	passport := pendingDoc(cust, domain.TypePassport)
	passport.Status = domain.StatusConfirmed
	repo.seed(passport)
	pendingSelfie := pendingDoc(cust, domain.TypeSelfie) // PENDING не считается
	repo.seed(pendingSelfie)

	out, err := uc.HasRequiredDocuments(clientCtx(cust), HasRequiredDocumentsInput{CustomerID: cust, Required: required})
	if err != nil {
		t.Fatal(err)
	}
	if out.HasAll || len(out.Missing) != 1 || out.Missing[0] != domain.TypeSelfie {
		t.Errorf("missing one: %+v", out)
	}

	pendingSelfie.Status = domain.StatusConfirmed
	repo.seed(pendingSelfie)
	out, err = uc.HasRequiredDocuments(clientCtx(cust), HasRequiredDocumentsInput{CustomerID: cust})
	if err != nil || !out.HasAll || len(out.Missing) != 0 {
		t.Errorf("all present (default required): %+v %v", out, err)
	}
}

func TestHasRequiredDocuments_IDORAndInvalidType(t *testing.T) {
	uc := NewHasRequiredDocumentsUseCase(newFakeRepo(), []domain.Type{domain.TypePassport})
	if _, err := uc.HasRequiredDocuments(clientCtx(uuid.New()), HasRequiredDocumentsInput{CustomerID: uuid.New()}); !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("idor: got %v", err)
	}
	cust := uuid.New()
	if _, err := uc.HasRequiredDocuments(clientCtx(cust), HasRequiredDocumentsInput{CustomerID: cust, Required: []domain.Type{"X"}}); !errors.Is(err, domain.ErrInvalidDocumentType) {
		t.Errorf("invalid type: got %v", err)
	}
}

func TestListDocuments_PaginationAndIDOR(t *testing.T) {
	cust := uuid.New()
	repo := newFakeRepo()
	for i := 0; i < 5; i++ {
		repo.seed(pendingDoc(cust, domain.TypePassport))
	}
	uc := NewListDocumentsUseCase(repo, 2, 10)

	seen := map[uuid.UUID]bool{}
	token := ""
	for pages := 0; pages < 10; pages++ {
		out, err := uc.ListDocuments(clientCtx(cust), ListDocumentsInput{CustomerID: cust, PageSize: 2, PageToken: token})
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range out.Documents {
			if seen[d.ID] {
				t.Fatalf("duplicate %s across pages", d.ID)
			}
			seen[d.ID] = true
		}
		if out.NextPageToken == "" {
			break
		}
		token = out.NextPageToken
	}
	if len(seen) != 5 {
		t.Errorf("got %d docs, want 5", len(seen))
	}

	if _, err := uc.ListDocuments(clientCtx(cust), ListDocumentsInput{CustomerID: cust, PageToken: "!!!"}); !errors.Is(err, domain.ErrInvalidPageToken) {
		t.Errorf("bad token: got %v", err)
	}
	if _, err := uc.ListDocuments(clientCtx(uuid.New()), ListDocumentsInput{CustomerID: cust}); !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("idor: got %v", err)
	}
}
