package postgres

import (
	"encoding/json"
	"errors"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

func serializarAcreditacionKMSBorradorPostgreSQL(
	a gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador,
) ([]byte, error) {
	persistida, err := proyectarAcreditacionKMSBorradorPostgreSQL(a)
	if err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(persistida)
	if err != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return contenido, nil
}

func proyectarAcreditacionKMSBorradorPostgreSQL(
	a gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador,
) (acreditacionKMSReciboPostgreSQL, error) {
	identidad, errIdentidad := proyectarIdentidadDiarioPostgreSQL(a.IdentidadPrimaria)
	firmaAtestacion, errAtestacion := proyectarFirmaEvidenciaBorradorPostgreSQL(a.FirmaAtestacionKMS)
	firmaRevalidacion, errRevalidacion := proyectarFirmaEvidenciaBorradorPostgreSQL(a.FirmaRevalidacionKMS)
	if errors.Join(errIdentidad, errAtestacion, errRevalidacion) != nil {
		return acreditacionKMSReciboPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return acreditacionKMSReciboPostgreSQL{
		Esquema: a.Esquema, AcreditacionRef: a.AcreditacionRef,
		VersionAcreditacion: a.VersionAcreditacion, Estado: a.Estado,
		AtestacionRef: a.AtestacionRef, VersionAtestacion: a.VersionAtestacion,
		Perfil:          proyectarPerfilCifradoBorradorPostgreSQL(a.Perfil),
		ClaveMaestraRef: a.ClaveMaestraRef, VersionClave: a.VersionClave,
		HuellaAAD: a.HuellaAAD, HuellaEnvolturaSHA256: a.HuellaEnvolturaSHA256,
		HuellaSobreSHA256:  a.HuellaSobreSHA256,
		Procedencia:        proyectarProcedenciaBorradorPostgreSQL(a.Procedencia),
		FirmaAtestacionKMS: firmaAtestacion, IdentidadPrimaria: identidad,
		RevisionReserva: a.RevisionReserva, RevisionConfirmada: a.RevisionConfirmada,
		Cercado:                     a.Cercado,
		ArrendamientoIniciaEn:       instanteMicrosegundoPostgreSQL(a.ArrendamientoIniciaEn),
		ArrendamientoVenceEn:        instanteMicrosegundoPostgreSQL(a.ArrendamientoVenceEn),
		ComprobacionKMSRef:          a.ComprobacionKMSRef,
		HuellaComprobacionKMSSHA256: a.HuellaComprobacionKMSSHA256,
		FirmaRevalidacionKMS:        firmaRevalidacion,
		TransaccionRef:              a.TransaccionRef, ReciboRef: a.ReciboRef,
		HuellaCuerpoReciboSHA256: a.HuellaCuerpoReciboSHA256,
		AtestacionEmitidaEn:      instanteMicrosegundoPostgreSQL(a.AtestacionEmitidaEn),
		AtestacionValidaHasta:    instanteMicrosegundoPostgreSQL(a.AtestacionValidaHasta),
		ConfirmacionSolicitadaEn: instanteMicrosegundoPostgreSQL(a.ConfirmacionSolicitadaEn),
		RevalidacionSolicitadaEn: instanteMicrosegundoPostgreSQL(a.RevalidacionSolicitadaEn),
		RevalidadaEn:             instanteMicrosegundoPostgreSQL(a.RevalidadaEn),
		ConfirmadaEn:             instanteMicrosegundoPostgreSQL(a.ConfirmadaEn),
		VerificadorRef:           a.VerificadorRef,
		HuellaAcreditacionSHA256: a.HuellaAcreditacionSHA256,
	}, nil
}

func serializarReciboBorradorPostgreSQL(
	r gobiernoconvocatorias.ProyeccionReciboBorrador,
) ([]byte, error) {
	if _, _, _, err := r.EvidenciasKMSParaVerificacion(); err != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	identidad, errIdentidad := proyectarIdentidadDiarioPostgreSQL(r.IdentidadPrimaria)
	acreditacion, errAcreditacion := proyectarAcreditacionKMSBorradorPostgreSQL(r.AcreditacionKMS)
	if errors.Join(errIdentidad, errAcreditacion) != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	contenido, err := json.Marshal(reciboBorradorPostgreSQL{
		Esquema: r.Esquema, ReciboRef: r.ReciboRef, TransaccionRef: r.TransaccionRef,
		Accion: r.Accion, EstadoPrincipal: r.EstadoPrincipal, Identidad: identidad,
		Decision:           proyectarDecisionDiarioPostgreSQL(r.Decision),
		SelladoMotivo:      proyectarSelladoMotivoBorradorPostgreSQL(r.SelladoMotivo),
		RevisionConfirmada: r.RevisionConfirmada, CercadoConfirmado: r.CercadoConfirmado,
		ArrendamientoIniciaEn: instanteMicrosegundoPostgreSQL(r.ArrendamientoIniciaEn),
		ArrendamientoVenceEn:  instanteMicrosegundoPostgreSQL(r.ArrendamientoVenceEn),
		AuditoriaRef:          r.AuditoriaRef, HuellaAuditoriaSHA256: r.HuellaAuditoriaSHA256,
		EventoOutboxRef:          r.EventoOutboxRef,
		HuellaEventoOutboxSHA256: r.HuellaEventoOutboxSHA256,
		Procedencia:              proyectarProcedenciaBorradorPostgreSQL(r.Procedencia),
		AcreditacionKMS:          acreditacion, ConfirmadaEn: instanteMicrosegundoPostgreSQL(r.ConfirmadaEn),
	})
	if err != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return contenido, nil
}

// restaurarCuerpoReciboBorradorPostgreSQL recompone la proyección provisional
// de fase A. Aún no admite acreditación: solo puede usarse para calcular la
// preimagen que la fase B ligará al recibo final.
func restaurarCuerpoReciboBorradorPostgreSQL(
	contenido []byte,
) (gobiernoconvocatorias.ProyeccionReciboBorrador, error) {
	var persistido reciboBorradorPostgreSQL
	if err := decodificarJSONCerradoDiarioPostgreSQL(contenido, &persistido); err != nil {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, err
	}
	identidad, errIdentidad := restaurarIdentidadDiarioPostgreSQL(persistido.Identidad)
	decision, errDecision := restaurarDecisionDiarioPostgreSQL(persistido.Decision)
	sellado, errSellado := restaurarSelladoMotivoReciboPostgreSQL(persistido.SelladoMotivo)
	procedencia, errProcedencia := restaurarProcedenciaReciboPostgreSQL(persistido.Procedencia)
	inicio, errInicio := instanteJSONDiarioPostgreSQL(persistido.ArrendamientoIniciaEn)
	vence, errVence := instanteJSONDiarioPostgreSQL(persistido.ArrendamientoVenceEn)
	confirmada, errConfirmada := instanteJSONDiarioPostgreSQL(persistido.ConfirmadaEn)
	if errors.Join(
		errIdentidad, errDecision, errSellado, errProcedencia, errInicio, errVence, errConfirmada,
	) != nil {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	recibo := gobiernoconvocatorias.ProyeccionReciboBorrador{
		Esquema: persistido.Esquema, ReciboRef: persistido.ReciboRef,
		TransaccionRef: persistido.TransaccionRef, Accion: persistido.Accion,
		EstadoPrincipal: persistido.EstadoPrincipal, IdentidadPrimaria: identidad,
		Decision: decision, SelladoMotivo: sellado,
		RevisionConfirmada:    persistido.RevisionConfirmada,
		CercadoConfirmado:     persistido.CercadoConfirmado,
		ArrendamientoIniciaEn: inicio, ArrendamientoVenceEn: vence,
		AuditoriaRef:             persistido.AuditoriaRef,
		HuellaAuditoriaSHA256:    persistido.HuellaAuditoriaSHA256,
		EventoOutboxRef:          persistido.EventoOutboxRef,
		HuellaEventoOutboxSHA256: persistido.HuellaEventoOutboxSHA256,
		Procedencia:              procedencia, ConfirmadaEn: confirmada,
	}
	if _, err := recibo.HuellaCuerpoParaRevalidacion(); err != nil {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return recibo, nil
}
