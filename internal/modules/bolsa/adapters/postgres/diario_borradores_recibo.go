package postgres

import (
	"errors"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type hmacMotivoReciboPostgreSQL struct {
	DominioCriptografico string `json:"dominio_criptografico"`
	GeneracionClave      uint32 `json:"generacion_clave"`
	ClaveHMACRef         string `json:"clave_hmac_ref"`
	ValorHMACSHA256      string `json:"valor_hmac_sha256"`
}

type selladoMotivoReciboPostgreSQL struct {
	Accion                 string                     `json:"accion"`
	ConvocatoriaRef        string                     `json:"convocatoria_ref"`
	HMAC                   hmacMotivoReciboPostgreSQL `json:"hmac"`
	AtestacionRef          string                     `json:"atestacion_ref"`
	VersionAtestacion      uint32                     `json:"version_atestacion"`
	EstadoAtestacion       string                     `json:"estado_atestacion"`
	HuellaAtestacionSHA256 string                     `json:"huella_atestacion_sha256"`
	TokenConsumoRef        string                     `json:"token_consumo_ref"`
	MaterializadorRef      string                     `json:"materializador_ref"`
	AtestacionEmitidaEn    string                     `json:"atestacion_emitida_en"`
	AtestacionValidaHasta  string                     `json:"atestacion_valida_hasta"`
}

type procedenciaReciboPostgreSQL struct {
	Esquema            string `json:"esquema"`
	PerfilEjecucion    string `json:"perfil_ejecucion"`
	Autoridad          string `json:"autoridad"`
	ProveedorRef       string `json:"proveedor_ref"`
	MigrableProduccion bool   `json:"migrable_produccion"`
}

type perfilCifradoReciboPostgreSQL struct {
	Referencia              string `json:"referencia"`
	Version                 uint32 `json:"version"`
	HuellaContenidoSHA256   string `json:"huella_contenido_sha256"`
	AlgoritmoAEAD           string `json:"algoritmo_aead"`
	AlgoritmoEnvolturaClave string `json:"algoritmo_envoltura_clave"`
}

type firmaEvidenciaReciboPostgreSQL struct {
	AlgoritmoFirma           string `json:"algoritmo_firma"`
	VerificadorRef           string `json:"verificador_ref"`
	HuellaClavePublicaSHA256 string `json:"huella_clave_publica_sha256"`
	HuellaPreimagenSHA256    string `json:"huella_preimagen_sha256"`
	FirmaBase64URL           string `json:"firma_base64url_sin_relleno"`
}

type acreditacionKMSReciboPostgreSQL struct {
	Esquema                     string                         `json:"esquema"`
	AcreditacionRef             string                         `json:"acreditacion_ref"`
	VersionAcreditacion         uint32                         `json:"version_acreditacion"`
	Estado                      string                         `json:"estado"`
	AtestacionRef               string                         `json:"atestacion_ref"`
	VersionAtestacion           uint32                         `json:"version_atestacion"`
	Perfil                      perfilCifradoReciboPostgreSQL  `json:"perfil"`
	ClaveMaestraRef             string                         `json:"clave_maestra_ref"`
	VersionClave                uint32                         `json:"version_clave"`
	HuellaAAD                   string                         `json:"huella_aad"`
	HuellaEnvolturaSHA256       string                         `json:"huella_envoltura_sha256"`
	HuellaSobreSHA256           string                         `json:"huella_sobre_sha256"`
	Procedencia                 procedenciaReciboPostgreSQL    `json:"procedencia"`
	FirmaAtestacionKMS          firmaEvidenciaReciboPostgreSQL `json:"firma_atestacion_kms"`
	IdentidadPrimaria           identidadDiarioPostgreSQL      `json:"identidad_primaria"`
	RevisionReserva             uint64                         `json:"revision_reserva"`
	RevisionConfirmada          uint64                         `json:"revision_confirmada"`
	Cercado                     uint64                         `json:"cercado"`
	ArrendamientoIniciaEn       string                         `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn        string                         `json:"arrendamiento_vence_en"`
	ComprobacionKMSRef          string                         `json:"comprobacion_kms_ref"`
	HuellaComprobacionKMSSHA256 string                         `json:"huella_comprobacion_kms_sha256"`
	FirmaRevalidacionKMS        firmaEvidenciaReciboPostgreSQL `json:"firma_revalidacion_kms"`
	TransaccionRef              string                         `json:"transaccion_ref"`
	ReciboRef                   string                         `json:"recibo_ref"`
	HuellaCuerpoReciboSHA256    string                         `json:"huella_cuerpo_recibo_sha256"`
	AtestacionEmitidaEn         string                         `json:"atestacion_emitida_en"`
	AtestacionValidaHasta       string                         `json:"atestacion_valida_hasta"`
	ConfirmacionSolicitadaEn    string                         `json:"confirmacion_solicitada_en"`
	RevalidacionSolicitadaEn    string                         `json:"revalidacion_solicitada_en"`
	RevalidadaEn                string                         `json:"revalidada_en"`
	ConfirmadaEn                string                         `json:"confirmada_en"`
	VerificadorRef              string                         `json:"verificador_ref"`
	HuellaAcreditacionSHA256    string                         `json:"huella_acreditacion_sha256"`
}

type reciboBorradorPostgreSQL struct {
	Esquema                  string                                           `json:"esquema"`
	ReciboRef                string                                           `json:"recibo_ref"`
	TransaccionRef           string                                           `json:"transaccion_ref"`
	Accion                   string                                           `json:"accion"`
	EstadoPrincipal          puertosbolsa.ReferenciaEstadoVersionConvocatoria `json:"estado_principal"`
	Identidad                identidadDiarioPostgreSQL                        `json:"identidad"`
	Decision                 decisionDiarioPostgreSQL                         `json:"decision"`
	SelladoMotivo            selladoMotivoReciboPostgreSQL                    `json:"sellado_motivo"`
	RevisionConfirmada       uint64                                           `json:"revision_confirmada"`
	CercadoConfirmado        uint64                                           `json:"cercado_confirmado"`
	ArrendamientoIniciaEn    string                                           `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn     string                                           `json:"arrendamiento_vence_en"`
	AuditoriaRef             string                                           `json:"auditoria_ref"`
	HuellaAuditoriaSHA256    string                                           `json:"huella_auditoria_sha256"`
	EventoOutboxRef          string                                           `json:"evento_outbox_ref"`
	HuellaEventoOutboxSHA256 string                                           `json:"huella_evento_outbox_sha256"`
	Procedencia              procedenciaReciboPostgreSQL                      `json:"procedencia"`
	AcreditacionKMS          acreditacionKMSReciboPostgreSQL                  `json:"acreditacion_kms"`
	ConfirmadaEn             string                                           `json:"confirmada_en"`
}

func restaurarReciboBorradorPostgreSQL(
	contenido []byte,
) (*gobiernoconvocatorias.ProyeccionReciboBorrador, error) {
	if len(contenido) == 0 {
		return nil, nil
	}
	var persistido reciboBorradorPostgreSQL
	if err := decodificarJSONCerradoDiarioPostgreSQL(contenido, &persistido); err != nil {
		return nil, err
	}
	identidad, errIdentidad := restaurarIdentidadDiarioPostgreSQL(persistido.Identidad)
	decision, errDecision := restaurarDecisionDiarioPostgreSQL(persistido.Decision)
	sellado, errSellado := restaurarSelladoMotivoReciboPostgreSQL(persistido.SelladoMotivo)
	procedencia, errProcedencia := restaurarProcedenciaReciboPostgreSQL(persistido.Procedencia)
	acreditacion, errAcreditacion := restaurarAcreditacionKMSReciboPostgreSQL(persistido.AcreditacionKMS)
	inicio, errInicio := instanteJSONDiarioPostgreSQL(persistido.ArrendamientoIniciaEn)
	vence, errVence := instanteJSONDiarioPostgreSQL(persistido.ArrendamientoVenceEn)
	confirmada, errConfirmada := instanteJSONDiarioPostgreSQL(persistido.ConfirmadaEn)
	if errors.Join(
		errIdentidad, errDecision, errSellado, errProcedencia, errAcreditacion,
		errInicio, errVence, errConfirmada,
	) != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
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
		Procedencia:              procedencia, AcreditacionKMS: acreditacion, ConfirmadaEn: confirmada,
	}
	// Esta llamada no atribuye autoridad criptografica. Obliga, antes de sacar
	// el recibo del adaptador, a recomponer y validar cuerpo, acreditacion, dos
	// preimagenes, firmas persistidas, lease, fence y procedencia exactos.
	if _, _, _, err := recibo.EvidenciasKMSParaVerificacion(); err != nil {
		return nil, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return &recibo, nil
}

func restaurarSelladoMotivoReciboPostgreSQL(
	persistido selladoMotivoReciboPostgreSQL,
) (gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador, error) {
	emitida, errEmitida := instanteJSONDiarioPostgreSQL(persistido.AtestacionEmitidaEn)
	validaHasta, errValida := instanteJSONDiarioPostgreSQL(persistido.AtestacionValidaHasta)
	if errors.Join(errEmitida, errValida) != nil {
		return gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	hmac := puertosbolsa.ProyeccionHMACMotivoGobiernoConvocatoriaDurable{
		DominioCriptografico: persistido.HMAC.DominioCriptografico,
		GeneracionClave:      persistido.HMAC.GeneracionClave,
		ClaveHMACRef:         persistido.HMAC.ClaveHMACRef,
		ValorHMACSHA256:      persistido.HMAC.ValorHMACSHA256,
	}
	if hmac.Validar() != nil {
		return gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador{
		Accion: persistido.Accion, ConvocatoriaRef: persistido.ConvocatoriaRef, HMAC: hmac,
		AtestacionRef: persistido.AtestacionRef, VersionAtestacion: persistido.VersionAtestacion,
		EstadoAtestacion:       persistido.EstadoAtestacion,
		HuellaAtestacionSHA256: persistido.HuellaAtestacionSHA256,
		TokenConsumoRef:        persistido.TokenConsumoRef, MaterializadorRef: persistido.MaterializadorRef,
		AtestacionEmitidaEn: emitida, AtestacionValidaHasta: validaHasta,
	}, nil
}

func restaurarProcedenciaReciboPostgreSQL(
	persistida procedenciaReciboPostgreSQL,
) (gobiernoconvocatorias.ProcedenciaActoBorrador, error) {
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		persistida.PerfilEjecucion, persistida.Autoridad, persistida.ProveedorRef,
		persistida.MigrableProduccion,
	)
	if err != nil || procedencia.Esquema != persistida.Esquema {
		return gobiernoconvocatorias.ProcedenciaActoBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return procedencia, nil
}

func restaurarAcreditacionKMSReciboPostgreSQL(
	persistida acreditacionKMSReciboPostgreSQL,
) (gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador, error) {
	perfil, errPerfil := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(
		persistida.Perfil.Referencia, persistida.Perfil.Version,
		persistida.Perfil.HuellaContenidoSHA256, persistida.Perfil.AlgoritmoAEAD,
		persistida.Perfil.AlgoritmoEnvolturaClave,
	)
	procedencia, errProcedencia := restaurarProcedenciaReciboPostgreSQL(persistida.Procedencia)
	firmaAtestacion, errFirmaAtestacion := restaurarFirmaEvidenciaReciboPostgreSQL(persistida.FirmaAtestacionKMS)
	firmaRevalidacion, errFirmaRevalidacion := restaurarFirmaEvidenciaReciboPostgreSQL(persistida.FirmaRevalidacionKMS)
	identidad, errIdentidad := restaurarIdentidadDiarioPostgreSQL(persistida.IdentidadPrimaria)
	inicio, errInicio := instanteJSONDiarioPostgreSQL(persistida.ArrendamientoIniciaEn)
	vence, errVence := instanteJSONDiarioPostgreSQL(persistida.ArrendamientoVenceEn)
	emitida, errEmitida := instanteJSONDiarioPostgreSQL(persistida.AtestacionEmitidaEn)
	validaHasta, errValida := instanteJSONDiarioPostgreSQL(persistida.AtestacionValidaHasta)
	confirmacionSolicitada, errConfirmacion := instanteJSONDiarioPostgreSQL(persistida.ConfirmacionSolicitadaEn)
	revalidacionSolicitada, errSolicitud := instanteJSONDiarioPostgreSQL(persistida.RevalidacionSolicitadaEn)
	revalidada, errRevalidada := instanteJSONDiarioPostgreSQL(persistida.RevalidadaEn)
	confirmada, errConfirmada := instanteJSONDiarioPostgreSQL(persistida.ConfirmadaEn)
	if errors.Join(
		errPerfil, errProcedencia, errFirmaAtestacion, errFirmaRevalidacion, errIdentidad,
		errInicio, errVence, errEmitida, errValida, errConfirmacion, errSolicitud,
		errRevalidada, errConfirmada,
	) != nil {
		return gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador{
		Esquema: persistida.Esquema, AcreditacionRef: persistida.AcreditacionRef,
		VersionAcreditacion: persistida.VersionAcreditacion, Estado: persistida.Estado,
		AtestacionRef: persistida.AtestacionRef, VersionAtestacion: persistida.VersionAtestacion,
		Perfil: perfil, ClaveMaestraRef: persistida.ClaveMaestraRef,
		VersionClave: persistida.VersionClave, HuellaAAD: persistida.HuellaAAD,
		HuellaEnvolturaSHA256: persistida.HuellaEnvolturaSHA256,
		HuellaSobreSHA256:     persistida.HuellaSobreSHA256, Procedencia: procedencia,
		FirmaAtestacionKMS: firmaAtestacion, IdentidadPrimaria: identidad,
		RevisionReserva:    persistida.RevisionReserva,
		RevisionConfirmada: persistida.RevisionConfirmada, Cercado: persistida.Cercado,
		ArrendamientoIniciaEn: inicio, ArrendamientoVenceEn: vence,
		ComprobacionKMSRef:          persistida.ComprobacionKMSRef,
		HuellaComprobacionKMSSHA256: persistida.HuellaComprobacionKMSSHA256,
		FirmaRevalidacionKMS:        firmaRevalidacion, TransaccionRef: persistida.TransaccionRef,
		ReciboRef: persistida.ReciboRef, HuellaCuerpoReciboSHA256: persistida.HuellaCuerpoReciboSHA256,
		AtestacionEmitidaEn: emitida, AtestacionValidaHasta: validaHasta,
		ConfirmacionSolicitadaEn: confirmacionSolicitada,
		RevalidacionSolicitadaEn: revalidacionSolicitada,
		RevalidadaEn:             revalidada, ConfirmadaEn: confirmada,
		VerificadorRef:           persistida.VerificadorRef,
		HuellaAcreditacionSHA256: persistida.HuellaAcreditacionSHA256,
	}, nil
}

func restaurarFirmaEvidenciaReciboPostgreSQL(
	persistida firmaEvidenciaReciboPostgreSQL,
) (gobiernoconvocatorias.FirmaEvidenciaBorrador, error) {
	firma, err := gobiernoconvocatorias.RestaurarFirmaEvidenciaBorradorPersistida(
		persistida.AlgoritmoFirma, persistida.VerificadorRef,
		persistida.HuellaClavePublicaSHA256, persistida.HuellaPreimagenSHA256,
		persistida.FirmaBase64URL,
	)
	if err != nil {
		return gobiernoconvocatorias.FirmaEvidenciaBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return firma, nil
}
