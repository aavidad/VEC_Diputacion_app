package handler

import (
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/candidate/adapters/repository"
	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
	"vec-diputacion-granada/internal/candidate/usecases"
	"vec-diputacion-granada/internal/shared/i18n"
)

func TestHTTPHandlerCandidateDocumentsMethodAuthHappyPathAndErrors(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano})
	body := `{"id":"doc-1","solicitud_id":"sol-1","procedure_id":"proc-1","purpose":"Alegacion","csv":"CSV-DOC-1","digest_sha256":"abc123","storage_object_ref":"obj-1","tsa_stamp_ref":"tsa-1","document_type":"alegacion","title":"Alegacion","format":"application/pdf","signature_ref":"sig-1","registered_at":"2026-06-19T13:00:00Z"}`

	created := performJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/documents", body)
	assertStatus(t, created, http.StatusCreated)
	assertMessage(t, created, "documento")
	var document administrativeDocumentView
	decodeData(t, created, &document)
	if document.CandidateID != "cand-1" || document.AuditSequence != 1 || document.AVStatus != domain.AVStatusPending {
		t.Fatalf("document = %+v", document)
	}

	listed := performJSON(t, handler, http.MethodGet, "/api/candidates/cand-1/documents", "")
	assertStatus(t, listed, http.StatusOK)
	var documents []administrativeDocumentView
	decodeData(t, listed, &documents)
	if len(documents) != 1 || documents[0].ID != "doc-1" {
		t.Fatalf("documents = %+v", documents)
	}

	method := performJSON(t, handler, http.MethodPut, "/api/candidates/cand-1/documents", body)
	assertStatus(t, method, http.StatusMethodNotAllowed)
	bad := performJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/documents", `{"id":"doc-2"}`)
	assertStatus(t, bad, http.StatusBadRequest)
}

func TestHTTPHandlerCandidateDocumentsRequireCandidateRole(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno})

	response := performStaffJSON(t, handler, http.MethodGet, "/api/candidates/cand-1/documents", "")

	assertStatus(t, response, http.StatusForbidden)
}

func TestHTTPHandlerCandidateClaimsMethodAuthHappyPathAndErrors(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano})
	body := `{"id":"claim-1","solicitud_id":"sol-1","text":"Alego contra baremo provisional","receipt_csv":"CSV-CLAIM-1","presented_at":"2026-06-19T13:05:00Z"}`

	created := performJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/claims", body)
	assertStatus(t, created, http.StatusCreated)
	assertMessage(t, created, "alegacion")
	var claim administrativeClaimView
	decodeData(t, created, &claim)
	if claim.CandidateID != "cand-1" || claim.State != domain.ClaimStatePresentada ||
		claim.AuditSequence != 1 || claim.ReceiptI18nKey != "module.bolsa.claim.presented" {
		t.Fatalf("claim = %+v", claim)
	}

	listed := performJSON(t, handler, http.MethodGet, "/api/candidates/cand-1/claims?solicitud_id=sol-1", "")
	assertStatus(t, listed, http.StatusOK)
	var claims []administrativeClaimView
	decodeData(t, listed, &claims)
	if len(claims) != 1 || claims[0].ID != "claim-1" {
		t.Fatalf("claims = %+v", claims)
	}

	assertStatus(t, performJSON(t, handler, http.MethodPut, "/api/candidates/cand-1/claims", body), http.StatusMethodNotAllowed)
	assertStatus(t, performJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/claims", `{"id":"claim-2"}`), http.StatusBadRequest)
}

func TestHTTPHandlerStaffNotificationsAndAuditMethodAuthHappyPathAndErrors(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno})
	body := `{"id":"note-1","solicitud_id":"sol-1","type":"aviso","subject":"Subsanacion","body":"Revise su solicitud","created_at":"2026-06-19T13:10:00Z"}`

	created := performStaffJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/notifications", body)
	assertStatus(t, created, http.StatusCreated)
	assertMessage(t, created, "notificacion")
	var notification administrativeNotificationView
	decodeData(t, created, &notification)
	if notification.CandidateID != "cand-1" || notification.State != domain.NotificationStateCreada ||
		notification.AuditSequence != 1 || notification.ReceiptI18nKey != "module.bolsa.notification.created" {
		t.Fatalf("notification = %+v", notification)
	}

	listed := performStaffJSON(t, handler, http.MethodGet, "/api/candidates/cand-1/notifications", "")
	assertStatus(t, listed, http.StatusOK)
	var notifications []administrativeNotificationView
	decodeData(t, listed, &notifications)
	if len(notifications) != 1 || notifications[0].ID != "note-1" {
		t.Fatalf("notifications = %+v", notifications)
	}

	audited := performStaffJSON(t, handler, http.MethodGet, "/api/candidates/cand-1/audit", "")
	assertStatus(t, audited, http.StatusOK)
	var audit []administrativeAuditView
	decodeData(t, audited, &audit)
	if len(audit) != 1 || audit[0].Action != "candidate.notification.created" {
		t.Fatalf("audit = %+v", audit)
	}

	assertStatus(t, performStaffJSON(t, handler, http.MethodPut, "/api/candidates/cand-1/notifications", body), http.StatusMethodNotAllowed)
	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/notifications", `{"id":"note-2"}`), http.StatusBadRequest)
	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/candidates/cand-1/audit", ""), http.StatusMethodNotAllowed)
}

