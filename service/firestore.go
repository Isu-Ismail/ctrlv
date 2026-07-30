package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceAccountFile struct {
	ProjectID string `json:"project_id"`
}

type FirestoreService struct {
	client    *firestore.Client
	projectID string
}

// NewFirestoreService initializes a Firestore client dynamically reading project_id from serviceAccountKey.json
func NewFirestoreService(credFile string) (*FirestoreService, error) {
	ctx := context.Background()

	data, err := os.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file %s: %w", credFile, err)
	}

	var sa serviceAccountFile
	if err := json.Unmarshal(data, &sa); err != nil || sa.ProjectID == "" {
		return nil, fmt.Errorf("failed to parse valid project_id from %s: %w", credFile, err)
	}

	client, err := firestore.NewClient(ctx, sa.ProjectID, option.WithCredentialsFile(credFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client for project %s: %w", sa.ProjectID, err)
	}

	log.Printf("[Firestore] Service connected successfully for project: %s", sa.ProjectID)
	return &FirestoreService{
		client:    client,
		projectID: sa.ProjectID,
	}, nil
}

// VerifyConnection tests network connectivity and credentials against Firestore with a 5-second timeout
func (fs *FirestoreService) VerifyConnection(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	docRef := fs.client.Collection("room").Doc(roomID)
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("network or auth verification failed: %w", err)
	}
	return nil
}

// UploadScreenshot stores the base64 screenshot data in room/<roomID> document
func (fs *FirestoreService) UploadScreenshot(roomID string, base64Image string) error {
	ctx := context.Background()
	docRef := fs.client.Collection("room").Doc(roomID)

	_, err := docRef.Set(ctx, map[string]interface{}{
		"image_path": base64Image,
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to upload screenshot to room %s: %w", roomID, err)
	}

	log.Printf("[Firestore] Screenshot uploaded successfully for room: %s", roomID)
	return nil
}

// UploadQuestionText stores clipboard question text in room/<roomID> document under pc_sent_text
func (fs *FirestoreService) UploadQuestionText(roomID string, text string) error {
	ctx := context.Background()
	docRef := fs.client.Collection("room").Doc(roomID)

	_, err := docRef.Set(ctx, map[string]interface{}{
		"pc_sent_text": text,
		"fetched":      false,
	}, firestore.MergeAll)

	if err != nil {
		return fmt.Errorf("failed to upload question text to room %s: %w", roomID, err)
	}

	log.Printf("[Firestore] Question text uploaded successfully for room: %s", roomID)
	return nil
}

// FetchTextAndMarkSeen fetches uploaded_text from room/<roomID> and sets fetched: true
func (fs *FirestoreService) FetchTextAndMarkSeen(roomID string) (string, error) {
	ctx := context.Background()
	docRef := fs.client.Collection("room").Doc(roomID)

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch room doc %s: %w", roomID, err)
	}

	data := docSnap.Data()
	uploadedText, _ := data["uploaded_text"].(string)

	// Mark fetched = true
	_, err = docRef.Set(ctx, map[string]interface{}{
		"fetched": true,
	}, firestore.MergeAll)

	if err != nil {
		log.Printf("[Firestore] Warning: failed to mark fetched: true for room %s: %v", roomID, err)
	} else {
		log.Printf("[Firestore] Marked text as fetched: true for room: %s", roomID)
	}

	return uploadedText, nil
}

// ListenRoomUpdates listens in real-time to room/<roomID> document changes.
// When uploaded_text changes and fetched is false, it automatically triggers onNewText callback and sets fetched: true.
func (fs *FirestoreService) ListenRoomUpdates(ctx context.Context, roomID string, onNewText func(text string)) {
	docRef := fs.client.Collection("room").Doc(roomID)
	iter := docRef.Snapshots(ctx)
	defer iter.Stop()

	var lastText string

	for {
		snap, err := iter.Next()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Firestore Listener Warning] %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if !snap.Exists() {
			continue
		}

		data := snap.Data()
		text, _ := data["uploaded_text"].(string)
		fetched, _ := data["fetched"].(bool)

		if text != "" && text != lastText && !fetched {
			lastText = text

			// Mark fetched: true so room document is flagged as seen
			_, _ = docRef.Set(ctx, map[string]interface{}{
				"fetched": true,
			}, firestore.MergeAll)

			// Trigger callback for auto clipboard write and overlay update
			if onNewText != nil {
				onNewText(text)
			}
		}
	}
}

// Close closes the Firestore client
func (fs *FirestoreService) Close() {
	if fs.client != nil {
		fs.client.Close()
	}
}
