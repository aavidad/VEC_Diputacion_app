package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRutasExactasAuditoriaEtiquetanSoloRutasNominales(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre     string
		metodo     string
		ruta       string
		denegacion error
		estado     int
		motivo     ports.MotivoAuditoriaFronteraRutaExacta
		superficie string
		auditada   bool
	}{
		{"opciones sin autenticacion", http.MethodGet, "/api/vec/auditoria/opciones", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true},
		{"opciones sin permiso", http.MethodGet, "/api/vec/auditoria/opciones", ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true},
		{"consultas sin autenticacion", http.MethodPost, "/api/vec/auditoria/consultas", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true},
		{"consultas sin permiso", http.MethodPost, "/api/vec/auditoria/consultas", ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true},
		{"contratacion", http.MethodPost, rutaAltaContratacionPrueba, ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal, true},
		{"prefijo auditoria", http.MethodGet, "/api/vec/auditoria/opciones_extra", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false},
		{"otra ruta auditoria", http.MethodGet, "/api/vec/auditoria/otra", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false},
		{"personal", http.MethodGet, "/api/vec/personal/vacantes", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			manejador := &manejadorExactoPrueba{}
			autoridad := &autoridadRutasExactasEspia{err: caso.denegacion}
			auditoria := &registradorAuditoriaFronteraRutaExactaEspia{}
			handler, err := NewHandlerSoloRutasExactas(
				[]RutaExacta{{Ruta: caso.ruta, Manejador: manejador}}, autoridad, auditoria,
			)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, httptest.NewRequest(caso.metodo, caso.ruta, nil))
			if respuesta.Code != caso.estado {
				t.Fatalf("estado=%d", respuesta.Code)
			}
			if llamadas, ruta := autoridad.estado(); llamadas != 1 || ruta != caso.ruta {
				t.Fatalf("autoridad=(%d, %q)", llamadas, ruta)
			}
			if llamadas, _, _ := manejador.estado(); llamadas != 0 {
				t.Fatalf("negocio invocado %d veces", llamadas)
			}
			ordenes := auditoria.ordenesRegistradas()
			if !caso.auditada {
				if len(ordenes) != 0 {
					t.Fatalf("ruta ajena obtuvo auditoria: %#v", ordenes)
				}
				return
			}
			if len(ordenes) != 1 || ordenes[0].Validar() != nil ||
				ordenes[0].Superficie != caso.superficie || ordenes[0].Ruta != caso.ruta ||
				ordenes[0].Motivo != caso.motivo {
				t.Fatalf("auditoria=%#v", ordenes)
			}
		})
	}
}
