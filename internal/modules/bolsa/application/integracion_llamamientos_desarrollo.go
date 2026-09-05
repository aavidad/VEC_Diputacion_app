package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ServicioIntegracionLlamamientosDesarrollo reutiliza el dominio de Bolsa.
// No conoce CT, HTTP, SQL ni claves. La fuente firmada solo sustituye los datos
// corporativos no disponibles; el repositorio debe conservar efectos reales.
type ServicioIntegracionLlamamientosDesarrollo struct {
	fuente      ports.FuenteFirmadaLlamamientosDesarrollo
	repositorio ports.RepositorioLlamamientoDesarrollo
	autorizador ports.AutorizadorLlamamientoDesarrollo
	reloj       ports.RelojLlamamientos
}

func NuevoServicioIntegracionLlamamientosDesarrollo(
	fuente ports.FuenteFirmadaLlamamientosDesarrollo,
	repositorio ports.RepositorioLlamamientoDesarrollo,
	autorizador ports.AutorizadorLlamamientoDesarrollo,
	reloj ports.RelojLlamamientos,
) (*ServicioIntegracionLlamamientosDesarrollo, error) {
	if dependenciaLlamamientoNula(fuente) || dependenciaLlamamientoNula(repositorio) ||
		dependenciaLlamamientoNula(autorizador) || dependenciaLlamamientoNula(reloj) {
		return nil, ports.ErrIntegracionLlamamientoDesarrollo
	}
	return &ServicioIntegracionLlamamientosDesarrollo{fuente, repositorio, autorizador, reloj}, nil
}

