package gobiernoconvocatorias

import (
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// HuellaCuerpoParaRevalidacion permite a persistencia construir la solicitud
// de firma B antes de que exista AcreditacionKMS. Valida todo el cuerpo
// provisional y reutiliza sin cambios la fórmula canónica privada.
func (r ProyeccionReciboBorrador) HuellaCuerpoParaRevalidacion() (string, error) {
	if !cuerpoReciboBorradorValido(r, r.IdentidadPrimaria) {
		return "", ErrResultadoBorradorInseguro
	}
	huella := huellaCuerpoReciboBorrador(r)
	if !huellaHexValida(huella) {
		return "", ErrResultadoBorradorInseguro
	}
	return huella, nil
}

// huellaCuerpoReciboBorrador compromete el recibo completo salvo su propia
// acreditación KMS, que contiene esta huella. La exclusión evita un ciclo sin
// dejar fuera decisión, sellado, auditoría, outbox, procedencia ni controles.
func huellaCuerpoReciboBorrador(r ProyeccionReciboBorrador) string {
	identidad, valida := representacionIdentidadCanonica(r.IdentidadPrimaria)
	if !valida || !r.Procedencia.valida() {
		return ""
	}
	d := r.Decision
	a := d.AtestacionPDP
	s := r.SelladoMotivo
	h := s.HMAC
	return huellaJSONCanonicaBorrador(struct {
		Esquema, ReciboRef, TransaccionRef, Accion string
		EstadoRef, EstadoHuella                    string
		EstadoRevision                             int
		Identidad                                  representacionIdentidadCanonicaBorrador
		Decision                                   struct {
			EsquemaHuella, DecisionRef, HuellaDecision, Accion, RecursoRef   string
			ModuloID, TipoRecurso, ContextoHuella, Finalidad                 string
			AsignacionRef, AsignacionHuella, VersionRolRef, VersionRolHuella string
			ControlRolRef, ControlRolHuella, CatalogoHuella                  string
			ControlRolRevision, RevisionCatalogo                             uint64
			EmitidaEn, VerificadaEn, ValidaHasta                             string
			Atestacion                                                       struct {
				DecisionRef, AtestacionRef, Estado, Huella, VerificadorRef string
				Version                                                    uint32
				VerificadaEn                                               string
			}
		}
		Sellado struct {
			Accion, ConvocatoriaRef, AtestacionRef, Estado, Huella string
			TokenConsumoRef, MaterializadorRef                     string
			Version                                                uint32
			EmitidaEn, ValidaHasta                                 string
			HMAC                                                   struct {
				Dominio, ClaveRef, Valor string
				Generacion               uint32
			}
		}
		Revision, Cercado                           uint64
		ArrendamientoIniciaEn, ArrendamientoVenceEn string
		AuditoriaRef, HuellaAuditoria               string
		EventoRef, HuellaEvento                     string
		ConfirmadaEn                                string
		Procedencia                                 struct {
			Esquema, Perfil, Autoridad, ProveedorRef string
			Migrable                                 bool
		}
	}{
		Esquema: r.Esquema, ReciboRef: r.ReciboRef, TransaccionRef: r.TransaccionRef,
		Accion: r.Accion, EstadoRef: r.EstadoPrincipal.Referencia,
		EstadoHuella:   r.EstadoPrincipal.HuellaEstadoSHA256,
		EstadoRevision: r.EstadoPrincipal.Revision, Identidad: identidad,
		Decision: struct {
			EsquemaHuella, DecisionRef, HuellaDecision, Accion, RecursoRef   string
			ModuloID, TipoRecurso, ContextoHuella, Finalidad                 string
			AsignacionRef, AsignacionHuella, VersionRolRef, VersionRolHuella string
			ControlRolRef, ControlRolHuella, CatalogoHuella                  string
			ControlRolRevision, RevisionCatalogo                             uint64
			EmitidaEn, VerificadaEn, ValidaHasta                             string
			Atestacion                                                       struct {
				DecisionRef, AtestacionRef, Estado, Huella, VerificadorRef string
				Version                                                    uint32
				VerificadaEn                                               string
			}
		}{
			d.EsquemaHuella, d.DecisionRef, d.HuellaDecisionSHA256, d.Accion, d.RecursoRef,
			d.ModuloID, d.TipoRecurso, d.ContextoRecursoHuellaSHA256, d.Finalidad,
			d.AsignacionRef, d.AsignacionHuellaSHA256, d.VersionRolRef, d.VersionRolHuellaSHA256,
			d.ControlVigenciaVersionRolRef, d.ControlVigenciaVersionRolHuellaSHA256,
			d.CatalogoPoliticasHuellaSHA256, d.ControlVigenciaVersionRolRevision,
			d.RevisionCatalogoPoliticas, instanteReciboCanonico(d.EmitidaEn),
			instanteReciboCanonico(d.VerificadaEn), instanteReciboCanonico(d.ValidaHasta),
			struct {
				DecisionRef, AtestacionRef, Estado, Huella, VerificadorRef string
				Version                                                    uint32
				VerificadaEn                                               string
			}{a.DecisionRef, a.AtestacionRef, a.EstadoAtestacion, a.HuellaAtestacionSHA256,
				a.VerificadorRef, a.VersionAtestacion, instanteReciboCanonico(a.VerificadaEn)},
		},
		Sellado: struct {
			Accion, ConvocatoriaRef, AtestacionRef, Estado, Huella string
			TokenConsumoRef, MaterializadorRef                     string
			Version                                                uint32
			EmitidaEn, ValidaHasta                                 string
			HMAC                                                   struct {
				Dominio, ClaveRef, Valor string
				Generacion               uint32
			}
		}{
			s.Accion, s.ConvocatoriaRef, s.AtestacionRef, s.EstadoAtestacion,
			s.HuellaAtestacionSHA256, s.TokenConsumoRef, s.MaterializadorRef,
			s.VersionAtestacion, instanteReciboCanonico(s.AtestacionEmitidaEn),
			instanteReciboCanonico(s.AtestacionValidaHasta),
			struct {
				Dominio, ClaveRef, Valor string
				Generacion               uint32
			}{h.DominioCriptografico, h.ClaveHMACRef, h.ValorHMACSHA256, h.GeneracionClave},
		},
		Revision: r.RevisionConfirmada, Cercado: r.CercadoConfirmado,
		ArrendamientoIniciaEn: instanteReciboCanonico(r.ArrendamientoIniciaEn),
		ArrendamientoVenceEn:  instanteReciboCanonico(r.ArrendamientoVenceEn),
		AuditoriaRef:          r.AuditoriaRef, HuellaAuditoria: r.HuellaAuditoriaSHA256,
		EventoRef: r.EventoOutboxRef, HuellaEvento: r.HuellaEventoOutboxSHA256,
		ConfirmadaEn: instanteReciboCanonico(r.ConfirmadaEn),
		Procedencia: struct {
			Esquema, Perfil, Autoridad, ProveedorRef string
			Migrable                                 bool
		}{r.Procedencia.Esquema, r.Procedencia.PerfilEjecucion, r.Procedencia.Autoridad,
			r.Procedencia.ProveedorRef, r.Procedencia.MigrableProduccion},
	})
}

func reciboProyectadoValido(r ProyeccionReciboBorrador, identidad ProyeccionIdentidadOperacion) bool {
	return cuerpoReciboBorradorValido(r, identidad) && r.AcreditacionKMS.validaParaRecibo(r)
}

func cuerpoReciboBorradorValido(
	r ProyeccionReciboBorrador,
	identidad ProyeccionIdentidadOperacion,
) bool {
	return r.Esquema == esquemaReciboBorradorV2 && referenciaProyeccionValida(r.ReciboRef) &&
		referenciaProyeccionValida(r.TransaccionRef) &&
		(r.Accion == puertosbolsa.AccionCrearBorradorConvocatoria ||
			r.Accion == puertosbolsa.AccionActualizarBorradorConvocatoria) &&
		r.EstadoPrincipal.Validar() == nil && identidadesProyectadasCoinciden(r.IdentidadPrimaria, identidad) &&
		r.Decision.valida() && r.Decision.Accion == r.Accion &&
		r.Decision.RecursoRef == r.EstadoPrincipal.Referencia &&
		r.SelladoMotivo.validaEstructural() && r.SelladoMotivo.Accion == r.Accion &&
		r.SelladoMotivo.ConvocatoriaRef == r.EstadoPrincipal.Referencia &&
		r.RevisionConfirmada > 0 && r.CercadoConfirmado > 0 &&
		instanteOperacionCanonico(r.ArrendamientoIniciaEn) && instanteOperacionCanonico(r.ArrendamientoVenceEn) &&
		r.ArrendamientoVenceEn.After(r.ArrendamientoIniciaEn) &&
		referenciaProyeccionValida(r.AuditoriaRef) && huellaHexValida(r.HuellaAuditoriaSHA256) &&
		referenciaProyeccionValida(r.EventoOutboxRef) && huellaHexValida(r.HuellaEventoOutboxSHA256) &&
		r.Procedencia.valida() &&
		r.TransaccionRef != r.AuditoriaRef && r.TransaccionRef != r.EventoOutboxRef &&
		r.AuditoriaRef != r.EventoOutboxRef && instanteOperacionCanonico(r.ConfirmadaEn) &&
		!r.ConfirmadaEn.Before(r.ArrendamientoIniciaEn) && r.ConfirmadaEn.Before(r.ArrendamientoVenceEn) &&
		!r.ConfirmadaEn.Before(r.Decision.VerificadaEn) && r.ConfirmadaEn.Before(r.Decision.ValidaHasta) &&
		!r.ConfirmadaEn.Before(r.SelladoMotivo.AtestacionEmitidaEn) &&
		r.ConfirmadaEn.Before(r.SelladoMotivo.AtestacionValidaHasta)
}

func instanteReciboCanonico(instante time.Time) string {
	return instante.UTC().Format(time.RFC3339Nano)
}
