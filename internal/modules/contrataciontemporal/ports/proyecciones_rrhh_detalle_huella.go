package ports

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"hash"
	"time"
)

const dominioHuellaDetalleRRHH = "vec.ct.proyeccion_rrhh.detalle.v1"

type escritorHuellaDetalleRRHH struct {
	destino hash.Hash
	entero  [8]byte
}

func calcularHuellaDetalleRRHH(
	detalle DetalleExpedienteRRHH,
) [sha256.Size]byte {
	destino := sha256.New()
	e := escritorHuellaDetalleRRHH{destino: destino}
	e.cadena(dominioHuellaDetalleRRHH)
	e.resumen(detalle.Resumen)
	e.solicitud(detalle.Solicitud)
	e.analisis(detalle.Analisis)
	e.cobertura(detalle.Cobertura)
	e.asignacion(detalle.Asignacion)
	e.hitos(detalle.Hitos)
	var resultado [sha256.Size]byte
	copy(resultado[:], destino.Sum(nil))
	return resultado
}

func (d DetalleExpedienteRRHH) huellaCoincide() bool {
	calculada := calcularHuellaDetalleRRHH(d)
	return subtle.ConstantTimeCompare(d.huella[:], calculada[:]) == 1
}

func (e *escritorHuellaDetalleRRHH) resumen(r ResumenExpedienteRRHH) {
	e.cadena(r.ExpedienteRef)
	e.cadena(r.OrganizacionRef)
	e.cadena(r.NumeroVisible)
	e.u64(r.Version)
	e.cadena(r.FlujoRef)
	e.u64(r.FlujoVersion)
	e.cadena(r.FlujoHuella)
	e.cadena(string(r.FaseClave))
	e.cadena(string(r.EstadoClave))
	e.cadena(r.CentroRef)
	e.cadena(r.CategoriaRef)
	e.cadena(string(r.ModalidadClave))
	e.cadena(r.UnidadRef)
	e.instante(r.CreadoEn)
	e.instante(r.ActualizadoEn)
}

func (e *escritorHuellaDetalleRRHH) solicitud(s SolicitudOperativaRRHH) {
	e.cadena(s.GrupoSubgrupo)
	e.cadena(string(s.MotivoClave))
	e.instante(s.PeriodoInicio)
	e.instante(s.PeriodoFin)
}

func (e *escritorHuellaDetalleRRHH) analisis(a *AnalisisOperativoRRHH) {
	e.presente(a != nil)
	if a == nil {
		return
	}
	e.cadena(string(a.ModalidadClave))
	e.cadena(a.CategoriaRef)
	e.cadena(string(a.CausaClave))
	e.instante(a.PeriodoInicio)
	e.instante(a.PeriodoFin)
	e.u64(uint64(a.PorcentajeJornada))
	e.cadena(string(a.ResultadoRC))
	e.presente(a.CostePrevisto != nil)
	if a.CostePrevisto != nil {
		e.i64(a.CostePrevisto.Centimos)
		e.cadena(a.CostePrevisto.Moneda)
	}
	e.cadena(a.FuenteCosteRef)
}

func (e *escritorHuellaDetalleRRHH) cobertura(c *CoberturaOperativaRRHH) {
	e.presente(c != nil)
	if c == nil {
		return
	}
	e.cadena(string(c.ViaClave))
	e.presente(c.DecisionGobernada)
	e.cadena(c.ProcedimientoRef)
	e.cadena(c.BolsaRef)
	e.u64(uint64(len(c.Comprobaciones)))
	for _, comprobacion := range c.Comprobaciones {
		e.cadena(string(comprobacion.Clave))
		e.cadena(string(comprobacion.Resultado))
	}
}

func (e *escritorHuellaDetalleRRHH) asignacion(a *AsignacionOperativaRRHH) {
	e.presente(a != nil)
	if a == nil {
		return
	}
	e.cadena(a.UnidadRef)
	e.instante(a.AsignadaEn)
	e.cadena(string(a.MotivoClave))
}

func (e *escritorHuellaDetalleRRHH) hitos(hitos []HitoExpedienteRRHH) {
	e.u64(uint64(len(hitos)))
	for _, hito := range hitos {
		e.u64(hito.Secuencia)
		e.u64(hito.VersionExpediente)
		e.cadena(string(hito.AccionClave))
		e.instante(hito.RealizadaEn)
		e.cadena(string(hito.FaseOrigen))
		e.cadena(string(hito.FaseDestino))
		e.cadena(string(hito.EstadoOrigen))
		e.cadena(string(hito.EstadoDestino))
	}
}

func (e *escritorHuellaDetalleRRHH) cadena(valor string) {
	e.u64(uint64(len(valor)))
	_, _ = e.destino.Write([]byte(valor))
}

func (e *escritorHuellaDetalleRRHH) instante(valor time.Time) {
	e.i64(valor.Unix())
	e.i64(int64(valor.Nanosecond()))
}

func (e *escritorHuellaDetalleRRHH) presente(valor bool) {
	if valor {
		_, _ = e.destino.Write([]byte{1})
		return
	}
	_, _ = e.destino.Write([]byte{0})
}

func (e *escritorHuellaDetalleRRHH) i64(valor int64) {
	e.u64(uint64(valor))
}

func (e *escritorHuellaDetalleRRHH) u64(valor uint64) {
	binary.BigEndian.PutUint64(e.entero[:], valor)
	_, _ = e.destino.Write(e.entero[:])
}
