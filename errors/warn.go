package errors

// WarnCode is a soft notice code (7xxx) returned with HTTP 200 for invite flows.
type WarnCode int

const WarnCodeOK WarnCode = 0

var WarnOK = WarnEntry{Code: WarnCodeOK}

// WarnEntry pairs a warn code with its default user-facing message.
type WarnEntry struct {
	Code    WarnCode
	Message string
}

// Invite warn codes and messages (single source of truth).
var (
	WarnInviteExistingUser = WarnEntry{
		Code:    7432,
		Message: "Existing users cannot use an invite code.",
	}
	WarnInviteLimitReached = WarnEntry{
		Code:    7433,
		Message: "This invite code has reached the invite limit.",
	}
	WarnInviteInvalidCode = WarnEntry{
		Code:    7434,
		Message: "Invalid invite code.",
	}
	WarnInviteSelfInvite = WarnEntry{
		Code:    7435,
		Message: "You cannot use your own invite code.",
	}

	warnCatalog = []WarnEntry{
		WarnInviteExistingUser,
		WarnInviteLimitReached,
		WarnInviteInvalidCode,
		WarnInviteSelfInvite,
	}

	// WInvite* are shorthand aliases for call sites that only need the code.
	WInviteExistingUser = WarnInviteExistingUser.Code
	WInviteLimitReached = WarnInviteLimitReached.Code
	WInviteInvalidCode  = WarnInviteInvalidCode.Code
	WInviteSelfInvite   = WarnInviteSelfInvite.Code
)

// WarnMessage returns the default user-facing message for a warn code.
func WarnMessage(code WarnCode) string {
	for _, entry := range warnCatalog {
		if entry.Code == code {
			return entry.Message
		}
	}
	return ""
}

// WarnEntryForCode returns the catalog entry for code, or false when unknown.
func WarnEntryForCode(code WarnCode) (WarnEntry, bool) {
	for _, entry := range warnCatalog {
		if entry.Code == code {
			return entry, true
		}
	}
	return WarnEntry{}, false
}
