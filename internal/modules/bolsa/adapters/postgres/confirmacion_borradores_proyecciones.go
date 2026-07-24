package postgres

import (
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

func proyectarPerfilCifradoBorradorPostgreSQL(
	p gobiernoconvocatorias.PerfilCifradoBorrador,
) perfilCifradoReciboPostgreSQL {
	return perfilCifradoReciboPostgreSQL{
		Referencia: p.Referencia, Version: p.Version, HuellaContenidoSHA256: p.HuellaContenidoSHA256,
		AlgoritmoAEAD: p.AlgoritmoAEAD, AlgoritmoEnvolturaClave: p.AlgoritmoEnvolturaClave,
	}
}

func proyectarProcedenciaBorradorPostgreSQL(
	p gobiernoconvocatorias.ProcedenciaActoBorrador,
) procedenciaReciboPostgreSQL {
	return procedenciaReciboPostgreSQL{
		Esquema: p.Esquema, PerfilEjecucion: p.PerfilEjecucion, Autoridad: p.Autoridad,
		ProveedorRef: p.ProveedorRef, MigrableProduccion: p.MigrableProduccion,
	}
}

func proyectarFirmaEvidenciaBorradorPostgreSQL(
	f gobiernoconvocatorias.FirmaEvidenciaBorrador,
) (firmaEvidenciaReciboPostgreSQL, error) {
	firma, err := f.FirmaBase64URLParaPersistencia()
	if err != nil {
		return firmaEvidenciaReciboPostgreSQL{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return firmaEvidenciaReciboPostgreSQL{
		AlgoritmoFirma: f.AlgoritmoFirma, VerificadorRef: f.VerificadorRef,
		HuellaClavePublicaSHA256: f.HuellaClavePublicaSHA256,
		HuellaPreimagenSHA256:    f.HuellaPreimagenSHA256, FirmaBase64URL: firma,
	}, nil
}

func proyectarSelladoMotivoBorradorPostgreSQL(
	s gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador,
) selladoMotivoReciboPostgreSQL {
	return selladoMotivoReciboPostgreSQL{
		Accion: s.Accion, ConvocatoriaRef: s.ConvocatoriaRef,
		HMAC: hmacMotivoReciboPostgreSQL{
			DominioCriptografico: s.HMAC.DominioCriptografico,
			GeneracionClave:      s.HMAC.GeneracionClave, ClaveHMACRef: s.HMAC.ClaveHMACRef,
			ValorHMACSHA256: s.HMAC.ValorHMACSHA256,
		},
		AtestacionRef: s.AtestacionRef, VersionAtestacion: s.VersionAtestacion,
		EstadoAtestacion: s.EstadoAtestacion, HuellaAtestacionSHA256: s.HuellaAtestacionSHA256,
		TokenConsumoRef: s.TokenConsumoRef, MaterializadorRef: s.MaterializadorRef,
		AtestacionEmitidaEn:   instanteMicrosegundoPostgreSQL(s.AtestacionEmitidaEn),
		AtestacionValidaHasta: instanteMicrosegundoPostgreSQL(s.AtestacionValidaHasta),
	}
}

func proyectarPoliticaCifradoBorradorPostgreSQL(
	p gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador,
	perfil perfilCifradoReciboPostgreSQL,
	identidad identidadDiarioPostgreSQL,
) politicaCifradoBorradorPostgreSQL {
	return politicaCifradoBorradorPostgreSQL{
		Esquema: p.Esquema, DecisionPoliticaRef: p.DecisionPoliticaRef,
		VersionDecisionPolitica: p.VersionDecisionPolitica, Estado: p.Estado,
		CatalogoRef: p.CatalogoRef, RevisionCatalogo: p.RevisionCatalogo,
		HuellaCatalogoSHA256: p.HuellaCatalogoSHA256, Accion: p.Accion,
		HuellaMaterialSHA256: p.HuellaMaterialSHA256, Perfil: perfil,
		IdentidadPrimaria: identidad, Revision: p.Revision, Cercado: p.Cercado,
		ArrendamientoIniciaEn: instanteMicrosegundoPostgreSQL(p.ArrendamientoIniciaEn),
		ArrendamientoVenceEn:  instanteMicrosegundoPostgreSQL(p.ArrendamientoVenceEn),
		SolicitadaEn:          instanteMicrosegundoPostgreSQL(p.SolicitadaEn),
		EmitidaEn:             instanteMicrosegundoPostgreSQL(p.EmitidaEn),
		VerificadaEn:          instanteMicrosegundoPostgreSQL(p.VerificadaEn),
		ValidaHasta:           instanteMicrosegundoPostgreSQL(p.ValidaHasta),
		AutoridadRef:          p.AutoridadRef, HuellaDecisionSHA256: p.HuellaDecisionSHA256,
	}
}

func proyectarEvidenciaPerfilBorradorPostgreSQL(
	e gobiernoconvocatorias.EvidenciaPerfilCifradoBorrador,
	perfil perfilCifradoReciboPostgreSQL,
	identidad identidadDiarioPostgreSQL,
) evidenciaPerfilBorradorPostgreSQL {
	return evidenciaPerfilBorradorPostgreSQL{
		Esquema: e.Esquema, EvidenciaRef: e.EvidenciaRef, VersionEvidencia: e.VersionEvidencia,
		Estado: e.Estado, CatalogoRef: e.CatalogoRef, RevisionCatalogo: e.RevisionCatalogo,
		HuellaCatalogoSHA256:         e.HuellaCatalogoSHA256,
		DecisionPoliticaRef:          e.DecisionPoliticaRef,
		VersionDecisionPolitica:      e.VersionDecisionPolitica,
		HuellaDecisionPoliticaSHA256: e.HuellaDecisionPoliticaSHA256,
		Accion:                       e.Accion, HuellaMaterialSHA256: e.HuellaMaterialSHA256,
		Perfil: perfil, IdentidadPrimaria: identidad, Revision: e.Revision, Cercado: e.Cercado,
		ArrendamientoIniciaEn: instanteMicrosegundoPostgreSQL(e.ArrendamientoIniciaEn),
		ArrendamientoVenceEn:  instanteMicrosegundoPostgreSQL(e.ArrendamientoVenceEn),
		SolicitudResolucionEn: instanteMicrosegundoPostgreSQL(e.SolicitudResolucionEn),
		EmitidaEn:             instanteMicrosegundoPostgreSQL(e.EmitidaEn),
		VerificadaEn:          instanteMicrosegundoPostgreSQL(e.VerificadaEn),
		ValidaHasta:           instanteMicrosegundoPostgreSQL(e.ValidaHasta),
		VerificadorRef:        e.VerificadorRef, HuellaEvidenciaSHA256: e.HuellaEvidenciaSHA256,
	}
}

func proyectarAtestacionKMSBorradorPostgreSQL(
	a gobiernoconvocatorias.AtestacionKMSBorrador,
	perfil perfilCifradoReciboPostgreSQL,
	procedencia procedenciaReciboPostgreSQL,
	firma firmaEvidenciaReciboPostgreSQL,
) atestacionKMSBorradorPostgreSQL {
	return atestacionKMSBorradorPostgreSQL{
		Esquema: a.Esquema, AtestacionRef: a.AtestacionRef,
		VersionAtestacion: a.VersionAtestacion, Estado: a.Estado, Perfil: perfil,
		ClaveMaestraRef: a.ClaveMaestraRef, VersionClave: a.VersionClave,
		HuellaAAD: a.HuellaAAD, HuellaEnvolturaSHA256: a.HuellaEnvolturaSHA256,
		HuellaSobreSHA256: a.HuellaSobreSHA256, VerificadorRef: a.VerificadorRef,
		Procedencia: procedencia, Firma: firma,
		EmitidaEn:   instanteMicrosegundoPostgreSQL(a.EmitidaEn),
		ValidaHasta: instanteMicrosegundoPostgreSQL(a.ValidaHasta),
	}
}

func instanteMicrosegundoPostgreSQL(instante time.Time) string {
	return instante.UTC().Format(formatoInstanteMicrosegundo)
}
