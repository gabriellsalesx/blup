package blupcloud

import (
	"context"
	"fmt"
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
	// EmailMissing: the account has no e-mail (passkey only), so no
	// reminder e-mail can go.
	EmailMissing bool `json:"emailMissing"`
}

// ReminderEmails tells the month's use.
func (c *Client) ReminderEmails(ctx context.Context) (ReminderUse, error) {
	return reminderUse(func(r any) error { return c.do(ctx, http.MethodGet, "/reminders", nil, "", r) })
}

// SetReminders replaces the reminders not sent yet (an empty list cancels
// them all).
func (c *Client) SetReminders(ctx context.Context, list []Reminder) (ReminderUse, error) {
	if list == nil {
		list = []Reminder{}
	}
	return reminderUse(func(r any) error {
		return c.do(ctx, http.MethodPut, "/reminders", map[string]any{"reminders": list}, "", r)
	})
}

// reminderUse reads the month's use: both numbers, never below zero.
func reminderUse(call func(out any) error) (ReminderUse, error) {
	var r struct {
		Limit *int `json:"limit"`
		Used  *int `json:"used"`
		// Older clouds don't send it: an e-mail is then assumed.
		EmailMissing *bool `json:"emailMissing"`
	}
	if err := call(&r); err != nil {
		return ReminderUse{}, err
	}
	if r.Limit == nil || r.Used == nil || *r.Limit < 0 || *r.Used < 0 {
		return ReminderUse{}, fmt.Errorf("%w: reminders", ErrBadAnswer)
	}
	u := ReminderUse{Limit: *r.Limit, Used: *r.Used}
	if r.EmailMissing != nil {
		u.EmailMissing = *r.EmailMissing
	}
	return u, nil
}