func TestHTTPHandlerGlobalNotificationsSendReadAndAuditHappyPath(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno})
	body := `{"candidate_id":"cand-1","id":"note-1","solicitud_id":"sol-1","type":"aviso","subject":"Subsanacion","body":"Revise su solicitud","created_at":"2026-06-19T13:10:00Z"}`

	created := performStaffJSON(t, handler, http.MethodPost, "/api/notifications", body)
	assertStatus(t, created, http.StatusCreated)
	var createdNotification administrativeNotificationView
	decodeData(t, created, &createdNotification)
	if createdNotification.CandidateID != "cand-1" || createdNotification.State != domain.NotificationStateCreada {
		t.Fatalf("created notification = %+v", createdNotification)
	}

	listed := performStaffJSON(t, handler, http.MethodGet, "/api/notifications?candidate_id=cand-1", "")
	assertStatus(t, listed, http.StatusOK)
	var notifications []administrativeNotificationView
	decodeData(t, listed, &notifications)
	if len(notifications) != 1 || notifications[0].ID != "note-1" {
		t.Fatalf("notifications = %+v", notifications)
	}

	sent := performStaffJSON(t, handler, http.MethodPost, "/api/notifications/note-1/send", `{"csv":"CSV-NOT-1","recipient_id":"cand-1","channel":"vec","issued_at":"2026-06-19T13:11:00Z"}`)
	assertStatus(t, sent, http.StatusOK)
	var sentNotification administrativeNotificationView
	decodeData(t, sent, &sentNotification)
	if sentNotification.State != domain.NotificationStateEnviada || sentNotification.AuditSequence != 2 {
		t.Fatalf("sent notification = %+v", sentNotification)
	}

	read := performStaffJSON(t, handler, http.MethodPost, "/api/notifications/note-1/read", `{"csv":"CSV-NOT-2","recipient_id":"cand-1","channel":"vec","issued_at":"2026-06-19T13:12:00Z"}`)
	assertStatus(t, read, http.StatusOK)
	var readNotification administrativeNotificationView
	decodeData(t, read, &readNotification)
	if readNotification.State != domain.NotificationStateLeida || readNotification.AuditSequence != 3 {
		t.Fatalf("read notification = %+v", readNotification)
	}

	audited := performStaffJSON(t, handler, http.MethodGet, "/api/audit?candidate_id=cand-1", "")
	assertStatus(t, audited, http.StatusOK)
	var audit []administrativeAuditView
	decodeData(t, audited, &audit)
	if len(audit) != 3 || audit[1].Action != "candidate.notification.sent" || audit[2].Action != "candidate.notification.read" {
		t.Fatalf("audit = %+v", audit)
	}

	scoped := performStaffJSON(t, handler, http.MethodGet, "/api/audit?scope=candidate:cand-1", "")
	assertStatus(t, scoped, http.StatusOK)
	var scopedAudit []administrativeAuditView
	decodeData(t, scoped, &scopedAudit)
	if len(scopedAudit) != 3 || scopedAudit[0].Action != "candidate.notification.created" {
		t.Fatalf("scoped audit = %+v", scopedAudit)
	}

	missingScope := performStaffJSON(t, handler, http.MethodGet, "/api/audit", "")
	assertStatus(t, missingScope, http.StatusBadRequest)
}

func TestHTTPHandlerRejectsCandidateAdministrativeOwnerMismatch(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano})

	for _, test := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/candidates/cand-2/documents", `{"id":"doc-2"}`},
		{http.MethodGet, "/api/candidates/cand-2/documents", ""},
		{http.MethodPost, "/api/candidates/cand-2/claims", `{"id":"claim-2"}`},
		{http.MethodGet, "/api/candidates/cand-2/claims", ""},
	} {
		assertStatus(t, performJSON(t, handler, test.method, test.path, test.body), http.StatusForbidden)
	}
}

