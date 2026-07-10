package service_test

import (
	"testing"

	actSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
)

func TestActivityLog_Direction(t *testing.T) {
	inbound := actSvc.ActivityLog{Direction: actSvc.DirectionInbound}
	outbound := actSvc.ActivityLog{Direction: actSvc.DirectionOutbound}

	if inbound.Direction != actSvc.DirectionInbound {
		t.Errorf("want inbound, got %q", inbound.Direction)
	}
	if outbound.Direction != actSvc.DirectionOutbound {
		t.Errorf("want outbound, got %q", outbound.Direction)
	}
}

func TestActivityLog_Constants(t *testing.T) {
	// Ensure all action constants are non-empty strings (catch accidental blank iota)
	actions := []string{
		actSvc.ActionUserLoginCredential,
		actSvc.ActionUserLoginGoogle,
		actSvc.ActionUserLoginApple,
		actSvc.ActionUserLoginGitHub,
		actSvc.ActionUserLoginPasskey,
		actSvc.ActionUserLoginBiometric,
		actSvc.ActionUserLogout,
		actSvc.ActionUserRegister,
		actSvc.ActionOutEmailSent,
		actSvc.ActionOutSMSSent,
		actSvc.ActionOutWhatsAppSent,
		actSvc.ActionOutPushSent,
		actSvc.ActionOutAPIResponse,
		actSvc.ActionPasskeyRegistered,
		actSvc.ActionMobileVerified,
	}
	for _, a := range actions {
		if a == "" {
			t.Errorf("action constant is empty: %q", a)
		}
	}
}

func TestActivityLog_Pair_CorrelationID(t *testing.T) {
	correlationID := "test-req-abc-123"
	inbound := actSvc.ActivityLog{CorrelationID: correlationID, Direction: actSvc.DirectionInbound}
	outbound := actSvc.ActivityLog{CorrelationID: correlationID, Direction: actSvc.DirectionOutbound}

	if inbound.CorrelationID != outbound.CorrelationID {
		t.Error("inbound and outbound must share the same correlationId for pairing")
	}
}
