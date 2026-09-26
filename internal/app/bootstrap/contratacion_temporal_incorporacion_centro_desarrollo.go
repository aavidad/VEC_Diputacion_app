package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Confirmación de la incorporación por el centro (duda 11, CT124 y AD3-88)
// dentro del circuito de peticiones de centro: misma identidad nominal del
// centro, misma autoridad de autorización y mismo motivo. Qué perfiles del
// centro confirman y con qué documento lo decide la regla c21 del catálogo.
const (
	rutaBandejaIncorporacionCentro      = "/api/vec/contratacion-temporal/peticiones-centro/incorporaciones"
	rutaConfirmacionIncorporacionCentro = "/api/vec/contratacion-temporal/peticiones-centro/incorporaciones/confirmaciones"
	maximoCuerpoIncorporacionCentro     = 8 * 1024
)

var errIncorporacionCentroDesarrollo = errors.New("contratacion temporal: confirmacion del centro de desarrollo no disponible")

func rutaIncorporacionCentroDesarrollo(ruta string) bool {
	return ruta == rutaBandejaIncorporacionCentro || ruta == rutaConfirmacionIncorporacionCentro
}

// incorporacionCentroDesarrollo es la composición: apagada no concede nada
// ni monta rutas (la conducta de hoy).
type incorporacionCentroDesarrollo struct {
	activo bool
	fuente fuenteReglaAcreditacionDesarrollo
	roles  []string
	reloj  relojContratacionTemporalDesarrollo
}

