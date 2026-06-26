package data

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent/enttest"
	announcementbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/announcement"

	_ "github.com/mattn/go-sqlite3"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestAdminAnnouncementSavePinnedUnpinsOtherAnnouncements(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:admin_announcement_save_pin?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewAdminAnnouncementRepo(&Data{db: client}, log.NewStdLogger(io.Discard))
	content := "content"

	first, err := repo.Save(ctx, &announcementbiz.Announcement{
		Title:   "first",
		Content: &content,
		Show:    boolPtr(true),
		Pinned:  boolPtr(true),
		Popup:   boolPtr(true),
	})
	if err != nil {
		t.Fatalf("save first announcement: %v", err)
	}

	second, err := repo.Save(ctx, &announcementbiz.Announcement{
		Title:   "second",
		Content: &content,
		Show:    boolPtr(true),
		Pinned:  boolPtr(true),
		Popup:   boolPtr(true),
	})
	if err != nil {
		t.Fatalf("save second announcement: %v", err)
	}

	firstRow := client.ProxyAnnouncement.GetX(ctx, first.ID)
	secondRow := client.ProxyAnnouncement.GetX(ctx, second.ID)
	if firstRow.Pinned {
		t.Fatal("first announcement stayed pinned after another pinned announcement was created")
	}
	if !secondRow.Pinned {
		t.Fatal("second announcement should stay pinned")
	}
}

func TestAdminAnnouncementUpdatePinnedUnpinsOtherAnnouncements(t *testing.T) {
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:admin_announcement_update_pin?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewAdminAnnouncementRepo(&Data{db: client}, log.NewStdLogger(io.Discard))
	content := "content"

	first, err := repo.Save(ctx, &announcementbiz.Announcement{
		Title:   "first",
		Content: &content,
		Show:    boolPtr(true),
		Pinned:  boolPtr(true),
	})
	if err != nil {
		t.Fatalf("save first announcement: %v", err)
	}

	second, err := repo.Save(ctx, &announcementbiz.Announcement{
		Title:   "second",
		Content: &content,
		Show:    boolPtr(true),
		Pinned:  boolPtr(false),
	})
	if err != nil {
		t.Fatalf("save second announcement: %v", err)
	}

	_, err = repo.Update(ctx, &announcementbiz.Announcement{
		ID:      second.ID,
		Title:   "second",
		Content: &content,
		Show:    boolPtr(true),
		Pinned:  boolPtr(true),
	})
	if err != nil {
		t.Fatalf("update second announcement: %v", err)
	}

	firstRow := client.ProxyAnnouncement.GetX(ctx, first.ID)
	secondRow := client.ProxyAnnouncement.GetX(ctx, second.ID)
	if firstRow.Pinned {
		t.Fatal("first announcement stayed pinned after another announcement was pinned")
	}
	if !secondRow.Pinned {
		t.Fatal("second announcement should stay pinned")
	}
}
