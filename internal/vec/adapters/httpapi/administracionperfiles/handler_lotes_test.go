package administracionperfiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type catalogoLoteHTTP map[string]domain.RolAdministrable

func (c catalogoLoteHTTP) ResolverRolAdministrable(_ context.Context, ref string) (domain.RolAdministrable, error) {
	rol, existe := c[ref]
	if !existe {
		return domain.RolAdministrable{}, ErrRecursoNoEncontrado
	}
	return rol, nil
}

type autoridadLoteHTTP struct {
	autoridadFocal
	solicitudes []domain.SolicitudLoteAdministracionPerfiles
	conservado  *domain.ReciboLoteAdministracionPerfiles
	err         error
	alterar     func(*domain.ReciboLoteAdministracionPerfiles)
}

func (a *autoridadLoteHTTP) AplicarLoteOrdinario(_ context.Context, s domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	a.solicitudes = append(a.solicitudes, s)
	if a.err != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, a.err
	}
	if a.conservado != nil {
		if a.conservado.HuellaSolicitudSHA256 != s.HuellaSolicitudSHA256 {
			return domain.ReciboLoteAdministracionPerfiles{}, ErrConflictoEstado
		}
		return *a.conservado, nil
	}
	r := domain.ReciboLoteAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef,
		ReciboRef: "recibo_admin:" + strings.Repeat("a", 32), AuditoriaRef: "auditoria:prueba:lote",
		HuellaSolicitudSHA256: s.HuellaSolicitudSHA256, FuentesSHA256: strings.Repeat("b", 64),
		ConfirmadoEn: s.Actor.ResueltoEn}
	for _, cambio := range s.Cambios {
		p := cambio.Objetivo
		inicio := domain.InicioEfectivoLoteAdministracion{Modo: cambio.InicioVigencia, VigenteDesde: p.VigenteDesde}
		if cambio.InicioVigencia == domain.InicioVigenciaLoteInmediato {
			inicio.VigenteDesde = r.ConfirmadoEn
		}
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			inicio = domain.InicioEfectivoLoteAdministracion{}
		}
		r.Inicios = append(r.Inicios, inicio)
		version, estado := uint64(1), domain.EstadoVinculoContextoActorActivo
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			version, estado = p.VinculoVersion+1, domain.EstadoVinculoContextoActorRevocado
		}
		r.Cambios = append(r.Cambios, domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef,
			ActoRef: r.ActoRef, ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, ConfirmadoEn: r.ConfirmadoEn,
			ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef,
			AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef,
			ObjetivoPersonaRef: p.PersonaRef, PerfilRef: p.PerfilRef, VinculoRef: p.VinculoRef,
			UnidadRef: p.UnidadRef, CentroRef: p.CentroRef, RolVersionRef: cambio.RolVersionRef,
			VersionPosterior: version, EstadoPosterior: estado, VigenteDesde: inicio.VigenteDesde, VigenteHasta: p.VigenteHasta,
			HuellaAntesSHA256: p.HuellaSHA256, HuellaDespuesSHA256: strings.Repeat("f", 64),
			Motivo: s.Motivo, ReferenciaActo: s.ReferenciaActo})
	}
	if a.alterar != nil {
		a.alterar(&r)
	}
	return r, nil
}

func loteHTTPPrueba(t *testing.T) (*Handler, SolicitudLote, *autoridadLoteHTTP, *sesionPrueba, *auditorPrueba, catalogoLoteHTTP) {
	t.Helper()
	sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
	ahora := sesion.resultado.Actor.ResueltoEn
	huella, err := sesion.resultado.InstantaneaAutorizacion.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	catalogo := catalogoLoteHTTP{sesion.resultado.InstantaneaAutorizacion.VersionRol.Referencia(): {
		VersionRef: sesion.resultado.InstantaneaAutorizacion.VersionRol.Referencia(), Clase: domain.ClaseControlPerfilAdministrador,
		CategoriaAdmin: "aplicacion", HuellaSHA256: huella, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}
	dto := SolicitudLote{OperacionRef: "acto_admin:" + strings.Repeat("b", 32), ReferenciaActo: "Resolución 2026/123",
		Motivo: Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "provision"}}
	for i, letra := range []string{"f", "g"} {
		ref := []string{"rol:dietas_liquidacion_rrhh:v1", "rol:gestor_cronos:v1"}[i]
		catalogo[ref] = domain.RolAdministrable{VersionRef: ref, Clase: domain.ClaseControlPerfilOrdinario,
			HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), UnidadRequerida: true}
		dto.Cambios = append(dto.Cambios, CambioPerfil{Operacion: "otorgar", InicioVigencia: "programado", RolVersionRef: ref,
			Objetivo: Objetivo{UnidadRef: "unidad:prueba", CuentaRef: "cta_" + strings.Repeat("f", 22), CuentaVersion: 1,
				PersonaRef: "per_" + strings.Repeat("f", 22), PersonaVersion: 1, PerfilRef: "prf_" + strings.Repeat(letra, 22),
				VinculoRef: "vca_" + strings.Repeat(letra, 22), HuellaSHA256: strings.Repeat("c", 64),
				ProcedenciaRef: "procedencia:maestra:prueba", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("d", 64),
				VigenteDesde: ahora.Add(time.Minute), VigenteHasta: ahora.Add(time.Hour)}})
	}
	autoridad, auditor := &autoridadLoteHTTP{}, &auditorPrueba{}
	servicio, err := application.NuevoServicioAdministracionPerfiles(catalogo, autoridad, relojFocal{ahora})
	if err != nil {
		t.Fatal(err)
	}
	h, err := NuevoHandlerLoteOrdinario("org_prueba", "https://admin.example.test", sesion, &lecturasPrueba{}, catalogo, servicio, auditor)
	if err != nil {
		t.Fatal(err)
	}
	return h, dto, autoridad, sesion, auditor, catalogo
}

