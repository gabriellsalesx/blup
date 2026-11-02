package blupcloud

import (
	"context"
	"net/http"
)

// The calendar's reminders also sent by e-mail (Max and Ultra). Only when
// each one is due goes up, under an opaque id: the event stays on the
// computer, and the e-mail only says there is a reminder.

// Reminder is one reminder: an opaque id (hex) and when (Unix ms).
type Reminder struct {
	ID string `json:"id"`
	At int64  `json:"at"`
}

// ReminderUse is how many reminder e-mails the plan has this month (0:
// none) and how many went.
type ReminderUse struct {
	Limit int `json:"limit"`
	Used  int `json:"used"`
}

// ReminderEmails tells the month's use.
func (c *Client) ReminderEmails(ctx context.Context) (ReminderUse, error) {
	var u ReminderUse
	return u, c.do(ctx, http.MethodGet, "/reminders", nil, "", &u)
}

// SetReminders replaces the reminders not sent yet (an empty list cancels
// them all).
func (c *Client) SetReminders(ctx context.Context, list []Reminder) (ReminderUse, error) {
	if list == nil {
		list = []Reminder{}
	}
	var u ReminderUse
	return u, c.do(ctx, http.MethodPut, "/reminders", map[string]any{"reminders": list}, "", &u)
}