func (s *ServicioIntegracionLlamamientosDesarrollo) datos(ctx context.Context, necesidad string) (ports.DatosAutoritativosLlamamiento, error) {
	if ctx == nil || s == nil || !ports.ReferenciaOpacaLlamamientoValida(necesidad) {
		return ports.DatosAutoritativosLlamamiento{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	if err := ctx.Err(); err != nil {
		return ports.DatosAutoritativosLlamamiento{}, err
	}
	datos, err := s.fuente.CargarDatosAutoritativosLlamamiento(ctx, necesidad)
	if err != nil || len(datos) != 1 {
		return ports.DatosAutoritativosLlamamiento{}, ports.ErrDatosLlamamientoNoConfiables
	}
	d, err := datos[0].Clonar()
	if err != nil || d.Necesidad.NecesidadRef != necesidad {
		return ports.DatosAutoritativosLlamamiento{}, ports.ErrDatosLlamamientoNoConfiables
	}
	return d, nil
}

func (s *ServicioIntegracionLlamamientosDesarrollo) ConsultarDisponibilidad(ctx context.Context, necesidadRef string, maximo uint32) (ports.DisponibilidadLlamamientosDesarrollo, error) {
	if maximo == 0 || maximo > 128 {
		return ports.DisponibilidadLlamamientosDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	d, err := s.datos(ctx, necesidadRef)
	if err != nil {
		return ports.DisponibilidadLlamamientosDesarrollo{}, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	i, err := domain.NuevaInstantaneaOrdenBolsa(domain.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: referenciaIntegracionDesarrollo("consulta", necesidadRef), Version: 1, Bolsa: d.Bolsa, ReferidaEn: ahora, GeneradaEn: ahora, Entradas: d.Entradas,
	})
	if err != nil || !d.Bolsa.VigenteEn(ahora) || !d.Politica.VigenteEn(ahora) {
		return ports.DisponibilidadLlamamientosDesarrollo{}, ports.ErrDatosLlamamientoNoConfiables
	}
	r := ports.DisponibilidadLlamamientosDesarrollo{Bolsa: d.Bolsa, Politica: d.Politica, Necesidad: d.Necesidad, CantidadExacta: true}
	for _, entrada := range i.Entradas {
		peticion, err := nuevaSolicitudMotor(d.Necesidad, i, d.Politica, entrada, ahora)
		if err != nil {
			return ports.DisponibilidadLlamamientosDesarrollo{}, err
		}
		e, err := s.fuente.EvaluarParticipacion(ctx, peticion)
		if err != nil || !evaluacionMotorExacta(e, peticion) {
			return ports.DisponibilidadLlamamientosDesarrollo{}, ports.ErrEvaluacionMotorNoConfiable
		}
		if e.Resultado == domain.ResultadoElegible {
			if r.CantidadDisponible == maximo {
				r.CantidadExacta = false
				break
			}
			r.CantidadDisponible++
		}
	}
	return r, nil
}

func (s *ServicioIntegracionLlamamientosDesarrollo) PrepararOrden(ctx context.Context, p ports.PeticionLlamamientoDesarrollo) (ports.ReciboLlamamientoDesarrollo, error) {
	if !peticionIntegracionDesarrolloValida(p) || p.OrdenOperacionRef != "" {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	d, err := s.datos(ctx, p.NecesidadRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	if len(d.Entradas) > int(p.MaximoPosiciones) {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrDatosLlamamientoNoConfiables
	}
	fuente, firma, err := s.fuente.ExportarFuenteFirmada(ctx, p.NecesidadRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	r, encontrado, err := s.repositorio.BuscarOperacion(ctx, p.OperacionRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	if encontrado {
		if r.Tipo != "orden" || r.NecesidadRef != p.NecesidadRef || r.VersionNecesidad != d.Necesidad.Version ||
			!bytes.Equal(r.Fuente, fuente) || !bytes.Equal(r.FirmaFuente, firma) {
			return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
		}
		return s.confirmar(ctx, r)
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	i, err := domain.NuevaInstantaneaOrdenBolsa(domain.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: referenciaIntegracionDesarrollo("orden", p.OperacionRef), Version: 1, Bolsa: d.Bolsa,
		ReferidaEn: ahora, GeneradaEn: ahora, Entradas: d.Entradas,
	})
	if err != nil || !d.Bolsa.VigenteEn(ahora) || !d.Politica.VigenteEn(ahora) ||
		ahora.Before(d.Necesidad.CreadaEn) || !ahora.Before(d.Necesidad.FinPrevisto) {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrDatosLlamamientoNoConfiables
	}
	r = ports.RegistroLlamamientoDesarrollo{
		Esquema: "vec.bolsa.integracion-llamamientos-desarrollo.v1", OperacionRef: p.OperacionRef, Tipo: "orden",
		NecesidadRef: p.NecesidadRef, VersionNecesidad: d.Necesidad.Version, CategoriaRef: d.Necesidad.CategoriaRef, UnidadRef: d.Necesidad.UnidadRef,
		Fuente: fuente, FirmaFuente: firma, Instantanea: i,
	}
	return s.confirmar(ctx, r)
}

func (s *ServicioIntegracionLlamamientosDesarrollo) SolicitarLlamamiento(ctx context.Context, p ports.PeticionLlamamientoDesarrollo) (ports.ReciboLlamamientoDesarrollo, error) {
	if !peticionIntegracionDesarrolloValida(p) || !ports.ReferenciaOpacaLlamamientoValida(p.OrdenOperacionRef) || p.OrdenOperacionRef == p.OperacionRef {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	d, err := s.datos(ctx, p.NecesidadRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	fuente, firma, err := s.fuente.ExportarFuenteFirmada(ctx, p.NecesidadRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	orden, existe, err := s.repositorio.BuscarOperacion(ctx, p.OrdenOperacionRef)
	if err != nil || !existe || orden.Tipo != "orden" || orden.NecesidadRef != p.NecesidadRef ||
		orden.VersionNecesidad != d.Necesidad.Version || !bytes.Equal(orden.Fuente, fuente) ||
		!bytes.Equal(orden.FirmaFuente, firma) || len(orden.Instantanea.Entradas) > int(p.MaximoPosiciones) {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	r, existe, err := s.repositorio.BuscarOperacion(ctx, p.OperacionRef)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	if existe {
		if r.Tipo != "propuesta" || r.OrdenOperacionRef != p.OrdenOperacionRef || r.NecesidadRef != p.NecesidadRef {
			return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
		}
		return s.confirmar(ctx, r)
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	evaluaciones := make([]domain.EvaluacionParticipacionLlamamiento, 0, len(orden.Instantanea.Entradas))
	for _, entrada := range orden.Instantanea.Entradas {
		peticion, err := nuevaSolicitudMotor(d.Necesidad, orden.Instantanea, d.Politica, entrada, ahora)
		if err != nil {
			return ports.ReciboLlamamientoDesarrollo{}, err
		}
		e, err := s.fuente.EvaluarParticipacion(ctx, peticion)
		if err != nil || !evaluacionMotorExacta(e, peticion) {
			return ports.ReciboLlamamientoDesarrollo{}, ports.ErrEvaluacionMotorNoConfiable
		}
		evaluaciones = append(evaluaciones, e)
		if e.Resultado == domain.ResultadoElegible {
			break
		}
		if e.Resultado != domain.ResultadoNoElegible {
			return ports.ReciboLlamamientoDesarrollo{}, ports.ErrEvaluacionMotorNoConfiable
		}
	}
	propuesta, err := domain.ProponerPrimerLlamamiento(domain.OrdenProponerPrimerLlamamiento{
		PropuestaRef: referenciaIntegracionDesarrollo("propuesta", p.OperacionRef), Bolsa: d.Bolsa, Necesidad: d.Necesidad,
		Instantanea: orden.Instantanea, Politica: d.Politica, Evaluaciones: evaluaciones, GeneradaEn: ahora,
	})
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	r = orden
	r.OperacionRef = p.OperacionRef
	r.OrdenOperacionRef = p.OrdenOperacionRef
	r.Tipo = "propuesta"
	r.Propuesta = &propuesta
	abierto, err := domain.NuevoLlamamientoAbierto(domain.DatosLlamamientoAbierto{
		LlamamientoRef: referenciaIntegracionDesarrollo("llamamiento", p.OperacionRef),
		BolsaRef:       propuesta.BolsaRef, NecesidadRef: propuesta.NecesidadRef, PropuestaRef: propuesta.PropuestaRef, Version: 1,
	})
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	llamamiento := abierto.Datos()
	r.Llamamiento = &llamamiento
	r.EstadoLlamamiento = abierto.Estado()
	return s.confirmar(ctx, r)
}

func (s *ServicioIntegracionLlamamientosDesarrollo) confirmar(ctx context.Context, r ports.RegistroLlamamientoDesarrollo) (ports.ReciboLlamamientoDesarrollo, error) {
	recurso, err := r.RecursoAutorizable()
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	material, err := s.autorizador.AutorizarOperacion(ctx, r.Accion(), recurso)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := material.ResumenCapacidad()
	if err != nil || material.ValidarEstructura() != nil || c.Operacion() != r.Accion() ||
		c.EfectoRef() != r.OperacionRef || c.EfectoHuellaSHA256() != h ||
		c.AudienciaConsumo() != ports.AudienciaIntegracionLlamamientoDesarrollo {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	recibo, err := s.repositorio.Guardar(ctx, r, material)
	if err != nil {
		return ports.ReciboLlamamientoDesarrollo{}, err
	}
	esperado, _ := r.Canonico()
	recibido, err := recibo.Registro.Canonico()
	if err != nil || !bytes.Equal(esperado, recibido) || !ports.ReferenciaOpacaLlamamientoValida(recibo.ReciboRef) ||
		!ports.ReferenciaOpacaLlamamientoValida(recibo.AuditoriaRef) || !ports.ReferenciaOpacaLlamamientoValida(recibo.EventoRef) ||
		recibo.ConfirmadaEn.IsZero() || recibo.ConfirmadaEn.Before(r.Instantanea.GeneradaEn) {
		return ports.ReciboLlamamientoDesarrollo{}, ports.ErrIntegracionLlamamientoDesarrollo
	}
	return recibo, nil
}

func peticionIntegracionDesarrolloValida(p ports.PeticionLlamamientoDesarrollo) bool {
	return ports.ReferenciaOpacaLlamamientoValida(p.OperacionRef) && ports.ReferenciaOpacaLlamamientoValida(p.NecesidadRef) &&
		p.MaximoPosiciones > 0 && p.MaximoPosiciones <= 128
}
func referenciaIntegracionDesarrollo(prefijo, operacion string) string {
	b, _ := json.Marshal([]string{prefijo, operacion})
	h := sha256.Sum256(b)
	return ports.ReferenciaDesdeHuellaLlamamientoDesarrollo(prefijo, hex.EncodeToString(h[:]))
}
