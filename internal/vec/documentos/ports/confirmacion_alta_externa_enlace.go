package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConfirmarAltaExternaEnlace    = "documentos.externo.alta_enlace.confirmar"
	AudienciaConfirmarAltaExternaEnlace = "vec_documentos.alta_externa_enlace.v1"
	FinalidadConfirmarAltaExternaEnlace = "confirmar_alta_externa_para_enlace"
)

// SolicitudConfirmacionAltaExternaEnlace liga la constancia de Documentos al
// material Cronos que consumirá la misma transacción. ActorAltaRef y ClaveAlta
// son los de la operación de alta original; el actor de esta lectura puede
// ser distinto.
type SolicitudConfirmacionAltaExternaEnlace struct {
	DocumentoID, ExpedienteRef, TipoRef string
	Version                             uint64
	ContenidoSHA256                     string
	CustodioID, CustodiaRef             string
	ClaveAlta, ActorAltaRef             string
	SolicitudRef, MaterialEnlaceSHA256  string
}

var personaRefEnlace = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
var solicitudRefCronosEnlace = regexp.MustCompile(`^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)

func principalRefEnlaceValida(s string) bool {
	return domain.ReferenciaOpacaValida(s) || personaRefEnlace.MatchString(s)
}

func (s SolicitudConfirmacionAltaExternaEnlace) Preimagen() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(s.DocumentoID) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.TipoRef) || s.Version == 0 || !domain.HuellaValida(s.ContenidoSHA256) ||
		!domain.IdentificadorTecnicoValido(s.CustodioID) || !domain.ReferenciaCustodioValida(s.CustodiaRef) ||
		!domain.ReferenciaOpacaValida(s.ClaveAlta) || !principalRefEnlaceValida(s.ActorAltaRef) ||
		!solicitudRefCronosEnlace.MatchString(s.SolicitudRef) || !domain.HuellaValida(s.MaterialEnlaceSHA256) {
		return nil, ErrSolicitudInvalida
	}
	return json.Marshal(struct {
		Accion               string `json:"accion"`
		DocumentoID          string `json:"documento_id"`
		Version              uint64 `json:"version"`
		ExpedienteRef        string `json:"expediente_ref"`
		TipoRef              string `json:"tipo_ref"`
		ModuloOrigen         string `json:"modulo_origen"`
		CustodioID           string `json:"custodio_id"`
		CustodiaRef          string `json:"custodia_ref"`
		ContenidoSHA256      string `json:"contenido_sha256"`
		ClaveAlta            string `json:"clave_alta"`
		ActorRef             string `json:"actor_ref"`
		SolicitudRef         string `json:"solicitud_ref"`
		MaterialEnlaceSHA256 string `json:"material_enlace_sha256"`
	}{AccionConfirmarAltaExternaEnlace, s.DocumentoID, s.Version, s.ExpedienteRef, s.TipoRef,
		"cronos", s.CustodioID, s.CustodiaRef, s.ContenidoSHA256, s.ClaveAlta,
		s.ActorAltaRef, s.SolicitudRef, s.MaterialEnlaceSHA256})
}

func (s SolicitudConfirmacionAltaExternaEnlace) RecursoV3() (vecdomain.RecursoAutorizable, error) {
	p, err := s.Preimagen()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	suma := sha256.Sum256(p)
	return vecdomain.RecursoAutorizable{
		Referencia: s.DocumentoID, ModuloID: "documentos", Tipo: "alta_externa_enlace",
		Ambitos:   map[string]string{"organizacion_ref": OrganizacionRefV3},
		Atributos: map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])},
	}, nil
}

// AutorizacionConfirmacionAltaExternaEnlace procede del PDP para la misma
// identidad autenticada de la petición Cronos. Se valida antes de entregar
// los bytes a la fachada SQL, que vuelve a verificar y consumir la decisión.
type AutorizacionConfirmacionAltaExternaEnlace struct {
	Material                                     vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	PrincipalID, PerfilActivoRef, CorrelacionRef string
}

func (a AutorizacionConfirmacionAltaExternaEnlace) Validar(s SolicitudConfirmacionAltaExternaEnlace, ahora time.Time) error {
	p, err := s.Preimagen()
	if err != nil || a.Material.ValidarEstructura() != nil ||
		!principalRefEnlaceValida(a.PrincipalID) || !domain.ReferenciaValida(a.PerfilActivoRef) ||
		!domain.ReferenciaValida(a.CorrelacionRef) || ahora.IsZero() {
		return ErrSolicitudInvalida
	}
	r := a.Material.ResumenCapacidad()
	if r.ValidarEstructura() != nil || r.AudienciaConsumo() != AudienciaConfirmarAltaExternaEnlace ||
		r.Operacion() != AccionConfirmarAltaExternaEnlace || r.EfectoRef() != s.DocumentoID ||
		r.EfectoHuellaSHA256() != HuellaEfectoV3(p) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) {
		return ErrSolicitudInvalida
	}
	return nil
}

func (a AutorizacionConfirmacionAltaExternaEnlace) AuthJSON(s SolicitudConfirmacionAltaExternaEnlace) ([]byte, error) {
	return json.Marshal(struct {
		Accion          string `json:"accion"`
		Finalidad       string `json:"finalidad"`
		RecursoRef      string `json:"recurso_ref"`
		AmbitoRef       string `json:"ambito_ref"`
		PrincipalID     string `json:"principal_id"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		CorrelacionRef  string `json:"correlacion_ref"`
	}{AccionConfirmarAltaExternaEnlace, FinalidadConfirmarAltaExternaEnlace,
		s.DocumentoID, s.ExpedienteRef, a.PrincipalID, a.PerfilActivoRef, a.CorrelacionRef})
}

// ProveedorConfirmacionAltaExternaEnlace usa el actor y perfil de la petición
// confiable. La implementación no acepta permiso aportado por el cliente.
type ProveedorConfirmacionAltaExternaEnlace interface {
	AutorizarConfirmacionAltaExternaEnlace(context.Context, SolicitudConfirmacionAltaExternaEnlace) (AutorizacionConfirmacionAltaExternaEnlace, error)
}
