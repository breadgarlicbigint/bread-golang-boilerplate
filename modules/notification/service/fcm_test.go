package service

import "testing"

// The Android/APNs message configuration.
//
// These are pure builders, tested directly, because the layer above them cannot
// be: sending needs a real Firebase client and a real token. What IS worth
// pinning is the shape of what we ask for, since every field here fails SILENTLY
// when it is wrong — a missing channel is delivered at default importance with
// no heads-up alert, and a missing AndroidConfig at normal priority. Nothing
// errors either way, so the only report is "push doesn't work" from a path that
// is in fact working.

func TestAndroidConfig_AlertCarriesTheHighImportanceChannel(t *testing.T) {
	c := androidConfig(PushPayload{Title: "Alert", Body: "2 left", ImageURL: "https://x/y.png"}, "myapp_alerts")

	if c.Priority != "high" {
		t.Errorf("priority = %q, want high — a normal-priority push may be deferred by Doze", c.Priority)
	}
	if c.Notification == nil {
		t.Fatal("an alert must carry an Android notification block")
	}
	if c.Notification.ChannelID != "myapp_alerts" {
		t.Errorf("channelID = %q, want the configured channel", c.Notification.ChannelID)
	}
	if c.Notification.Title != "Alert" || c.Notification.Body != "2 left" {
		t.Errorf("title/body not carried: %+v", c.Notification)
	}
	if c.Notification.ImageURL != "https://x/y.png" {
		t.Errorf("imageURL = %q, want it carried", c.Notification.ImageURL)
	}
}

func TestAndroidConfig_SilentCarriesNoNotificationBlock(t *testing.T) {
	c := androidConfig(PushPayload{Silent: true, Title: "should not show"}, "myapp_alerts")

	// A notification block present is what makes the OS draw an alert, whatever
	// the priority says — so a "silent" push carrying one is a visible push with
	// a confusing name.
	if c.Notification != nil {
		t.Error("a silent message must carry no Android notification block")
	}
	if c.Priority != "high" {
		t.Errorf("priority = %q, want high", c.Priority)
	}
}

func TestAPNSConfig_PriorityFollowsVisibility(t *testing.T) {
	alert := apnsConfig(PushPayload{Title: "t", Body: "b", Badge: 3})
	if got := alert.Headers["apns-priority"]; got != "10" {
		t.Errorf("alert apns-priority = %q, want 10", got)
	}
	if alert.Payload.Aps.Alert == nil || alert.Payload.Aps.Alert.Title != "t" {
		t.Error("alert payload must carry the alert block")
	}
	if alert.Payload.Aps.Badge == nil || *alert.Payload.Aps.Badge != 3 {
		t.Error("badge must be carried by value, not by a pointer into the loop variable")
	}

	// APNs REJECTS a background push sent at priority 10 — Apple's rule, not a
	// preference, so it is worth a test rather than a comment.
	silent := apnsConfig(PushPayload{Silent: true})
	if got := silent.Headers["apns-priority"]; got != "5" {
		t.Errorf("silent apns-priority = %q, want 5", got)
	}
	if silent.Payload.Aps.Alert != nil {
		t.Error("a silent message must carry no aps alert")
	}
	if !silent.Payload.Aps.ContentAvailable {
		t.Error("a silent message must set content-available, or iOS will not wake the app")
	}
}

func TestNotification_AbsentOnlyForSilent(t *testing.T) {
	if n := notification(PushPayload{Title: "t", Body: "b"}); n == nil {
		t.Fatal("a visible message must carry a notification block")
	}
	if n := notification(PushPayload{Silent: true, Title: "t"}); n != nil {
		t.Error("a silent message must carry none — SendMulticast used to send one regardless, " +
			"which drew an empty notification for every silent push")
	}
}