func postLotePrueba(t *testing.T, h *Handler, dto SolicitudLote) *httptest.ResponseRecorder {
	t.Helper()
	cuerpo, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/lotes-ordinarios", string(cuerpo)))
	return w
}

func TestHTTPLoteEntregaOrdenIndivisibleYReciboCompleto(t *testing.T) {
	h, dto, autoridad, sesion, auditor, _ := loteHTTPPrueba(t)
	// La segunda fila revoca un vínculo existente; la primera otorga uno nuevo.
	dto.Cambios[1].Operacion = "revocar"
	dto.Cambios[1].InicioVigencia = ""
	dto.Cambios[1].Objetivo.PerfilVersion, dto.Cambios[1].Objetivo.VinculoVersion = 3, 3
	dto.Cambios[1].Objetivo.VigenteDesde, dto.Cambios[1].Objetivo.VigenteHasta = time.Time{}, time.Time{}
	w := postLotePrueba(t, h, dto)
	if w.Code != http.StatusOK || len(autoridad.solicitudes) != 1 || autoridad.llamada != nil || auditor.llamadas != 0 {
		t.Fatalf("orden no indivisible: estado=%d cuerpo=%s", w.Code, w.Body.String())
	}
	solicitud := autoridad.solicitudes[0]
	if solicitud.Validar() != nil || solicitud.Evidencia.ValidarPara(sesion.resultado.Actor) != nil ||
		solicitud.OrganizacionRef != "org_prueba" ||
		solicitud.Actor.PersonaRef != sesion.resultado.Actor.PersonaRef || solicitud.CorrelacionRef != sesion.resultado.CorrelacionRef ||
		solicitud.ReferenciaActo != dto.ReferenciaActo || solicitud.Motivo != dto.Motivo.dominio() {
		t.Fatal("orden perdió identidad confiable o material del efecto")
	}
	var resultado struct {
		Recibo ReciboLote `json:"recibo"`
	}
	if json.Unmarshal(w.Body.Bytes(), &resultado) != nil || len(resultado.Recibo.Cambios) != 2 ||
		resultado.Recibo.HuellaSolicitudSHA256 != solicitud.HuellaSolicitudSHA256 ||
		resultado.Recibo.Cambios[1].EstadoPosterior != "revocado" || resultado.Recibo.Cambios[1].VersionPosterior != 4 ||
		resultado.Recibo.Cambios[0].Motivo != dto.Motivo || resultado.Recibo.Cambios[0].ReferenciaActo != dto.ReferenciaActo {
		t.Fatal("recibo incompleto o cruzado")
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatal("recibo introduce persistencia web")
	}
}

func TestHTTPLoteSinOrganizacionPrivadaPermaneceCerrado(t *testing.T) {
	h, dto, autoridad, sesiones, auditor, catalogo := loteHTTPPrueba(t)
	_, err := NuevoHandlerLoteOrdinario("org_ajena*", "https://admin.example.test", sesiones,
		&lecturasPrueba{}, catalogo, h.actos, auditor)
	if err == nil {
		t.Fatal("organizacion privada invalida")
	}
	h.organizacionLote = ""
	w := postLotePrueba(t, h, dto)
	if w.Code != http.StatusServiceUnavailable || len(autoridad.solicitudes) != 0 || auditor.llamadas != 1 {
		t.Fatal("lote abierto sin organizacion privada")
	}
}

