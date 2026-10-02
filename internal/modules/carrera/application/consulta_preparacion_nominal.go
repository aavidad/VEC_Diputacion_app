package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"time"
	"vec-diputacion-granada/internal/modules/carrera/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrPreparacionNominalNoDisponible = errors.New("carrera.preparacion.no_disponible")
var ErrPreparacionNominalDenegada = errors.New("carrera.preparacion.denegada")

// El resultado reutiliza el dato propietario de Personal; no guarda otra ficha
// ni convierte periodos [Desde,Hasta) a los del preparador sintético.
type PreparacionNominal struct {
	Estado       string                                       `json:"estado"`
	Antecedentes personalports.ResultadoAntecedentesCarreraV1 `json:"antecedentes"`
	Pendientes   []string                                     `json:"pendientes"`
}

type ServicioConsultaPreparacionNominal struct {
	raiz   ports.FuenteConsultaAutorizadaPreparacionCarrera
	lector personalports.LectorAntecedentesCarreraV1
	ahora  func() time.Time
}

func NuevoServicioConsultaPreparacionNominal(r ports.FuenteConsultaAutorizadaPreparacionCarrera, l personalports.LectorAntecedentesCarreraV1, reloj func() time.Time) (*ServicioConsultaPreparacionNominal, error) {
	if nuloPreparacion(r) || nuloPreparacion(l) || reloj == nil {
		return nil, ErrPreparacionNominalNoDisponible
	}
	return &ServicioConsultaPreparacionNominal{r, l, reloj}, nil
}

