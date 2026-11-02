package app

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

// The calendar's reminders by e-mail through Blup Cloud (Max and Ultra).
// Signed out, both answer blupcloud.ErrSignedOut.

// ReminderEmails is the month's use of reminder e-mails.
func ReminderEmails(ctx context.Context) (blupcloud.ReminderUse, error) {
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.ReminderUse{}, err
	}
	return c.ReminderEmails(ctx)
}

// PushReminders replaces the reminders waiting in the cloud.
func PushReminders(ctx context.Context, list []blupcloud.Reminder) (blupcloud.ReminderUse, error) {
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.ReminderUse{}, err
	}
	return c.SetReminders(ctx, list)
}