func TestHTTPLoteDeniegaAntesDelEfectoYAudita(t *testing.T) {
	for _, caso := range []string{"vacio", "excesivo", "autoasignacion", "duplicado", "CAS", "sin_unidad", "sin_vigencia", "sensible", "Sistemas", "sin_lote", "lecturas"} {
		t.Run(caso, func(t *testing.T) {
			h, dto, autoridad, sesion, auditor, catalogo := loteHTTPPrueba(t)
			estado := http.StatusBadRequest
			switch caso {
			case "vacio":
				dto.Cambios = nil
			case "excesivo":
				dto.Cambios = make([]CambioPerfil, 33)
			case "autoasignacion":
				for i := range dto.Cambios {
					dto.Cambios[i].Objetivo.PersonaRef = sesion.resultado.Actor.PersonaRef
				}
			case "duplicado":
				dto.Cambios[1] = dto.Cambios[0]
			case "CAS":
				dto.Cambios[1].Objetivo.PersonaVersion++
			case "sin_unidad":
				dto.Cambios[0].Objetivo.UnidadRef, dto.Cambios[0].Objetivo.CentroRef = "", "centro:prueba"
			case "sin_vigencia":
				dto.Cambios[0].Objetivo.VigenteHasta = time.Time{}
			case "sensible":
				rol := catalogo[dto.Cambios[1].RolVersionRef]
				rol.Clase, rol.CategoriaAdmin = domain.ClaseControlPerfilAdministrador, "aplicacion"
				catalogo[rol.VersionRef] = rol
			case "Sistemas":
				rol := catalogo[sesion.resultado.InstantaneaAutorizacion.VersionRol.Referencia()]
				rol.CategoriaAdmin = "sistemas"
				catalogo[rol.VersionRef] = rol
				// Sin la categoría del lote: denegación, igual que en la preparación.
				estado = http.StatusForbidden
			case "sin_lote":
				h.actos, estado = &actosPrueba{}, http.StatusServiceUnavailable
			case "lecturas":
				var err error
				h, err = NuevoHandlerLecturas("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
				if err != nil {
					t.Fatal(err)
				}
				estado = http.StatusServiceUnavailable
			}
			w := postLotePrueba(t, h, dto)
			if w.Code != estado || len(autoridad.solicitudes) != 0 || autoridad.llamada != nil || auditor.llamadas != 1 {
				t.Fatalf("rechazo %s no cerrado/auditado: %d %s", caso, w.Code, w.Body.String())
			}
		})
	}
}

func TestHTTPLoteRechazaCamposNoConfiablesYLimitaCuerpo(t *testing.T) {
	for _, campo := range []string{"actor", "evidencia", "instantanea_autorizacion", "correlacion_ref", "huella_solicitud_sha256", "organizacion_ref", "proponente_nombre", "clase"} {
		t.Run(campo, func(t *testing.T) {
			h, dto, autoridad, _, auditor, _ := loteHTTPPrueba(t)
			cuerpo, _ := json.Marshal(dto)
			var objeto map[string]any
			_ = json.Unmarshal(cuerpo, &objeto)
			objeto[campo] = "cliente"
			cuerpo, _ = json.Marshal(objeto)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/lotes-ordinarios", string(cuerpo)))
			if w.Code != http.StatusBadRequest || len(autoridad.solicitudes) != 0 || auditor.llamadas != 1 {
				t.Fatal("campo cliente admitido")
			}
		})
	}
	for _, conocida := range []bool{true, false} {
		h, _, autoridad, _, auditor, _ := loteHTTPPrueba(t)
		r := peticionADMIN(http.MethodPost, PrefijoV1+"/lotes-ordinarios", strings.Repeat(" ", 64*1024+1))
		if !conocida {
			r.ContentLength = -1
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusRequestEntityTooLarge || len(autoridad.solicitudes) != 0 || auditor.llamadas != 1 {
			t.Fatal("cuerpo excesivo admitido")
		}
	}
}

func TestHTTPLoteRecuperaReciboHistoricoYConflictoNoDaExito(t *testing.T) {
	h, dto, autoridad, sesion, auditor, _ := loteHTTPPrueba(t)
	primera := postLotePrueba(t, h, dto)
	if primera.Code != http.StatusOK {
		t.Fatal(primera.Body.String())
	}
	recibo, err := autoridad.AplicarLoteOrdinario(context.Background(), autoridad.solicitudes[0])
	if err != nil {
		t.Fatal(err)
	}
	autoridad.solicitudes = autoridad.solicitudes[:1]
	autoridad.conservado = &recibo
	sesion.resultado.CorrelacionRef = "correlacion_" + strings.Repeat("9", 32)
	sesion.resultado.InstantaneaAutorizacion.RevisionCatalogoPoliticas++
	replay := postLotePrueba(t, h, dto)
	if replay.Code != http.StatusOK || replay.Body.String() != primera.Body.String() || len(autoridad.solicitudes) != 2 {
		t.Fatal("recuperación cambió recibo")
	}
	dto.ReferenciaActo = "Resolución 2026/456"
	conflicto := postLotePrueba(t, h, dto)
	if conflicto.Code != http.StatusConflict || auditor.llamadas != 1 || strings.Contains(conflicto.Body.String(), "recibo_ref") {
		t.Fatal("conflicto presentó recibo")
	}
}

func TestHTTPLoteReciboParcialYAuditoriaCaidaDeniegan(t *testing.T) {
	h, dto, autoridad, _, auditor, _ := loteHTTPPrueba(t)
	autoridad.alterar = func(r *domain.ReciboLoteAdministracionPerfiles) { r.Cambios = r.Cambios[:1] }
	w := postLotePrueba(t, h, dto)
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 1 || strings.Contains(w.Body.String(), "recibo_ref") {
		t.Fatal("recibo parcial admitido")
	}
	autoridad.alterar, autoridad.err = nil, ErrConflictoEstado
	auditor.err = errors.New("auditoría no disponible")
	w = postLotePrueba(t, h, dto)
	if w.Code != http.StatusServiceUnavailable || auditor.llamadas != 2 {
		t.Fatal("denegación sin auditoría presentada como completada")
	}
}

func TestHTTPLoteMantieneFronteraMTLSYOrigenAntesDeSesion(t *testing.T) {
	for _, caso := range []string{"certificado", "origen", "cabecera_identidad", "sesion_caida"} {
		t.Run(caso, func(t *testing.T) {
			h, dto, autoridad, sesion, auditor, _ := loteHTTPPrueba(t)
			cuerpo, _ := json.Marshal(dto)
			r := peticionADMIN(http.MethodPost, PrefijoV1+"/lotes-ordinarios", string(cuerpo))
			estado, sesiones := http.StatusUnauthorized, 0
			switch caso {
			case "certificado":
				r.TLS.VerifiedChains = nil
			case "origen":
				r.Header.Set("Origin", "https://otro.example.test")
				estado = http.StatusForbidden
			case "cabecera_identidad":
				r.Header.Set("X-Remote-User", "persona_declarada")
			case "sesion_caida":
				sesion.err = errors.New("fuente no disponible")
				estado, sesiones = http.StatusServiceUnavailable, 1
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != estado || sesion.llamadas != sesiones || len(autoridad.solicitudes) != 0 || auditor.llamadas != 1 {
				t.Fatalf("frontera %s admitida o sin auditoría: %d", caso, w.Code)
			}
		})
	}
}

type fuentePropuestaNominalPrueba struct {
	lecturasPrueba
	material Propuesta
}

func (f *fuentePropuestaNominalPrueba) ListarPropuestas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (PaginaPropuestas, error) {
	return PaginaPropuestas{Propuestas: []Propuesta{f.material}}, nil
}

func TestHTTPPropuestaProyectaSoloMaterialDeFuente(t *testing.T) {
	h, _, _, sesion, auditor, _ := loteHTTPPrueba(t)
	material := Propuesta{ProponenteNombre: "Ana Romero", ProponentePerfilNombre: "Administrador de la aplicación",
		Ambitos:      []AmbitoPerfil{{Dimension: "unidad", Referencia: "unidad:prueba", Nombre: "Unidad de personal"}},
		VigenteDesde: sesion.resultado.Actor.ResueltoEn, VigenteHasta: sesion.resultado.Actor.ResueltoEn.Add(time.Hour),
		Motivo: &MotivoLectura{Motivo: Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "provision"}, Etiqueta: "Suplencia"}}
	h.lecturas = &fuentePropuestaNominalPrueba{material: material}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/propuestas", ""))
	var pagina PaginaPropuestas
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &pagina) != nil || len(pagina.Propuestas) != 1 ||
		pagina.Propuestas[0].ProponenteNombre != material.ProponenteNombre ||
		pagina.Propuestas[0].ProponentePerfilNombre != material.ProponentePerfilNombre ||
		pagina.Propuestas[0].Ambitos[0] != material.Ambitos[0] || *pagina.Propuestas[0].Motivo != *material.Motivo || auditor.llamadas != 0 {
		t.Fatal("material legible no procede de la fuente")
	}
	// Una fuente antigua no recibe nombres ni fechas fabricados por el handler.
	h.lecturas = &fuentePropuestaNominalPrueba{}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionADMIN(http.MethodGet, PrefijoV1+"/propuestas", ""))
	for _, campo := range []string{"proponente_nombre", "proponente_perfil_nombre", "ambitos", "vigente_desde", "vigente_hasta", "\"motivo\":"} {
		if strings.Contains(w.Body.String(), campo) {
			t.Fatalf("fuente vacía inventó %s", campo)
		}
	}
}