// Campos cerrados para el primer consumidor. El catálogo central deberá
// acordarlos; declararlos aquí no publica ni concede ningún permiso.
func CamposConsultaPreparacionCarrera() []string {
	return []string{"antecedentes_meta", "relaciones", "servicios", "puestos", "procedencia", "recibo"}
}
func RecursoConsultaPreparacionCarrera(q personalports.ConsultaAntecedentesCarreraV1) (core.RecursoAutorizable, error) {
	if q.Actor.Validar() != nil || !personaldomain.ReferenciaEmpleadoValida(q.EmpleadoRef) || q.Corte.Validar() != nil {
		return core.RecursoAutorizable{}, ErrPreparacionNominalNoDisponible
	}
	r := core.RecursoAutorizable{Referencia: q.EmpleadoRef, ModuloID: "carrera", Tipo: "preparacion_carrera", Ambitos: map[string]string{"empleado_ref": q.EmpleadoRef, "organismo_ref": q.OrganismoRef}, Atributos: map[string]string{"vigente_en": q.Corte.VigenteEn.Texto(), "conocido_en": q.Corte.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z")}}
	if r.Validar() != nil {
		return core.RecursoAutorizable{}, ErrPreparacionNominalNoDisponible
	}
	return r, nil
}

func (s *ServicioConsultaPreparacionNominal) Consultar(ctx context.Context) (PreparacionNominal, error) {
	var cero PreparacionNominal
	if s == nil || ctx == nil || ctx.Err() != nil || nuloPreparacion(s.raiz) || nuloPreparacion(s.lector) || s.ahora == nil {
		return cero, ErrPreparacionNominalNoDisponible
	}
	q, orden, confirmacion, err := s.raiz.ResolverConsultaPreparacionCarrera(ctx)
	if err != nil {
		return cero, errorPreparacionNominal(err)
	}
	if err := consultaCarreraAutorizada(q, orden, confirmacion, s.ahora()); err != nil {
		return cero, errorPreparacionNominal(err)
	}
	if ctx.Err() != nil {
		return cero, ErrPreparacionNominalNoDisponible
	}
	q.Actor, err = q.Actor.Clonar()
	if err != nil {
		return cero, ErrPreparacionNominalNoDisponible
	}
	resultado, err := s.lector.ConsultarAntecedentesCarrera(ctx, q)
	if err != nil {
		return cero, errorPreparacionNominal(err)
	}
	if ctx.Err() != nil || !confirmacion.DentroDeVentanaEn(s.ahora().UTC().Truncate(time.Microsecond)) || !resultadoNominalValido(q, resultado) {
		return cero, ErrPreparacionNominalNoDisponible
	}
	resultado = proyectarAntecedentesNominales(resultado)
	return PreparacionNominal{Estado: "pendiente", Antecedentes: resultado, Pendientes: []string{"grupo", "grado", "politica"}}, nil
}

func consultaCarreraAutorizada(q personalports.ConsultaAntecedentesCarreraV1, o vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3, c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ahora time.Time) error {
	esperado, err := RecursoConsultaPreparacionCarrera(q)
	if err != nil {
		return err
	}
	if err := c.ValidarPara(o); err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	if !c.DentroDeVentanaEn(ahora.UTC().Truncate(time.Microsecond)) {
		return ErrPreparacionNominalNoDisponible
	}
	d, err := o.Datos()
	if err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	sol, err := d.Solicitud.Datos()
	if err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	if sol.Accion != ports.AccionConsultaPreparacionCarrera || sol.Finalidad != ports.FinalidadConsultaPreparacionCarrera || sol.Recurso.Referencia != esperado.Referencia || sol.Recurso.ModuloID != esperado.ModuloID || sol.Recurso.Tipo != esperado.Tipo {
		return ErrPreparacionNominalNoDisponible
	}
	if err := d.Decision.ExigirProyeccionPara(d.Solicitud, CamposConsultaPreparacionCarrera(), nil); err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	h, err := esperado.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	actual, err := sol.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	if h != actual {
		return ErrPreparacionNominalNoDisponible
	}
	canon, err := q.Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	if !bytes.Equal(canon, d.ResultadoContexto.RepresentacionCanonica) {
		return ErrPreparacionNominalNoDisponible
	}
	if err := sol.VinculoAutenticacionActor.ValidarPara(d.ResultadoContexto); err != nil {
		return errors.Join(ErrPreparacionNominalNoDisponible, err)
	}
	return nil
}

func resultadoNominalValido(q personalports.ConsultaAntecedentesCarreraV1, r personalports.ResultadoAntecedentesCarreraV1) bool {
	if r.EmpleadoRef != q.EmpleadoRef || r.OrganismoRef != q.OrganismoRef || r.Version <= 0 || r.Corte.VigenteEn != q.Corte.VigenteEn || !r.Corte.ConocidoEn.Equal(q.Corte.ConocidoEn) || len(r.Relaciones) > 128 || len(r.Servicios) > 128 || len(r.Puestos) > 128 {
		return false
	}
	if r.Cobertura != personalports.CoberturaPersonalCompletaV1 && r.Cobertura != personalports.CoberturaPersonalParcialV1 && r.Cobertura != personalports.CoberturaPersonalNoAcreditadaV1 {
		return false
	}
	for _, fila := range r.Relaciones {
		if !refNominal(fila.RelacionRef) || fila.Version <= 0 || !periodoNominal(fila.Periodo) || !refNominal(fila.Estado) || !refNominal(fila.RegimenRef) || fila.RegimenVersion < 0 || !procedenciaNominal(fila.Procedencia) {
			return false
		}
	}
	for _, fila := range r.Servicios {
		if !refNominal(fila.ServicioRef) || !refNominal(fila.RelacionRef) || fila.Version <= 0 || !periodoNominal(fila.Periodo) || !refNominal(fila.Estado) || !refNominal(fila.ClaseRef) || fila.ClaseVersion < 0 || !procedenciaNominal(fila.Procedencia) {
			return false
		}
	}
	for _, fila := range r.Puestos {
		if !refNominal(fila.PuestoRef) || !refNominal(fila.PuestoVersion) || !refNominal(fila.RelacionRef) || !periodoNominal(fila.Periodo) || (fila.Nivel != nil && *fila.Nivel < 0) || !procedenciaNominal(fila.Procedencia) {
			return false
		}
	}
	e := r.Evidencia
	for _, ref := range []string{e.ReciboRef, e.DecisionRef, e.EfectoRef, e.AuditoriaRef} {
		if strings.TrimSpace(ref) == "" || len(ref) > 160 {
			return false
		}
	}
	if e.EfectoRef != q.EmpleadoRef || len(e.ConsumoHuellaSHA256) != 64 || e.ConsultadaEn.IsZero() {
		return false
	}
	_, err := hex.DecodeString(e.ConsumoHuellaSHA256)
	return err == nil && strings.ToLower(e.ConsumoHuellaSHA256) == e.ConsumoHuellaSHA256
}
func proyectarAntecedentesNominales(r personalports.ResultadoAntecedentesCarreraV1) personalports.ResultadoAntecedentesCarreraV1 {
	r.Relaciones = append([]personalports.RelacionAntecedenteCarreraV1{}, r.Relaciones...)
	r.Servicios = append([]personalports.ServicioParaCertificadosV1{}, r.Servicios...)
	r.Puestos = append([]personalports.PuestoAntecedenteCarreraV1{}, r.Puestos...)
	r.Situaciones = nil
	for i, p := range r.Puestos {
		if p.Nivel != nil {
			n := *p.Nivel
			r.Puestos[i].Nivel = &n
		}
	}
	return r
}
func errorPreparacionNominal(err error) error {
	if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) && !errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return ErrPreparacionNominalDenegada
	}
	return ErrPreparacionNominalNoDisponible
}
func nuloPreparacion(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func refNominal(s string) bool { return strings.TrimSpace(s) != "" && len(s) <= 1024 }
func periodoNominal(p personalports.PeriodoPersonalNominalV1) bool {
	return p.Desde.Validar() == nil && (p.Hasta == "" || (p.Hasta.Validar() == nil && p.Desde.AntesDe(p.Hasta)))
}
func procedenciaNominal(p personalports.ProcedenciaPersonalNominalV1) bool {
	return len(p.ActoRef) <= 1024 && len(p.FuenteRef) <= 1024 && len(p.FuenteVersion) <= 1024 && (p.Certeza == personalports.CertezaPersonalAcreditadaV1 || p.Certeza == personalports.CertezaPersonalPendienteV1 || p.Certeza == personalports.CertezaPersonalNoAcreditadaV1)
}