func nuevaIncorporacionCentroDesarrollo(cfg config.Config, reloj relojContratacionTemporalDesarrollo) (*incorporacionCentroDesarrollo, error) {
	if !incorporacionAcreditadaSolicitada(cfg) {
		return &incorporacionCentroDesarrollo{}, nil
	}
	consulta, err := fichero.NuevaConsultaCatalogos(cfg.Normalize().ReglasEjemplo.CTSourcePath)
	if err != nil {
		return nil, errors.Join(errIncorporacionCentroDesarrollo, err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: reglas.CatalogoContratacionTemporal, ModuloID: reglas.ModuloContratacionTemporal, Reloj: reloj, MunicipioSede: reglas.MunicipioSedeDiputacion})
	if err != nil {
		return nil, errors.Join(errIncorporacionCentroDesarrollo, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	regla, err := resolutor.Regla(ctx, reglas.CTAcreditacionIncorporacion)
	if err != nil {
		return nil, errors.Join(errIncorporacionCentroDesarrollo, err)
	}
	var roles []string
	for _, rol := range strings.Split(regla.Atributos[reglas.AtributoRolesConfirmanIncorporacion], ",") {
		if rol = strings.TrimSpace(rol); rol == "solicitante_centro" || rol == "ratificador_centro" {
			roles = append(roles, rol)
		}
	}
	// Sin perfiles que confirmen la regla es inservible: se detiene el arranque.
	if len(roles) == 0 || len(regla.Elementos()) != 1 {
		return nil, errIncorporacionCentroDesarrollo
	}
	return &incorporacionCentroDesarrollo{activo: true, fuente: fuenteReglaAcreditacionDesarrollo{reglas: resolutor}, roles: roles, reloj: reloj}, nil
}

// concesiones añade la bandeja a los dos perfiles del centro y la
// confirmación a los que la regla c21 designa.
func (i *incorporacionCentroDesarrollo) concesiones(rol string) []vecdomain.ConcesionRol {
	if i == nil || !i.activo {
		return nil
	}
	acciones := []string{ports.AccionConsultarIncorporacionesCentro}
	if slices.Contains(i.roles, rol) {
		acciones = append(acciones, ports.AccionConfirmarIncorporacionCentro)
	}
	concesiones := make([]vecdomain.ConcesionRol, 0, len(acciones))
	for _, accion := range acciones {
		concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: accion, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoIncorporacionCentro,
			Finalidades: []string{finalidadPeticionCentro}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
	}
	return concesiones
}

func (i *incorporacionCentroDesarrollo) rutas(p *proveedorPeticionCentroDesarrollo) ([]vechttp.RutaExacta, error) {
	if i == nil || !i.activo {
		return nil, nil
	}
	if p == nil || p.alta == nil {
		return nil, errIncorporacionCentroDesarrollo
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := comprobarMigracionesIncorporacionAcreditadaDesarrollo(ctx, p.alta.postgresql.ejecucion); err != nil {
		return nil, err
	}
	repositorio, err := postgresct.NuevoRepositorioIncorporacionCentroPostgreSQL(p.alta.postgresql.ejecucion, p)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioIncorporacionCentro(repositorio, i.fuente, i.reloj)
	if err != nil {
		return nil, err
	}
	m := &manejadorIncorporacionCentroDesarrollo{proveedor: p, servicio: servicio, roles: i.roles}
	return []vechttp.RutaExacta{{Ruta: rutaBandejaIncorporacionCentro, Manejador: m}, {Ruta: rutaConfirmacionIncorporacionCentro, Manejador: m}}, nil
}

// recursoIncorporacionCentroDesarrollo liga la solicitud de autorización al
// recurso exacto que se va a consumir en SQL.
type recursoIncorporacionCentroDesarrollo struct {
	accion  string
	recurso vecdomain.RecursoAutorizable
	actor   domain.ActorPeticionCentro
}

func (r *recursoIncorporacionCentroDesarrollo) validaPara(principal, perfil string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	return r != nil && (r.accion == ports.AccionConsultarIncorporacionesCentro || r.accion == ports.AccionConfirmarIncorporacionCentro) &&
		d.Accion == r.accion && r.actor.ActorRef == principal && r.actor.PerfilRef == perfil &&
		d.Recurso.Referencia == r.recurso.Referencia && d.Recurso.ModuloID == r.recurso.ModuloID && d.Recurso.Tipo == r.recurso.Tipo &&
		d.Recurso.Tipo == ports.TipoRecursoIncorporacionCentro && maps.Equal(d.Recurso.Ambitos, r.recurso.Ambitos) &&
		maps.Equal(d.Recurso.Atributos, r.recurso.Atributos) && d.Recurso.Ambitos["centro_ref"] == r.actor.CentroRef &&
		d.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo
}

var _ postgresct.ProveedorAutorizacionIncorporacionCentro = (*proveedorPeticionCentroDesarrollo)(nil)

// AutorizarIncorporacionCentro exige, con la identidad del centro de esta
// petición, la decisión V3 para el recurso exacto y entrega su material.
func (p *proveedorPeticionCentroDesarrollo) AutorizarIncorporacionCentro(ctx context.Context, accion string, recurso vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a, err := p.identidad(ctx)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrIncorporacionCentroDenegada
	}
	m := materialAutorizacionPeticionCentroDesarrollo{incorporacion: &recursoIncorporacionCentroDesarrollo{accion: accion, recurso: recurso, actor: a.actor}}
	return p.autorizar(ctx, a.actor, accion, recurso, m)
}

// fuenteReglaAcreditacionDesarrollo lee la regla c21: «valor_<modalidad>» o,
// si no existe, el único elemento de «valor».
type fuenteReglaAcreditacionDesarrollo struct {
	reglas *reglas.Resolutor
}

func (f fuenteReglaAcreditacionDesarrollo) DocumentoAcreditativo(ctx context.Context, modalidad domain.ClaveCatalogo) (domain.ClaveCatalogo, ports.ReglaDocumentoIncorporacion, error) {
	if f.reglas == nil || !modalidad.Valida() {
		return "", ports.ReglaDocumentoIncorporacion{}, ports.ErrReglaAcreditacionNoDisponible
	}
	regla, err := f.reglas.Regla(ctx, reglas.CTAcreditacionIncorporacion)
	if err != nil {
		return "", ports.ReglaDocumentoIncorporacion{}, ports.ErrReglaAcreditacionNoDisponible
	}
	tipo := strings.TrimSpace(regla.Atributos[reglas.PrefijoValorModalidad+string(modalidad)])
	if tipo == "" {
		if elementos := regla.Elementos(); len(elementos) == 1 {
			tipo = strings.TrimSpace(elementos[0])
		}
	}
	r := ports.ReglaDocumentoIncorporacion{Referencia: regla.Referencia, HuellaSHA256: regla.HuellaCatalogo, ModalidadClave: string(modalidad)}
	if !domain.ClaveCatalogo(tipo).Valida() || !r.Valida() {
		return "", ports.ReglaDocumentoIncorporacion{}, ports.ErrReglaAcreditacionNoDisponible
	}
	return domain.ClaveCatalogo(tipo), r, nil
}

type manejadorIncorporacionCentroDesarrollo struct {
	proveedor *proveedorPeticionCentroDesarrollo
	servicio  *application.ServicioIncorporacionCentro
	roles     []string
}

func (m *manejadorIncorporacionCentroDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	fallo := func(estado int, codigo string) {
		w.WriteHeader(estado)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo,
			"clave_i18n": "api.contratacion_temporal.incorporacion_centro.error." + codigo}})
	}
	if m == nil || m.proveedor == nil || m.servicio == nil || r == nil || r.URL == nil {
		fallo(http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if !rutaIncorporacionCentroDesarrollo(r.URL.Path) || r.URL.RawPath != "" || r.URL.RawQuery != "" || len(r.TransferEncoding) != 0 ||
		len(r.Trailer) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	escritura := r.URL.Path == rutaConfirmacionIncorporacionCentro
	metodo := http.MethodGet
	if escritura {
		metodo = http.MethodPost
	}
	if r.Method != metodo {
		w.Header().Set("Allow", metodo)
		fallo(http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	a, err := m.proveedor.identidad(r.Context())
	if err != nil {
		slog.Info("incorporación del centro denegada: identidad", "ruta", r.URL.Path, "causa", err)
		fallo(http.StatusForbidden, "operacion_denegada")
		return
	}
	responder := func(estado int, data any) {
		w.WriteHeader(estado)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}
	if !escritura {
		if r.ContentLength != 0 {
			fallo(http.StatusBadRequest, "solicitud_invalida")
			return
		}
		filas, err := m.servicio.Bandeja(r.Context(), organizacionAltaContratacionTemporalDesarrollo, a.actor)
		if err != nil {
			fallo(estadoErrorIncorporacionCentro(err))
			return
		}
		responder(http.StatusOK, map[string]any{"esquema": "vec.contratacion-temporal.incorporaciones-centro.v1", "expedientes": filas,
			"puede_confirmar": slices.Contains(m.roles, a.principal.Roles[0]), "limite": ports.LimiteExpedientesIncorporacionCentro()})
		return
	}
	if !slices.Contains(m.roles, a.principal.Roles[0]) {
		fallo(http.StatusForbidden, "operacion_denegada")
		return
	}
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 0 && (len(params) != 1 || !strings.EqualFold(params["charset"], "utf-8")) ||
		r.ContentLength > maximoCuerpoIncorporacionCentro || r.Body == nil {
		slog.Info("incorporación del centro rechazada: tipo de contenido", "causa", err)
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoIncorporacionCentro))
	if err != nil || validarClavesJSONUnicas(b) != nil {
		slog.Info("incorporación del centro rechazada: cuerpo", "causa", err)
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	var in struct {
		ClaveIdempotencia  string `json:"clave_idempotencia"`
		PeticionRef        string `json:"peticion_ref"`
		ExpedienteRef      string `json:"expediente_ref"`
		FechaIncorporacion string `json:"fecha_incorporacion"`
		DocumentoRef       string `json:"documento_ref"`
		DocumentoSHA256    string `json:"documento_sha256"`
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(&struct{}{}) != io.EOF {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	recibo, err := m.servicio.Confirmar(r.Context(), application.SolicitudConfirmarIncorporacionCentro{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		Actor: a.actor, ClaveIdempotencia: in.ClaveIdempotencia, PeticionRef: in.PeticionRef, ExpedienteRef: in.ExpedienteRef,
		FechaIncorporacion: in.FechaIncorporacion, DocumentoRef: in.DocumentoRef, DocumentoSHA256: in.DocumentoSHA256})
	if err != nil {
		fallo(estadoErrorIncorporacionCentro(err))
		return
	}
	responder(http.StatusCreated, recibo)
}

func estadoErrorIncorporacionCentro(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrIncorporacionCentroInvalida):
		return http.StatusBadRequest, "solicitud_invalida"
	case errors.Is(err, ports.ErrIncorporacionCentroDenegada), errors.Is(err, domain.ErrRatificacionCentroDenegada):
		return http.StatusForbidden, "operacion_denegada"
	case errors.Is(err, ports.ErrClaveIncorporacionCentroUsada):
		return http.StatusConflict, "clave_reutilizada"
	case errors.Is(err, ports.ErrIncorporacionCentroNoAdmitida):
		return http.StatusConflict, "no_admitida"
	}
	return http.StatusServiceUnavailable, "servicio_no_disponible"
}
