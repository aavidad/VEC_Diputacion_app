package application

import (
	"context"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Mismas formas que BC14/BC15: la base de datos vuelve a comprobarlas todas.
var (
	referenciaVersionPublicacionOficial = regexp.MustCompile(`^cv1_[0-9a-f]{64}_v[1-9][0-9]{0,15}$`)
	huellaPublicacionOficial            = regexp.MustCompile(`^[0-9a-f]{64}$`)
	anuncioPublicacionOficial           = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9./_-]{0,79}$`)
	direccionPublicacionOficial         = regexp.MustCompile(`^https://[A-Za-z0-9.-]{3,253}(/[A-Za-z0-9._~:/?#@!$&()*+,;=%-]*)?$`)
	sustitucionPublicacionOficial       = regexp.MustCompile(`^publicacion_bases:[0-9a-f]{64}$`)
	clavePublicacionOficial             = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,128}$`)
)

// MediosPublicacionOficial son los boletines y tablones que admite BC14.
var MediosPublicacionOficial = []string{"bop_granada", "tablon_edictos_electronico", "boe", "boja"}

type ServicioPublicacionOficialBases struct {
	autorizador ports.AutorizadorSituacionParticipacionV3
	repositorio ports.RepositorioPublicacionOficial
}

func NuevoServicioPublicacionOficialBases(a ports.AutorizadorSituacionParticipacionV3, r ports.RepositorioPublicacionOficial) (*ServicioPublicacionOficialBases, error) {
	if a == nil || r == nil {
		return nil, ports.ErrPublicacionOficialNoDisponible
	}
	return &ServicioPublicacionOficialBases{a, r}, nil
}

// Declarar resuelve la versión vigente y su ámbito en la base de datos, pide
// una decisión V3 ligada a ese recurso y al material exacto, y la entrega a
// BC15, que la consume en la misma transacción que escribe la declaración.
func (s *ServicioPublicacionOficialBases) Declarar(ctx context.Context, q ports.SolicitudDeclaracionPublicacionOficial) (ports.ReciboPublicacionOficial, error) {
	var cero ports.ReciboPublicacionOficial
	if s == nil || ctx == nil || s.autorizador == nil || s.repositorio == nil {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	q.Declaracion = copiarDeclaracionPublicacionOficial(q.Declaracion)
	if q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Correlacion.Validar() != nil || !vd.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return cero, ports.ErrPublicacionOficialInvalida
	}
	if err := ValidarDeclaracionPublicacionOficial(q.Declaracion); err != nil {
		return cero, err
	}
	d := q.Declaracion
	pagina, err := s.repositorio.ConsultarVersiones(ctx, d.ConvocatoriaRef, 1)
	if err != nil {
		return cero, errorPublicacionOficial(err)
	}
	if len(pagina.Items) != 1 || pagina.Items[0].ConvocatoriaRef != d.ConvocatoriaRef {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	version := pagina.Items[0]
	if version.HuellaVersionSHA256 != d.HuellaVersionSHA256 {
		return cero, ports.ErrPublicacionOficialNoVigente
	}
	if version.UnidadRef == "" || version.AmbitoRef == "" {
		return cero, vd.ErrAutorizacionDenegada
	}
	canon, huella, err := s.repositorio.NormalizarDeclaracion(ctx, d)
	if err != nil {
		return cero, errorPublicacionOficial(err)
	}
	if canon == "" || !huellaPublicacionOficial.MatchString(huella) {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	recurso := vd.RecursoAutorizable{Referencia: d.ConvocatoriaRef, ModuloID: ports.ModuloPublicacionOficialBases,
		Tipo:      ports.TipoRecursoVersionConvocatoria,
		Ambitos:   map[string]string{"unidad_ref": version.UnidadRef, "ambito_ref": version.AmbitoRef},
		Atributos: map[string]string{"material_sha256": huella}}
	auth, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion,
		Accion: ports.AccionDeclararPublicacionOficialBases, Recurso: recurso,
		Finalidad: ports.FinalidadDeclararPublicacionOficialBases, Correlacion: q.Correlacion})
	if err != nil {
		return cero, vd.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil {
		return cero, errorPublicacionOficial(err)
	}
	if exportador == nil || decision.ValidarPara(auth) != nil {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto,
		q.MotivoAutorizacion, material, ports.AudienciaDeclararPublicacionOficialBases) {
		return cero, ports.ErrPublicacionOficialNoDisponible
	}
	return s.repositorio.Declarar(ctx, ports.IntencionPublicacionOficial{Solicitud: q, CanonPostgreSQL: canon,
		HuellaSHA256: huella, Material: material})
}

// ValidarDeclaracionPublicacionOficial comprueba la forma antes de pedir una
// decisión; la fecha futura y el resto de reglas las vuelve a imponer BC14.
func ValidarDeclaracionPublicacionOficial(d ports.DeclaracionPublicacionOficial) error {
	if !referenciaVersionPublicacionOficial.MatchString(d.ConvocatoriaRef) ||
		!huellaPublicacionOficial.MatchString(d.HuellaVersionSHA256) ||
		!clavePublicacionOficial.MatchString(d.ClaveIdempotencia) ||
		!anuncioPublicacionOficial.MatchString(d.AnuncioRef) {
		return ports.ErrPublicacionOficialInvalida
	}
	medio := false
	for _, m := range MediosPublicacionOficial {
		medio = medio || d.Medio == m
	}
	tablon := d.Medio == "tablon_edictos_electronico"
	if !medio || tablon != (d.BoletinNumero == nil) ||
		(d.BoletinNumero != nil && (*d.BoletinNumero < 1 || *d.BoletinNumero > 99999)) {
		return ports.ErrPublicacionOficialInvalida
	}
	fecha, err := time.Parse(time.DateOnly, d.FechaPublicacion)
	if err != nil || fecha.Format(time.DateOnly) != d.FechaPublicacion || fecha.Year() < 2000 {
		return ports.ErrPublicacionOficialInvalida
	}
	if d.Direccion != nil && (len(*d.Direccion) < 12 || len(*d.Direccion) > 512 || !direccionPublicacionOficial.MatchString(*d.Direccion)) {
		return ports.ErrPublicacionOficialInvalida
	}
	if d.SustituyeRef != nil && !sustitucionPublicacionOficial.MatchString(*d.SustituyeRef) {
		return ports.ErrPublicacionOficialInvalida
	}
	return nil
}

func copiarDeclaracionPublicacionOficial(d ports.DeclaracionPublicacionOficial) ports.DeclaracionPublicacionOficial {
	if d.BoletinNumero != nil {
		v := *d.BoletinNumero
		d.BoletinNumero = &v
	}
	if d.Direccion != nil {
		v := *d.Direccion
		d.Direccion = &v
	}
	if d.SustituyeRef != nil {
		v := *d.SustituyeRef
		d.SustituyeRef = &v
	}
	return d
}

func errorPublicacionOficial(err error) error {
	if errors.Is(err, vp.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return vd.ErrAutorizacionDenegada
	}
	for _, conocido := range []error{vd.ErrAutorizacionDenegada, vd.ErrPermissionDenied,
		ports.ErrPublicacionOficialInvalida, ports.ErrPublicacionOficialConflicto,
		ports.ErrPublicacionOficialNoVigente, ports.ErrPublicacionOficialNoConfirmada} {
		if errors.Is(err, conocido) {
			return err
		}
	}
	return ports.ErrPublicacionOficialNoDisponible
}
