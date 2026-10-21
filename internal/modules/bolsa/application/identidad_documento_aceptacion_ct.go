package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaIdentidadDocumentoAceptacionCT = "vec.bolsa.aceptacion-nativa-ct.identidad-documento.consulta.v1"

var camposIdentidadDocumentoAceptacionCT = []string{"apellidos", "documento_enmascarado", "nombre", "version_fuente"}

func PrepararIdentidadDocumentoAceptacionCT(q ports.SolicitudIdentidadDocumentoAceptacionCT) (ports.PreparacionIdentidadDocumentoAceptacionCT, error) {
	x := q.Selector
	for _, ref := range []string{x.OrganizacionRef, x.ExpedienteRef, x.ClaveCT, x.UnidadRef, x.CategoriaRef, x.NecesidadRef, x.BolsaRef, x.LlamamientoRef, x.ReciboEmisionRef, x.ReciboRespuestaRef} {
		if !referenciaAceptacionNativaCT.MatchString(ref) {
			return ports.PreparacionIdentidadDocumentoAceptacionCT{}, ports.ErrIdentidadDocumentoAceptacionCTInvalida
		}
	}
	if q.ActorConfiable.Resultado.Validar() != nil || q.ActorConfiable.Vinculo.ValidarPara(q.ActorConfiable.Resultado) != nil {
		return ports.PreparacionIdentidadDocumentoAceptacionCT{}, ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	actor, err := q.ActorConfiable.Resultado.Clonar()
	if err != nil {
		return ports.PreparacionIdentidadDocumentoAceptacionCT{}, ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	q.ActorConfiable.Resultado = actor
	material, err := json.Marshal(struct {
		Esquema string `json:"esquema"`
		ports.SelectorAceptacionNativaCT
	}{EsquemaIdentidadDocumentoAceptacionCT, x})
	if err != nil {
		return ports.PreparacionIdentidadDocumentoAceptacionCT{}, ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	h := sha256.Sum256(material)
	recurso := core.RecursoAutorizable{
		Referencia: x.ReciboRespuestaRef, ModuloID: "bolsa", Tipo: ports.TipoIdentidadDocumentoAceptacionCT,
		Ambitos:   map[string]string{"unidad_ref": x.UnidadRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if _, err = recurso.HuellaContextoAutorizacionSHA256(); err != nil {
		return ports.PreparacionIdentidadDocumentoAceptacionCT{}, ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	return ports.PreparacionIdentidadDocumentoAceptacionCT{Solicitud: q, MaterialCanonico: material, Recurso: recurso}, nil
}

func ValidarOrdenIdentidadDocumentoAceptacionCT(o ports.OrdenIdentidadDocumentoAceptacionCT, ahora time.Time) error {
	p, err := PrepararIdentidadDocumentoAceptacionCT(o.Solicitud)
	if err != nil {
		return err
	}
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	a := o.Material
	x := a.ResumenCapacidad()
	c := p.Solicitud.ActorConfiable.Resultado
	if err != nil || a.ValidarEstructura() != nil || !instantePersonaAceptacionCT(ahora) ||
		x.Operacion() != ports.AccionIdentidadDocumentoAceptacionCT || x.AudienciaConsumo() != ports.AudienciaIdentidadDocumentoAceptacionCT ||
		x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h ||
		x.ContextoRef() != c.RegistroContextoRef || x.ContextoHuellaSHA256() != c.HuellaSHA256 ||
		!bytes.Equal(a.ContextoActorCanonico(), c.RepresentacionCanonica) ||
		a.PersonaVersion() != c.Contexto.Instantanea.PersonaVersion || a.PerfilVersion() != c.Contexto.Instantanea.PerfilVersion ||
		ahora.Before(x.EmitidaEn()) || !ahora.Before(x.ExpiraEn()) {
		return ports.ErrIdentidadDocumentoAceptacionCTDenegada
	}
	return nil
}

func ValidarLecturaFuenteIdentidadDocumentoAceptacionCT(o ports.OrdenIdentidadDocumentoAceptacionCT, r ports.LecturaFuenteIdentidadDocumentoAceptacionCT, ahora time.Time) error {
	if ValidarOrdenIdentidadDocumentoAceptacionCT(o, ahora) != nil {
		return ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	x, e := o.Material.ResumenCapacidad(), r.Evidencia
	if !referenciaPersonaAceptacionCT.MatchString(e.DecisionRef) || e.DecisionRef != x.DecisionRef() ||
		!huellaPersonaAceptacionCT.MatchString(e.ConsumoHuellaSHA256) || !referenciaPersonaAceptacionCT.MatchString(e.AuditoriaRef) ||
		!instantePersonaAceptacionCT(e.ConsultadaEn) || e.ConsultadaEn.Before(x.EmitidaEn()) ||
		!e.ConsultadaEn.Before(x.ExpiraEn()) || e.ConsultadaEn.After(ahora) {
		return ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	if r.Estado == "no_encontrada" || r.Estado == "pendiente" {
		if r.Fuente == nil {
			return nil
		}
		return ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	f, s := r.Fuente, o.Solicitud.Selector
	if r.Estado != "acreditado" || f == nil || f.FilaNumero < 2 || f.VersionInstantanea == 0 ||
		f.BolsaRef != s.BolsaRef || f.ReciboEmisionRef != s.ReciboEmisionRef || f.ReciboRespuestaRef != s.ReciboRespuestaRef ||
		!referenciaAceptacionNativaCT.MatchString(f.ActaRef) || !referenciaAceptacionNativaCT.MatchString(f.InstantaneaRef) ||
		!huellaPersonaAceptacionCT.MatchString(f.HuellaInstantaneaSHA256) || !huellaPersonaAceptacionCT.MatchString(f.HuellaListadoSHA256) {
		return ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	return nil
}

type ServicioIdentidadDocumentoAceptacionCT struct {
	proveedor   ports.ProveedorAutorizacionIdentidadDocumentoAceptacionCT
	repositorio ports.RepositorioIdentidadDocumentoAceptacionCT
	recuperador constitucion.Recuperador
	ahora       func() time.Time
}

func NuevoServicioIdentidadDocumentoAceptacionCT(p ports.ProveedorAutorizacionIdentidadDocumentoAceptacionCT, r ports.RepositorioIdentidadDocumentoAceptacionCT, recuperador constitucion.Recuperador, ahora func() time.Time) (*ServicioIdentidadDocumentoAceptacionCT, error) {
	if nuloPersonaAceptacionCT(p) || nuloPersonaAceptacionCT(r) || nuloPersonaAceptacionCT(recuperador) || ahora == nil {
		return nil, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	return &ServicioIdentidadDocumentoAceptacionCT{p, r, recuperador, ahora}, nil
}

func (s *ServicioIdentidadDocumentoAceptacionCT) ConsultarIdentidadDocumentoAceptacionCT(ctx context.Context, q ports.SolicitudIdentidadDocumentoAceptacionCT) (ports.ResultadoIdentidadDocumentoAceptacionCT, error) {
	var cero ports.ResultadoIdentidadDocumentoAceptacionCT
	if ctx == nil || s == nil || nuloPersonaAceptacionCT(s.proveedor) || nuloPersonaAceptacionCT(s.repositorio) || nuloPersonaAceptacionCT(s.recuperador) || s.ahora == nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	p, err := PrepararIdentidadDocumentoAceptacionCT(q)
	if err != nil {
		return cero, err
	}
	m, err := s.proveedor.AutorizarIdentidadDocumentoAceptacionCT(ctx, p)
	if err != nil {
		return cero, errorIdentidadDocumentoAceptacionCT(ctx, err)
	}
	o := ports.OrdenIdentidadDocumentoAceptacionCT{Solicitud: p.Solicitud, Material: m}
	if err = ValidarOrdenIdentidadDocumentoAceptacionCT(o, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	lectura, err := s.repositorio.ConsultarFuenteIdentidadDocumentoAceptacionCT(ctx, o)
	if err != nil {
		return cero, errorIdentidadDocumentoAceptacionCT(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = ValidarLecturaFuenteIdentidadDocumentoAceptacionCT(o, lectura, s.ahora().UTC().Truncate(time.Microsecond)); err != nil {
		return cero, err
	}
	r := ports.ResultadoIdentidadDocumentoAceptacionCT{Estado: lectura.Estado, Evidencia: lectura.Evidencia}
	if lectura.Estado != "acreditado" {
		return r, nil
	}
	f := lectura.Fuente
	lote, estado, existe, err := s.recuperador.RecuperarLote(ctx, f.HuellaListadoSHA256, q.Selector.CategoriaRef)
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if err != nil {
		return cero, fmt.Errorf("%w: %w", ports.ErrIdentidadDocumentoAceptacionCTNoDisponible, err)
	}
	if !existe || lote.Validar() != nil || estado.Validar() != nil ||
		estado.EstadoStaging != importacionapp.EstadoStagingDisponible ||
		!estado.Acta.CoincideExactamente(lote.Acta) ||
		lote.Acta.ActaRef != f.ActaRef || (lote.Acta.BolsaRef != "" && lote.Acta.BolsaRef != f.BolsaRef) ||
		lote.Acta.CategoriaRef != q.Selector.CategoriaRef || lote.Acta.HuellaFicheroSHA256 != f.HuellaListadoSHA256 {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	identidad, unica := identidadFilaDocumentoAceptacionCT(lote.Aceptadas, f.FilaNumero)
	if !unica {
		r.Estado = "pendiente"
		return r, nil
	}
	r.Identidad = &ports.IdentidadDocumentoAceptacionCT{Nombre: strings.Join(strings.Fields(identidad.Nombre), " "),
		Apellidos:            strings.Join(strings.Fields(identidad.PrimerApellido+" "+identidad.SegundoApellido), " "),
		DocumentoEnmascarado: identidad.Documento, VersionFuente: f.VersionInstantanea}
	if r.Identidad.Nombre == "" || r.Identidad.Apellidos == "" {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	return r, nil
}

var _ ports.ConsultaIdentidadDocumentoAceptacionCT = (*ServicioIdentidadDocumentoAceptacionCT)(nil)

func errorIdentidadDocumentoAceptacionCT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTInvalida) {
		return ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	if errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTDenegada) || errors.Is(ErrorConsultaPersonaAceptacionCT(ctx, err), ports.ErrConsultaPersonaAceptacionCTDenegada) {
		return ports.ErrIdentidadDocumentoAceptacionCTDenegada
	}
	return fmt.Errorf("%w: %w", ports.ErrIdentidadDocumentoAceptacionCTNoDisponible, err)
}

type ProveedorNominalIdentidadDocumentoAceptacionCT struct {
	emisor      ports.AutorizadorSituacionParticipacionV3
	motivo      core.ReferenciaEntradaCatalogo
	correlacion func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error)
	ahora       func() time.Time
}

func NuevoProveedorNominalIdentidadDocumentoAceptacionCT(e ports.AutorizadorSituacionParticipacionV3, m core.ReferenciaEntradaCatalogo, c func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error), ahora func() time.Time) (*ProveedorNominalIdentidadDocumentoAceptacionCT, error) {
	if nuloPersonaAceptacionCT(e) || !core.ReferenciaMotivoAutorizacionV2Valida(m) || c == nil || ahora == nil {
		return nil, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	return &ProveedorNominalIdentidadDocumentoAceptacionCT{e, m, c, ahora}, nil
}

func (p *ProveedorNominalIdentidadDocumentoAceptacionCT) AutorizarIdentidadDocumentoAceptacionCT(ctx context.Context, entrada ports.PreparacionIdentidadDocumentoAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nuloPersonaAceptacionCT(p.emisor) || p.correlacion == nil || p.ahora == nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	preparada, err := PrepararIdentidadDocumentoAceptacionCT(entrada.Solicitud)
	if err != nil || !bytes.Equal(entrada.MaterialCanonico, preparada.MaterialCanonico) ||
		!reflect.DeepEqual(entrada.Recurso, preparada.Recurso) {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTInvalida
	}
	correlacion, err := p.correlacion(ctx)
	if err != nil {
		return cero, errorIdentidadDocumentoAceptacionCT(ctx, err)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: preparada.Solicitud.ActorConfiable.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: ports.AccionIdentidadDocumentoAceptacionCT, Recurso: preparada.Recurso,
		Finalidad: ports.FinalidadIdentidadDocumentoAceptacionCT, Correlacion: correlacion,
	})
	if err != nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTDenegada
	}
	c := preparada.Solicitud.ActorConfiable.Resultado
	d, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil {
		return cero, errorIdentidadDocumentoAceptacionCT(ctx, err)
	}
	concedida, _, resultadoErr := d.Resultado()
	if d.ValidarPara(s) != nil || resultadoErr != nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	if !concedida {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTDenegada
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, d, p.motivo, c)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	restricciones, err := d.RestriccionesProyeccionPara(s)
	ahora := p.ahora().UTC().Truncate(time.Microsecond)
	if err != nil || !reflect.DeepEqual(restricciones.CamposPermitidos, camposIdentidadDocumentoAceptacionCT) ||
		len(restricciones.Obligaciones) != 0 || !confirmacion.DentroDeVentanaEn(ahora) {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTDenegada
	}
	if nuloPersonaAceptacionCT(exportador) {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(s, d, confirmacion, c, p.motivo, m, ports.AudienciaIdentidadDocumentoAceptacionCT) {
		return cero, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible
	}
	if err = ValidarOrdenIdentidadDocumentoAceptacionCT(ports.OrdenIdentidadDocumentoAceptacionCT{Solicitud: preparada.Solicitud, Material: m}, ahora); err != nil {
		return cero, err
	}
	return m, nil
}

var _ ports.ProveedorAutorizacionIdentidadDocumentoAceptacionCT = (*ProveedorNominalIdentidadDocumentoAceptacionCT)(nil)

// La misma clave de identidad que usa la constitución detecta colisiones sin
// volver a derivar HMAC ni atribuir una fila dudosa a otra persona.
func identidadFilaDocumentoAceptacionCT(filas []importacion.FilaAceptada, numero int) (importacion.IdentidadEnmascarada, bool) {
	var elegida importacion.IdentidadEnmascarada
	claveElegida := ""
	coincidencias := make(map[string]int, len(filas))
	for _, fila := range filas {
		clave, err := constitucion.ClaveIdentidadCandidato(fila.Identidad)
		if err != nil {
			return importacion.IdentidadEnmascarada{}, false
		}
		coincidencias[clave]++
		if fila.Numero == numero {
			elegida = fila.Identidad
			claveElegida = clave
		}
	}
	if claveElegida == "" || coincidencias[claveElegida] != 1 {
		return importacion.IdentidadEnmascarada{}, false
	}
	enmascarado, err := constitucion.EnmascararDocumento(elegida.Documento)
	if err != nil {
		return importacion.IdentidadEnmascarada{}, false
	}
	elegida.Documento = enmascarado
	return elegida, true
}