func TestHTTPHandlerRejectsNotificationRecipientMismatch(t *testing.T) {
	handler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno})
	body := `{"candidate_id":"cand-1","id":"note-1","solicitud_id":"sol-1","type":"aviso","subject":"Subsanacion","body":"Revise su solicitud"}`

	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/notifications", body), http.StatusCreated)
	assertStatus(t, performStaffJSON(t, handler, http.MethodPost, "/api/notifications/note-1/send", `{"csv":"CSV-NOT-1","recipient_id":"cand-2","channel":"vec"}`), http.StatusBadRequest)
	audit := performStaffJSON(t, handler, http.MethodGet, "/api/audit?candidate_id=cand-1", "")
	assertStatus(t, audit, http.StatusOK)
	var entries []administrativeAuditView
	decodeData(t, audit, &entries)
	if len(entries) != 1 || entries[0].Action != "candidate.notification.created" {
		t.Fatalf("audit after rejected mismatch = %+v", entries)
	}
}

func TestHTTPHandlerRejectsWrongRolesAcrossCandidateAndStaffFlows(t *testing.T) {
	staffHandler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "staff", Role: ports.AuthRolePersonalInterno})
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodPost, "/api/candidates", `{"id":"cand-1"}`), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodPost, "/api/candidates/cand-1/merits", `{}`), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodPost, "/api/candidates/cand-1/baremo", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodGet, "/api/candidates/cand-1/expediente", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodGet, "/api/candidates/cand-1/documents", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodGet, "/api/candidates/cand-1/claims", ""), http.StatusForbidden)
	assertStatus(t, performStaffJSON(t, staffHandler, http.MethodPost, "/api/candidates/cand-1/claims", `{}`), http.StatusForbidden)

	candidateHandler := mustTestHandlerWithAdministrativeFlow(t, ports.AuthPrincipal{Subject: "cand-1", Role: ports.AuthRoleCiudadano})
	assertStatus(t, performJSON(t, candidateHandler, http.MethodPost, "/api/demo", ""), http.StatusForbidden)
	assertStatus(t, performJSON(t, candidateHandler, http.MethodGet, "/api/portal", ""), http.StatusForbidden)
	assertStatus(t, performJSON(t, candidateHandler, http.MethodGet, "/api/modules/bolsa", ""), http.StatusForbidden)
	assertStatus(t, performJSON(t, candidateHandler, http.MethodPost, "/api/candidates/cand-1/notifications", `{}`), http.StatusForbidden)
	assertStatus(t, performJSON(t, candidateHandler, http.MethodGet, "/api/candidates/cand-1/notifications", ""), http.StatusForbidden)
	assertStatus(t, performJSON(t, candidateHandler, http.MethodGet, "/api/candidates/cand-1/audit", ""), http.StatusForbidden)
}

func mustTestHandlerWithAdministrativeFlow(t *testing.T, principal ports.AuthPrincipal) http.Handler {
	t.Helper()
	store := repository.NewAdministrativeFlowMemoryStore()
	documents := repository.NewAdministrativeCandidateDocumentRepository(store)
	claims := repository.NewAdministrativeClaimRepository(store)
	notifications := repository.NewAdministrativeNotificationRepository(store)
	audit := repository.NewAdministrativeAuditTrail(store)
	usecase, err := usecases.NewAdministrativeFlowUseCase(documents, claims, notifications, audit)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}
	admin := NewAdministrativeFlowService(documents, usecase)
	handler, err := NewHTTPHandlerWithModules(
		&recordingService{}, nil, admin,
		&recordingAuthenticator{principal: principal},
		testCatalog(t),
	)
	if err != nil {
		t.Fatalf("NewHTTPHandlerWithModules() error = %v", err)
	}
	return handler
}

func testCatalog(t *testing.T) *i18n.Catalog {
	t.Helper()
	catalog, err := i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: {
		"api.candidate.document_registered":  "documento",
		"api.candidate.documents_listed":     "documentos",
		"api.candidate.claim_presented":      "alegacion",
		"api.candidate.claims_listed":        "alegaciones",
		"api.candidate.notification_created": "notificacion",
		"api.candidate.notifications_listed": "notificaciones",
		"api.candidate.notification_sent":    "notificacion enviada",
		"api.candidate.notification_read":    "notificacion leida",
		"api.candidate.audit_listed":         "auditoria",
		"api.module.bolsa.manifest_loaded":   "manifiesto",
		"api.error.bad_request":              "mal json",
		"api.error.unauthorized":             "auth requerida",
		"api.error.forbidden":                "sin permisos",
		"api.error.not_found":                "no encontrado",
		"api.error.method_not_allowed":       "metodo no permitido",
		"api.error.internal":                 "interno",
	}})
	if err != nil {
		t.Fatalf("i18n catalog: %v", err)
	}
	return catalog
}
