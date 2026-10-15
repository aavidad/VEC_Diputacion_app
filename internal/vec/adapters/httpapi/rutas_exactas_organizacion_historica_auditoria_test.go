package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRutasExactasOrganizacionHistoricaAuditaAntesDelHandlerYFallaCerrado(t *testing.T) {
	for _, caso := range []struct {
		nombre         string
		denegacion     error
		auditoriaCaida bool
		estado         int
		motivo         ports.MotivoAuditoriaFronteraRutaExacta
	}{
		{"sin autenticacion", ErrAutenticacionRutaExactaRequerida, false, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
		{"sin permiso", ErrAccesoRutaExactaDenegado, false, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"auditor caido sin autenticacion", ErrAutenticacionRutaExactaRequerida, true, http.StatusServiceUnavailable, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
		{"auditor caido sin permiso", ErrAccesoRutaExactaDenegado, true, http.StatusServiceUnavailable, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			negocio := &manejadorExactoPrueba{}
			autoridad := &autoridadRutasExactasEspia{err: caso.denegacion}
			auditor := &registradorAuditoriaFronteraRutaExactaEspia{}
			if caso.auditoriaCaida {
				auditor.err = errors.New("detalle privado de PostgreSQL")
			}
			h, err := NewHandlerSoloRutasExactas([]RutaExacta{{Ruta: RutaOrganizacionHistoricaPersonal, Manejador: negocio}}, autoridad, auditor)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaOrganizacionHistoricaPersonal, nil))
			if w.Code != caso.estado {
				t.Fatalf("estado=%d", w.Code)
			}
			ordenes := auditor.ordenesRegistradas()
			if len(ordenes) != 1 || ordenes[0].Validar() != nil || ordenes[0].Superficie != ports.SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal ||
				ordenes[0].Ruta != RutaOrganizacionHistoricaPersonal || ordenes[0].Motivo != caso.motivo {
				t.Fatalf("auditoria=%#v", ordenes)
			}
			if llamadas, _, _ := negocio.estado(); llamadas != 0 {
				t.Fatal("el handler recibió una petición denegada")
			}
			if strings.Contains(w.Body.String(), "detalle privado") {
				t.Fatal("la respuesta filtró el fallo del auditor")
			}
		})
	}
}
