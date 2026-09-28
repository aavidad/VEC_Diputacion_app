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
		conActor   bool
	}{
		{"opciones sin autenticacion", http.MethodGet, "/api/vec/auditoria/opciones", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true, false},
		{"opciones sin permiso", http.MethodGet, "/api/vec/auditoria/opciones", ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true, false},
		{"consultas sin autenticacion", http.MethodPost, "/api/vec/auditoria/consultas", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true, false},
		{"consultas sin permiso", http.MethodPost, "/api/vec/auditoria/consultas", ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaAuditoria, true, false},
		{"contratacion", http.MethodPost, rutaAltaContratacionPrueba, ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal, true, false},
		{"usuarios sin autenticacion", http.MethodGet, "/api/vec/usuarios/mis-preferencias", ErrAutenticacionRutaExactaRequerida, http.StatusUnauthorized, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, ports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, true, false},
		{"usuarios sin permiso", http.MethodPut, "/api/vec/usuarios/mis-preferencias", ErrAccesoRutaExactaDenegado, http.StatusForbidden, ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado, ports.SuperficieAuditoriaFronteraRutaExactaUsuariosPreferencias, true, true},
		{"prefijo auditoria", http.MethodGet, "/api/vec/auditoria/opciones_extra", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false, false},
		{"otra ruta auditoria", http.MethodGet, "/api/vec/auditoria/otra", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false, false},
		{"prefijo usuarios", http.MethodGet, "/api/vec/usuarios/mis-preferencias-extra", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false, false},
		{"otra ruta usuarios", http.MethodGet, "/api/vec/usuarios/otras-preferencias", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false, false},
		{"personal", http.MethodGet, "/api/vec/personal/vacantes", ErrAccesoRutaExactaDenegado, http.StatusForbidden, "", "", false, false},
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
			peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
			if caso.conActor {
				ctx, err := ConActorVerificadoAuditoriaPreferenciasUsuarios(peticion.Context(), actorOrganizacionHistoricaPrueba(t))
				if err != nil {
					t.Fatal(err)
				}
				peticion = peticion.WithContext(ctx)
			}
			handler.ServeHTTP(respuesta, peticion)
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
			if caso.conActor && ordenes[0].ActorRef != actorOrganizacionHistoricaPrueba(t).PersonaRef ||
				!caso.conActor && ordenes[0].ActorRef != "" {
				t.Fatalf("actor no minimizado: %q", ordenes[0].ActorRef)
			}
		})
	}
}

func TestRutasExactasUsuariosRechazanDuplicadoYNoInventanActor(t *testing.T) {
	const ruta = "/api/vec/usuarios/mis-preferencias"
	manejador := &manejadorExactoPrueba{}
	autoridad := &autoridadRutasExactasEspia{err: ErrAccesoRutaExactaDenegado}
	auditoria := &registradorAuditoriaFronteraRutaExactaEspia{}
	declarada := RutaExacta{Ruta: ruta, Manejador: manejador}
	if h, err := NewHandlerSoloRutasExactas([]RutaExacta{declarada, declarada}, autoridad, auditoria); h != nil || err != ErrRutaExactaInvalida {
		t.Fatalf("duplicado aceptado: (%T, %v)", h, err)
	}
	h, err := NewHandlerSoloRutasExactas([]RutaExacta{declarada}, autoridad, auditoria)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, ruta, nil))
	if w.Code != http.StatusForbidden || len(auditoria.ordenesRegistradas()) != 0 {
		t.Fatalf("403 sin actor verificado produjo auditoria: estado=%d ordenes=%#v", w.Code, auditoria.ordenesRegistradas())
	}
}
