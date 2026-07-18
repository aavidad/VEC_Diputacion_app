package gobiernoconvocatorias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	esquemaPoliticaGobernadaCifradoBorradorV1    = "bolsa.convocatoria.borrador.politica-cifrado.v1"
	estadoPoliticaGobernadaCifradoVigente        = "vigente"
	esquemaEvidenciaPerfilCifradoBorradorV1      = "bolsa.convocatoria.borrador.evidencia-perfil-cifrado.v1"
	estadoEvidenciaPerfilCifradoVigente          = "vigente"
	esquemaAcreditacionKMSConfirmacionBorradorV1 = "bolsa.convocatoria.borrador.acreditacion-kms-confirmacion.v1"
	estadoAcreditacionKMSConfirmada              = "confirmada"
)

type representacionHMACCanonicaBorrador struct {
	VersionEsquema  uint16 `json:"version_esquema"`
	Dominio         string `json:"dominio"`
	ClaveRef        string `json:"clave_ref"`
	GeneracionClave uint32 `json:"generacion_clave"`
	ValorHMACSHA256 string `json:"valor_hmac_sha256"`
}

type representacionIdentidadCanonicaBorrador struct {
	Localizador     representacionHMACCanonicaBorrador `json:"localizador"`
	HuellaSolicitud representacionHMACCanonicaBorrador `json:"huella_solicitud"`
}

func representacionIdentidadCanonica(
	identidad ProyeccionIdentidadOperacion,
) (representacionIdentidadCanonicaBorrador, bool) {
	if !identidad.valida() {
		return representacionIdentidadCanonicaBorrador{}, false
	}
	convertir := func(p ProyeccionHMACDiario) representacionHMACCanonicaBorrador {
		return representacionHMACCanonicaBorrador{
			VersionEsquema: p.VersionEsquema, Dominio: p.Dominio, ClaveRef: p.ClaveRef,
			GeneracionClave: p.GeneracionClave, ValorHMACSHA256: p.ValorHMACSHA256,
		}
	}
	return representacionIdentidadCanonicaBorrador{
		Localizador:     convertir(identidad.Localizador),
		HuellaSolicitud: convertir(identidad.HuellaSolicitud),
	}, true
}

func huellaJSONCanonicaBorrador(valor any) string {
	representacion, err := json.Marshal(valor)
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(representacion)
	return hex.EncodeToString(suma[:])
}

// SolicitudSeleccionPoliticaCifradoBorrador se entrega a una autoridad de
// política independiente del resolvedor técnico del perfil. La separación es
// parte del TCB: un proveedor criptográfico no puede autoaprobar algoritmos.
type SolicitudSeleccionPoliticaCifradoBorrador struct {
	bloqueoSerializacionDiario
	Reserva       ProyeccionReservaDecision
	Control       ResultadoOperacionDiario
	Material      puertosbolsa.MaterialIntencionGobiernoConvocatoria
	SelladoMotivo ProyeccionSelladoMotivoBorrador
	SolicitadaEn  time.Time
}

func (s SolicitudSeleccionPoliticaCifradoBorrador) Validar() error {
	if !solicitudBasePerfilCifradoValida(
		s.Reserva, s.Control, s.Material, s.SelladoMotivo, s.SolicitadaEn,
	) {
		return ErrCifradoBorradorInvalido
	}
	return nil
}

type PoliticaGobernadaCifradoBorrador struct {
	bloqueoSerializacionDiario
	PerfilEsperado          PerfilCifradoBorrador
	Esquema                 string
	DecisionPoliticaRef     string
	VersionDecisionPolitica uint32
	Estado                  string
	CatalogoRef             string
	RevisionCatalogo        uint64
	HuellaCatalogoSHA256    string
	Accion                  string
	HuellaMaterialSHA256    string
	IdentidadPrimaria       ProyeccionIdentidadOperacion
	Revision                uint64
	Cercado                 uint64
	ArrendamientoIniciaEn   time.Time
	ArrendamientoVenceEn    time.Time
	SolicitadaEn            time.Time
	EmitidaEn               time.Time
	VerificadaEn            time.Time
	ValidaHasta             time.Time
	AutoridadRef            string
	HuellaDecisionSHA256    string
}

func NuevaPoliticaGobernadaCifradoBorrador(
	perfil PerfilCifradoBorrador,
	s SolicitudSeleccionPoliticaCifradoBorrador,
	decisionPoliticaRef string,
	versionDecisionPolitica uint32,
	catalogoRef string,
	revisionCatalogo uint64,
	huellaCatalogoSHA256, autoridadRef string,
	emitidaEn, verificadaEn, validaHasta time.Time,
) (PoliticaGobernadaCifradoBorrador, error) {
	huellaMaterial, err := s.Material.HuellaSHA256()
	p := PoliticaGobernadaCifradoBorrador{
		PerfilEsperado: perfil, Esquema: esquemaPoliticaGobernadaCifradoBorradorV1,
		DecisionPoliticaRef: decisionPoliticaRef, VersionDecisionPolitica: versionDecisionPolitica,
		Estado:      estadoPoliticaGobernadaCifradoVigente,
		CatalogoRef: catalogoRef, RevisionCatalogo: revisionCatalogo,
		HuellaCatalogoSHA256: huellaCatalogoSHA256,
		Accion:               s.Material.Accion, HuellaMaterialSHA256: huellaMaterial,
		IdentidadPrimaria: s.Reserva.IdentidadPrimaria,
		Revision:          s.Control.Revision, Cercado: s.Control.Cercado,
		ArrendamientoIniciaEn: s.Control.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:  s.Control.ArrendamientoVenceEn,
		SolicitadaEn:          s.SolicitadaEn, EmitidaEn: emitidaEn, VerificadaEn: verificadaEn,
		ValidaHasta: validaHasta, AutoridadRef: autoridadRef,
	}
	p.HuellaDecisionSHA256 = p.calcularHuella()
	if err != nil || !p.validaPara(s) {
		return PoliticaGobernadaCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	return p, nil
}

func (p PoliticaGobernadaCifradoBorrador) calcularHuella() string {
	identidad, valida := representacionIdentidadCanonica(p.IdentidadPrimaria)
	if !valida {
		return ""
	}
	perfil := p.PerfilEsperado
	return huellaJSONCanonicaBorrador(struct {
		Esquema, DecisionRef, Estado, CatalogoRef, HuellaCatalogo string
		Accion, HuellaMaterial, PerfilRef, HuellaPerfil           string
		AlgoritmoAEAD, AlgoritmoEnvoltura, AutoridadRef           string
		VersionDecision, PerfilVersion                            uint32
		RevisionCatalogo, Revision, Cercado                       uint64
		Identidad                                                 representacionIdentidadCanonicaBorrador
		ArrendamientoIniciaEn, ArrendamientoVenceEn               string
		SolicitadaEn, EmitidaEn, VerificadaEn, ValidaHasta        string
	}{
		p.Esquema, p.DecisionPoliticaRef, p.Estado, p.CatalogoRef, p.HuellaCatalogoSHA256,
		p.Accion, p.HuellaMaterialSHA256, perfil.Referencia, perfil.HuellaContenidoSHA256,
		perfil.AlgoritmoAEAD, perfil.AlgoritmoEnvolturaClave, p.AutoridadRef,
		p.VersionDecisionPolitica, perfil.Version, p.RevisionCatalogo, p.Revision, p.Cercado,
		identidad, p.ArrendamientoIniciaEn.Format(time.RFC3339Nano),
		p.ArrendamientoVenceEn.Format(time.RFC3339Nano), p.SolicitadaEn.Format(time.RFC3339Nano),
		p.EmitidaEn.Format(time.RFC3339Nano), p.VerificadaEn.Format(time.RFC3339Nano),
		p.ValidaHasta.Format(time.RFC3339Nano),
	})
}

func (p PoliticaGobernadaCifradoBorrador) validaPara(
	s SolicitudSeleccionPoliticaCifradoBorrador,
) bool {
	huellaMaterial, err := s.Material.HuellaSHA256()
	return err == nil && s.Validar() == nil && p.PerfilEsperado.valida() &&
		p.Esquema == esquemaPoliticaGobernadaCifradoBorradorV1 &&
		referenciaProyeccionValida(p.DecisionPoliticaRef) && p.VersionDecisionPolitica > 0 &&
		p.Estado == estadoPoliticaGobernadaCifradoVigente &&
		referenciaProyeccionValida(p.CatalogoRef) && p.RevisionCatalogo > 0 &&
		huellaHexValida(p.HuellaCatalogoSHA256) && p.Accion == s.Material.Accion &&
		coincideTextoConstante(p.HuellaMaterialSHA256, huellaMaterial) &&
		identidadesProyectadasCoinciden(p.IdentidadPrimaria, s.Reserva.IdentidadPrimaria) &&
		p.Revision == s.Control.Revision && p.Cercado == s.Control.Cercado &&
		p.ArrendamientoIniciaEn.Equal(s.Control.ArrendamientoIniciaEn) &&
		p.ArrendamientoVenceEn.Equal(s.Control.ArrendamientoVenceEn) &&
		p.SolicitadaEn.Equal(s.SolicitadaEn) && instanteOperacionCanonico(p.EmitidaEn) &&
		instanteOperacionCanonico(p.VerificadaEn) && instanteOperacionCanonico(p.ValidaHasta) &&
		!p.VerificadaEn.Before(p.EmitidaEn) && !p.VerificadaEn.Before(p.ArrendamientoIniciaEn) &&
		!p.SolicitadaEn.Before(p.VerificadaEn) && p.SolicitadaEn.Before(p.ValidaHasta) &&
		!p.ValidaHasta.After(p.ArrendamientoVenceEn) &&
		referenciaProyeccionValida(p.AutoridadRef) && p.AutoridadRef != p.DecisionPoliticaRef &&
		huellaHexValida(p.HuellaDecisionSHA256) &&
		coincideTextoConstante(p.HuellaDecisionSHA256, p.calcularHuella())
}

type ProveedorPoliticaGobernadaCifradoBorrador interface {
	DescriptorAutoridadBorrador
	SeleccionarPoliticaCifradoBorrador(
		context.Context,
		SolicitudSeleccionPoliticaCifradoBorrador,
	) (PoliticaGobernadaCifradoBorrador, error)
}

// EvidenciaPerfilCifradoBorrador registra qué entrada versionada del catálogo
// gobernado autorizó el perfil y la liga a la operación reservada exacta.
type EvidenciaPerfilCifradoBorrador struct {
	bloqueoSerializacionDiario
	Esquema                      string
	EvidenciaRef                 string
	VersionEvidencia             uint32
	Estado                       string
	CatalogoRef                  string
	RevisionCatalogo             uint64
	HuellaCatalogoSHA256         string
	DecisionPoliticaRef          string
	VersionDecisionPolitica      uint32
	HuellaDecisionPoliticaSHA256 string
	Accion                       string
	HuellaMaterialSHA256         string
	IdentidadPrimaria            ProyeccionIdentidadOperacion
	Revision                     uint64
	Cercado                      uint64
	ArrendamientoIniciaEn        time.Time
	ArrendamientoVenceEn         time.Time
	SolicitudResolucionEn        time.Time
	EmitidaEn                    time.Time
	VerificadaEn                 time.Time
	ValidaHasta                  time.Time
	VerificadorRef               string
	HuellaEvidenciaSHA256        string
}

// ResolucionPerfilCifradoBorrador evita que un perfil meramente sintáctico se
// convierta en política esperada: siempre viaja con evidencia gobernada.
type ResolucionPerfilCifradoBorrador struct {
	bloqueoSerializacionDiario
	Perfil    PerfilCifradoBorrador
	Evidencia EvidenciaPerfilCifradoBorrador
}

func NuevaResolucionPerfilCifradoBorrador(
	perfil PerfilCifradoBorrador,
	solicitud SolicitudResolucionPerfilCifradoBorrador,
	evidenciaRef string,
	versionEvidencia uint32,
	verificadorRef string,
	emitidaEn, verificadaEn, validaHasta time.Time,
) (ResolucionPerfilCifradoBorrador, error) {
	huellaMaterial, err := solicitud.Material.HuellaSHA256()
	r := ResolucionPerfilCifradoBorrador{
		Perfil: perfil,
		Evidencia: EvidenciaPerfilCifradoBorrador{
			Esquema:      esquemaEvidenciaPerfilCifradoBorradorV1,
			EvidenciaRef: evidenciaRef, VersionEvidencia: versionEvidencia,
			Estado:                       estadoEvidenciaPerfilCifradoVigente,
			CatalogoRef:                  solicitud.PoliticaEsperada.CatalogoRef,
			RevisionCatalogo:             solicitud.PoliticaEsperada.RevisionCatalogo,
			HuellaCatalogoSHA256:         solicitud.PoliticaEsperada.HuellaCatalogoSHA256,
			DecisionPoliticaRef:          solicitud.PoliticaEsperada.DecisionPoliticaRef,
			VersionDecisionPolitica:      solicitud.PoliticaEsperada.VersionDecisionPolitica,
			HuellaDecisionPoliticaSHA256: solicitud.PoliticaEsperada.HuellaDecisionSHA256,
			Accion:                       solicitud.Material.Accion, HuellaMaterialSHA256: huellaMaterial,
			IdentidadPrimaria: solicitud.Reserva.IdentidadPrimaria,
			Revision:          solicitud.Control.Revision, Cercado: solicitud.Control.Cercado,
			ArrendamientoIniciaEn: solicitud.Control.ArrendamientoIniciaEn,
			ArrendamientoVenceEn:  solicitud.Control.ArrendamientoVenceEn,
			SolicitudResolucionEn: solicitud.SolicitadaEn,
			EmitidaEn:             emitidaEn, VerificadaEn: verificadaEn, ValidaHasta: validaHasta,
			VerificadorRef: verificadorRef,
		},
	}
	r.Evidencia.HuellaEvidenciaSHA256 = r.calcularHuellaEvidencia()
	if err != nil || r.ValidarPara(solicitud) != nil {
		return ResolucionPerfilCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	return r, nil
}

func (r ResolucionPerfilCifradoBorrador) calcularHuellaEvidencia() string {
	identidad, valida := representacionIdentidadCanonica(r.Evidencia.IdentidadPrimaria)
	if !valida {
		return ""
	}
	p := r.Perfil
	e := r.Evidencia
	return huellaJSONCanonicaBorrador(struct {
		Esquema, EvidenciaRef, Estado, CatalogoRef, HuellaCatalogoSHA256 string
		DecisionPoliticaRef, HuellaDecisionPoliticaSHA256                string
		Accion, HuellaMaterialSHA256, PerfilRef, HuellaPerfilSHA256      string
		AlgoritmoAEAD, AlgoritmoEnvoltura, VerificadorRef                string
		VersionEvidencia, PerfilVersion, VersionDecisionPolitica         uint32
		RevisionCatalogo, Revision, Cercado                              uint64
		Identidad                                                        representacionIdentidadCanonicaBorrador
		ArrendamientoIniciaEn, ArrendamientoVenceEn                      string
		SolicitudResolucionEn, EmitidaEn, VerificadaEn, ValidaHasta      string
	}{
		e.Esquema, e.EvidenciaRef, e.Estado, e.CatalogoRef, e.HuellaCatalogoSHA256,
		e.DecisionPoliticaRef, e.HuellaDecisionPoliticaSHA256,
		e.Accion, e.HuellaMaterialSHA256, p.Referencia, p.HuellaContenidoSHA256,
		p.AlgoritmoAEAD, p.AlgoritmoEnvolturaClave, e.VerificadorRef,
		e.VersionEvidencia, p.Version, e.VersionDecisionPolitica,
		e.RevisionCatalogo, e.Revision, e.Cercado,
		identidad, e.ArrendamientoIniciaEn.Format(time.RFC3339Nano),
		e.ArrendamientoVenceEn.Format(time.RFC3339Nano),
		e.SolicitudResolucionEn.Format(time.RFC3339Nano), e.EmitidaEn.Format(time.RFC3339Nano),
		e.VerificadaEn.Format(time.RFC3339Nano), e.ValidaHasta.Format(time.RFC3339Nano),
	})
}

func (r ResolucionPerfilCifradoBorrador) validaVinculo(
	reserva ProyeccionReservaDecision,
	control ResultadoOperacionDiario,
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
) bool {
	huellaMaterial, err := material.HuellaSHA256()
	e := r.Evidencia
	return err == nil && r.Perfil.valida() && reserva.valida() && resultadoDiarioValido(control) &&
		control.Estado == ResultadoDiarioReservado &&
		e.Esquema == esquemaEvidenciaPerfilCifradoBorradorV1 &&
		referenciaProyeccionValida(e.EvidenciaRef) && e.VersionEvidencia > 0 &&
		e.Estado == estadoEvidenciaPerfilCifradoVigente &&
		referenciaProyeccionValida(e.CatalogoRef) && e.RevisionCatalogo > 0 &&
		huellaHexValida(e.HuellaCatalogoSHA256) &&
		referenciaProyeccionValida(e.DecisionPoliticaRef) && e.VersionDecisionPolitica > 0 &&
		huellaHexValida(e.HuellaDecisionPoliticaSHA256) &&
		e.Accion == material.Accion && e.Accion == reserva.Accion &&
		coincideTextoConstante(e.HuellaMaterialSHA256, huellaMaterial) &&
		identidadesProyectadasCoinciden(e.IdentidadPrimaria, reserva.IdentidadPrimaria) &&
		e.Revision == control.Revision && e.Cercado == control.Cercado &&
		e.ArrendamientoIniciaEn.Equal(control.ArrendamientoIniciaEn) &&
		e.ArrendamientoVenceEn.Equal(control.ArrendamientoVenceEn) &&
		instanteOperacionCanonico(e.SolicitudResolucionEn) &&
		instanteOperacionCanonico(e.EmitidaEn) && instanteOperacionCanonico(e.VerificadaEn) &&
		instanteOperacionCanonico(e.ValidaHasta) && !e.VerificadaEn.Before(e.EmitidaEn) &&
		!e.VerificadaEn.Before(e.ArrendamientoIniciaEn) &&
		!e.SolicitudResolucionEn.Before(e.VerificadaEn) && e.SolicitudResolucionEn.Before(e.ValidaHasta) &&
		e.ValidaHasta.After(e.VerificadaEn) && !e.ValidaHasta.After(e.ArrendamientoVenceEn) &&
		referenciaProyeccionValida(e.VerificadorRef) && e.EvidenciaRef != e.VerificadorRef &&
		huellaHexValida(e.HuellaEvidenciaSHA256) &&
		coincideTextoConstante(e.HuellaEvidenciaSHA256, r.calcularHuellaEvidencia())
}

func (r ResolucionPerfilCifradoBorrador) ValidarPara(
	s SolicitudResolucionPerfilCifradoBorrador,
) error {
	if s.Validar() != nil || !r.validaVinculo(s.Reserva, s.Control, s.Material) ||
		!r.Evidencia.SolicitudResolucionEn.Equal(s.SolicitadaEn) ||
		!perfilesCifradoCoinciden(r.Perfil, s.PoliticaEsperada.PerfilEsperado) ||
		r.Evidencia.CatalogoRef != s.PoliticaEsperada.CatalogoRef ||
		r.Evidencia.RevisionCatalogo != s.PoliticaEsperada.RevisionCatalogo ||
		!coincideTextoConstante(r.Evidencia.HuellaCatalogoSHA256, s.PoliticaEsperada.HuellaCatalogoSHA256) ||
		r.Evidencia.DecisionPoliticaRef != s.PoliticaEsperada.DecisionPoliticaRef ||
		r.Evidencia.VersionDecisionPolitica != s.PoliticaEsperada.VersionDecisionPolitica ||
		!coincideTextoConstante(
			r.Evidencia.HuellaDecisionPoliticaSHA256,
			s.PoliticaEsperada.HuellaDecisionSHA256,
		) || r.Evidencia.VerificadaEn.Before(s.PoliticaEsperada.VerificadaEn) ||
		r.Evidencia.ValidaHasta.After(s.PoliticaEsperada.ValidaHasta) {
		return ErrCifradoBorradorInvalido
	}
	return nil
}

// AcreditacionKMSConfirmacionBorrador es la evidencia durable del adaptador
// transaccional. Liga la revalidación autoritativa del KMS con el sobre, la
// reserva CAS y el recibo. El núcleo no puede demostrar el orden físico frente
// a COMMIT, pero nunca acepta un resultado confirmado sin esta acreditación.
type AcreditacionKMSConfirmacionBorrador struct {
	bloqueoSerializacionDiario
	Esquema                     string
	AcreditacionRef             string
	VersionAcreditacion         uint32
	Estado                      string
	AtestacionRef               string
	VersionAtestacion           uint32
	Perfil                      PerfilCifradoBorrador
	ClaveMaestraRef             string
	VersionClave                uint32
	HuellaAAD                   string
	HuellaEnvolturaSHA256       string
	HuellaSobreSHA256           string
	Procedencia                 ProcedenciaActoBorrador
	FirmaAtestacionKMS          FirmaEvidenciaBorrador
	IdentidadPrimaria           ProyeccionIdentidadOperacion
	RevisionReserva             uint64
	RevisionConfirmada          uint64
	Cercado                     uint64
	ArrendamientoIniciaEn       time.Time
	ArrendamientoVenceEn        time.Time
	ComprobacionKMSRef          string
	HuellaComprobacionKMSSHA256 string
	FirmaRevalidacionKMS        FirmaEvidenciaBorrador
	TransaccionRef              string
	ReciboRef                   string
	HuellaCuerpoReciboSHA256    string
	AtestacionEmitidaEn         time.Time
	AtestacionValidaHasta       time.Time
	ConfirmacionSolicitadaEn    time.Time
	RevalidacionSolicitadaEn    time.Time
	RevalidadaEn                time.Time
	ConfirmadaEn                time.Time
	VerificadorRef              string
	HuellaAcreditacionSHA256    string
}

func NuevaAcreditacionKMSConfirmacionBorrador(
	confirmacion SolicitudConfirmacionBorrador,
	solicitudRevalidacion SolicitudRevalidacionAtestacionKMSBorrador,
	revalidacion ResultadoRevalidacionAtestacionKMSBorrador,
	recibo ProyeccionReciboBorrador,
	acreditacionRef string,
	versionAcreditacion uint32,
	verificadorRef string,
) (AcreditacionKMSConfirmacionBorrador, error) {
	atestacion := confirmacion.Cifrado.AtestacionKMS
	a := AcreditacionKMSConfirmacionBorrador{
		Esquema:         esquemaAcreditacionKMSConfirmacionBorradorV1,
		AcreditacionRef: acreditacionRef, VersionAcreditacion: versionAcreditacion,
		Estado:        estadoAcreditacionKMSConfirmada,
		AtestacionRef: atestacion.AtestacionRef, VersionAtestacion: atestacion.VersionAtestacion,
		Perfil: atestacion.Perfil, ClaveMaestraRef: atestacion.ClaveMaestraRef,
		VersionClave: atestacion.VersionClave, HuellaAAD: atestacion.HuellaAAD,
		HuellaEnvolturaSHA256: atestacion.HuellaEnvolturaSHA256,
		HuellaSobreSHA256:     atestacion.HuellaSobreSHA256,
		Procedencia:           atestacion.Procedencia,
		FirmaAtestacionKMS:    atestacion.Firma,
		IdentidadPrimaria:     confirmacion.Reserva.IdentidadPrimaria,
		RevisionReserva:       confirmacion.Control.Revision, RevisionConfirmada: recibo.RevisionConfirmada,
		Cercado:                     confirmacion.Control.Cercado,
		ArrendamientoIniciaEn:       confirmacion.Control.ArrendamientoIniciaEn,
		ArrendamientoVenceEn:        confirmacion.Control.ArrendamientoVenceEn,
		ComprobacionKMSRef:          revalidacion.ComprobacionRef,
		HuellaComprobacionKMSSHA256: revalidacion.HuellaComprobacionSHA256,
		FirmaRevalidacionKMS:        revalidacion.Firma,
		TransaccionRef:              recibo.TransaccionRef, ReciboRef: recibo.ReciboRef,
		HuellaCuerpoReciboSHA256: huellaCuerpoReciboBorrador(recibo),
		AtestacionEmitidaEn:      atestacion.EmitidaEn, AtestacionValidaHasta: atestacion.ValidaHasta,
		ConfirmacionSolicitadaEn: confirmacion.SolicitadaEn,
		RevalidacionSolicitadaEn: solicitudRevalidacion.SolicitadaEn,
		RevalidadaEn:             revalidacion.ComprobadaEn, ConfirmadaEn: recibo.ConfirmadaEn,
		VerificadorRef: verificadorRef,
	}
	a.HuellaAcreditacionSHA256 = a.calcularHuella()
	if !a.validaPara(confirmacion, solicitudRevalidacion, revalidacion, recibo) {
		return AcreditacionKMSConfirmacionBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return a, nil
}

func (a AcreditacionKMSConfirmacionBorrador) calcularHuella() string {
	identidad, valida := representacionIdentidadCanonica(a.IdentidadPrimaria)
	if !valida {
		return ""
	}
	p := a.Perfil
	firmaAtestacion, _ := a.FirmaAtestacionKMS.FirmaBase64URLParaPersistencia()
	firmaRevalidacion, _ := a.FirmaRevalidacionKMS.FirmaBase64URLParaPersistencia()
	return huellaJSONCanonicaBorrador(struct {
		Esquema, AcreditacionRef, Estado, AtestacionRef, PerfilRef           string
		HuellaPerfilSHA256, AlgoritmoAEAD, AlgoritmoEnvoltura, ClaveRef      string
		HuellaAAD, HuellaEnvoltura, HuellaSobre, ComprobacionRef             string
		HuellaComprobacion, TransaccionRef, ReciboRef, HuellaCuerpoRecibo    string
		VerificadorRef                                                       string
		ProcedenciaEsquema, PerfilEjecucion, Autoridad, ProveedorProcedencia string
		AlgoritmoFirmaAtestacion, VerificadorAtestacion                      string
		HuellaClaveAtestacion, HuellaPreimagenAtestacion, FirmaAtestacion    string
		AlgoritmoFirmaRevalidacion, VerificadorRevalidacion                  string
		HuellaClaveRevalidacion, HuellaPreimagenRevalidacion                 string
		FirmaRevalidacion                                                    string
		VersionAcreditacion, VersionAtestacion, PerfilVersion, VersionClave  uint32
		RevisionReserva, RevisionConfirmada, Cercado                         uint64
		MigrableProduccion                                                   bool
		Identidad                                                            representacionIdentidadCanonicaBorrador
		ArrendamientoIniciaEn, ArrendamientoVenceEn                          string
		AtestacionEmitidaEn, AtestacionValidaHasta                           string
		ConfirmacionSolicitadaEn, RevalidacionSolicitadaEn                   string
		RevalidadaEn, ConfirmadaEn                                           string
	}{
		a.Esquema, a.AcreditacionRef, a.Estado, a.AtestacionRef, p.Referencia,
		p.HuellaContenidoSHA256, p.AlgoritmoAEAD, p.AlgoritmoEnvolturaClave, a.ClaveMaestraRef,
		a.HuellaAAD, a.HuellaEnvolturaSHA256, a.HuellaSobreSHA256, a.ComprobacionKMSRef,
		a.HuellaComprobacionKMSSHA256, a.TransaccionRef, a.ReciboRef,
		a.HuellaCuerpoReciboSHA256, a.VerificadorRef,
		a.Procedencia.Esquema, a.Procedencia.PerfilEjecucion, a.Procedencia.Autoridad,
		a.Procedencia.ProveedorRef,
		a.FirmaAtestacionKMS.AlgoritmoFirma, a.FirmaAtestacionKMS.VerificadorRef,
		a.FirmaAtestacionKMS.HuellaClavePublicaSHA256,
		a.FirmaAtestacionKMS.HuellaPreimagenSHA256, firmaAtestacion,
		a.FirmaRevalidacionKMS.AlgoritmoFirma, a.FirmaRevalidacionKMS.VerificadorRef,
		a.FirmaRevalidacionKMS.HuellaClavePublicaSHA256,
		a.FirmaRevalidacionKMS.HuellaPreimagenSHA256, firmaRevalidacion,
		a.VersionAcreditacion, a.VersionAtestacion, p.Version, a.VersionClave,
		a.RevisionReserva, a.RevisionConfirmada, a.Cercado,
		a.Procedencia.MigrableProduccion, identidad,
		a.ArrendamientoIniciaEn.Format(time.RFC3339Nano), a.ArrendamientoVenceEn.Format(time.RFC3339Nano),
		a.AtestacionEmitidaEn.Format(time.RFC3339Nano), a.AtestacionValidaHasta.Format(time.RFC3339Nano),
		a.ConfirmacionSolicitadaEn.Format(time.RFC3339Nano),
		a.RevalidacionSolicitadaEn.Format(time.RFC3339Nano),
		a.RevalidadaEn.Format(time.RFC3339Nano), a.ConfirmadaEn.Format(time.RFC3339Nano),
	})
}

func (a AcreditacionKMSConfirmacionBorrador) validaEstructural() bool {
	return a.Esquema == esquemaAcreditacionKMSConfirmacionBorradorV1 &&
		referenciaProyeccionValida(a.AcreditacionRef) && a.VersionAcreditacion > 0 &&
		a.Estado == estadoAcreditacionKMSConfirmada &&
		referenciaProyeccionValida(a.AtestacionRef) && a.VersionAtestacion > 0 &&
		a.Perfil.valida() && referenciaProyeccionValida(a.ClaveMaestraRef) && a.VersionClave > 0 &&
		huellaHexValida(a.HuellaAAD) && huellaHexValida(a.HuellaEnvolturaSHA256) &&
		huellaHexValida(a.HuellaSobreSHA256) && a.IdentidadPrimaria.valida() &&
		a.Procedencia.valida() && a.FirmaAtestacionKMS.formaValida() &&
		a.RevisionReserva > 0 && a.RevisionConfirmada > a.RevisionReserva && a.Cercado > 0 &&
		instanteOperacionCanonico(a.ArrendamientoIniciaEn) &&
		instanteOperacionCanonico(a.ArrendamientoVenceEn) &&
		a.ArrendamientoVenceEn.After(a.ArrendamientoIniciaEn) &&
		referenciaProyeccionValida(a.ComprobacionKMSRef) &&
		huellaHexValida(a.HuellaComprobacionKMSSHA256) &&
		a.FirmaRevalidacionKMS.formaValida() &&
		a.FirmaAtestacionKMS.VerificadorRef != a.FirmaRevalidacionKMS.VerificadorRef &&
		!coincideTextoConstante(
			a.FirmaAtestacionKMS.HuellaClavePublicaSHA256,
			a.FirmaRevalidacionKMS.HuellaClavePublicaSHA256,
		) &&
		referenciaProyeccionValida(a.TransaccionRef) && referenciaProyeccionValida(a.ReciboRef) &&
		huellaHexValida(a.HuellaCuerpoReciboSHA256) &&
		instanteOperacionCanonico(a.AtestacionEmitidaEn) &&
		instanteOperacionCanonico(a.AtestacionValidaHasta) &&
		instanteOperacionCanonico(a.ConfirmacionSolicitadaEn) &&
		instanteOperacionCanonico(a.RevalidacionSolicitadaEn) &&
		instanteOperacionCanonico(a.RevalidadaEn) && instanteOperacionCanonico(a.ConfirmadaEn) &&
		!a.RevalidacionSolicitadaEn.Before(a.ConfirmacionSolicitadaEn) &&
		!a.RevalidadaEn.Before(a.RevalidacionSolicitadaEn) &&
		!a.ConfirmadaEn.Before(a.RevalidadaEn) &&
		!a.ConfirmadaEn.Before(a.ArrendamientoIniciaEn) && a.ConfirmadaEn.Before(a.ArrendamientoVenceEn) &&
		!a.RevalidadaEn.Before(a.AtestacionEmitidaEn) && a.RevalidadaEn.Before(a.AtestacionValidaHasta) &&
		a.ConfirmadaEn.Before(a.AtestacionValidaHasta) &&
		referenciaProyeccionValida(a.VerificadorRef) &&
		a.AcreditacionRef != a.VerificadorRef && a.AcreditacionRef != a.AtestacionRef &&
		a.AcreditacionRef != a.TransaccionRef && a.AcreditacionRef != a.ReciboRef &&
		huellaHexValida(a.HuellaAcreditacionSHA256) &&
		coincideTextoConstante(a.HuellaAcreditacionSHA256, a.calcularHuella())
}

func (a AcreditacionKMSConfirmacionBorrador) validaParaRecibo(
	recibo ProyeccionReciboBorrador,
) bool {
	return a.validaEstructural() && a.ReciboRef == recibo.ReciboRef &&
		a.TransaccionRef == recibo.TransaccionRef &&
		procedenciasActoCoinciden(a.Procedencia, recibo.Procedencia) &&
		coincideTextoConstante(a.HuellaCuerpoReciboSHA256, huellaCuerpoReciboBorrador(recibo)) &&
		identidadesProyectadasCoinciden(a.IdentidadPrimaria, recibo.IdentidadPrimaria) &&
		a.RevisionConfirmada == recibo.RevisionConfirmada && a.Cercado == recibo.CercadoConfirmado &&
		a.ArrendamientoIniciaEn.Equal(recibo.ArrendamientoIniciaEn) &&
		a.ArrendamientoVenceEn.Equal(recibo.ArrendamientoVenceEn) &&
		a.ConfirmadaEn.Equal(recibo.ConfirmadaEn)
}

func (a AcreditacionKMSConfirmacionBorrador) validaPara(
	confirmacion SolicitudConfirmacionBorrador,
	solicitudRevalidacion SolicitudRevalidacionAtestacionKMSBorrador,
	revalidacion ResultadoRevalidacionAtestacionKMSBorrador,
	recibo ProyeccionReciboBorrador,
) bool {
	esperada, err := NuevaSolicitudRevalidacionAtestacionKMSBorrador(
		confirmacion, solicitudRevalidacion.HuellaCuerpoReciboSHA256,
		solicitudRevalidacion.SolicitadaEn,
	)
	atestacion := confirmacion.Cifrado.AtestacionKMS
	return err == nil && esperada == solicitudRevalidacion &&
		revalidacion.ValidarPara(solicitudRevalidacion) == nil && a.validaParaRecibo(recibo) &&
		a.AtestacionRef == atestacion.AtestacionRef &&
		a.VersionAtestacion == atestacion.VersionAtestacion &&
		procedenciasActoCoinciden(a.Procedencia, atestacion.Procedencia) &&
		a.FirmaAtestacionKMS == atestacion.Firma &&
		perfilesCifradoCoinciden(a.Perfil, atestacion.Perfil) &&
		a.ClaveMaestraRef == atestacion.ClaveMaestraRef && a.VersionClave == atestacion.VersionClave &&
		coincideTextoConstante(a.HuellaAAD, atestacion.HuellaAAD) &&
		coincideTextoConstante(a.HuellaEnvolturaSHA256, atestacion.HuellaEnvolturaSHA256) &&
		coincideTextoConstante(a.HuellaSobreSHA256, atestacion.HuellaSobreSHA256) &&
		identidadesProyectadasCoinciden(a.IdentidadPrimaria, confirmacion.Reserva.IdentidadPrimaria) &&
		a.RevisionReserva == confirmacion.Control.Revision &&
		a.Cercado == confirmacion.Control.Cercado &&
		a.ArrendamientoIniciaEn.Equal(confirmacion.Control.ArrendamientoIniciaEn) &&
		a.ArrendamientoVenceEn.Equal(confirmacion.Control.ArrendamientoVenceEn) &&
		a.ComprobacionKMSRef == revalidacion.ComprobacionRef &&
		coincideTextoConstante(
			a.HuellaCuerpoReciboSHA256,
			solicitudRevalidacion.HuellaCuerpoReciboSHA256,
		) &&
		a.FirmaRevalidacionKMS == revalidacion.Firma &&
		coincideTextoConstante(a.HuellaComprobacionKMSSHA256, revalidacion.HuellaComprobacionSHA256) &&
		a.AtestacionEmitidaEn.Equal(atestacion.EmitidaEn) &&
		a.AtestacionValidaHasta.Equal(atestacion.ValidaHasta) &&
		a.ConfirmacionSolicitadaEn.Equal(confirmacion.SolicitadaEn) &&
		a.RevalidacionSolicitadaEn.Equal(solicitudRevalidacion.SolicitadaEn) &&
		a.RevalidadaEn.Equal(revalidacion.ComprobadaEn)
}

func (a AcreditacionKMSConfirmacionBorrador) validaParaConfirmacion(
	confirmacion SolicitudConfirmacionBorrador,
	recibo ProyeccionReciboBorrador,
) bool {
	solicitud, err := NuevaSolicitudRevalidacionAtestacionKMSBorrador(
		confirmacion, a.HuellaCuerpoReciboSHA256, a.RevalidacionSolicitadaEn,
	)
	if err != nil {
		return false
	}
	revalidacion := ResultadoRevalidacionAtestacionKMSBorrador{
		AtestacionRef:     confirmacion.Cifrado.AtestacionKMS.AtestacionRef,
		VersionAtestacion: confirmacion.Cifrado.AtestacionKMS.VersionAtestacion,
		Estado:            estadoRevalidacionKMSAutorizada, HuellaAAD: a.HuellaAAD,
		ComprobacionRef:          a.ComprobacionKMSRef,
		HuellaComprobacionSHA256: a.HuellaComprobacionKMSSHA256,
		ComprobadaEn:             a.RevalidadaEn,
		Firma:                    a.FirmaRevalidacionKMS,
	}
	return a.validaPara(confirmacion, solicitud, revalidacion, recibo)
}

// EvidenciasKMSParaVerificacion valida primero el vínculo de la acreditación
// con el cuerpo completo del recibo y reconstruye las dos preimágenes KMS. No
// verifica criptografía: esa responsabilidad corresponde al verificador con
// credencial de solo lectura y claves públicas independientes del confirmador.
func (r ProyeccionReciboBorrador) EvidenciasKMSParaVerificacion() (
	AtestacionKMSBorrador,
	SolicitudRevalidacionAtestacionKMSBorrador,
	ResultadoRevalidacionAtestacionKMSBorrador,
	error,
) {
	if !reciboProyectadoValido(r, r.IdentidadPrimaria) {
		return AtestacionKMSBorrador{}, SolicitudRevalidacionAtestacionKMSBorrador{},
			ResultadoRevalidacionAtestacionKMSBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return r.AcreditacionKMS.evidenciasKMSParaVerificacion()
}

func (a AcreditacionKMSConfirmacionBorrador) evidenciasKMSParaVerificacion() (
	AtestacionKMSBorrador,
	SolicitudRevalidacionAtestacionKMSBorrador,
	ResultadoRevalidacionAtestacionKMSBorrador,
	error,
) {
	if !a.validaEstructural() {
		return AtestacionKMSBorrador{}, SolicitudRevalidacionAtestacionKMSBorrador{},
			ResultadoRevalidacionAtestacionKMSBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	atestacion := AtestacionKMSBorrador{
		Esquema: esquemaAtestacionKMSBorradorV1, AtestacionRef: a.AtestacionRef,
		VersionAtestacion: a.VersionAtestacion, Estado: estadoAtestacionKMSVigente,
		Perfil: a.Perfil, ClaveMaestraRef: a.ClaveMaestraRef, VersionClave: a.VersionClave,
		HuellaAAD: a.HuellaAAD, HuellaEnvolturaSHA256: a.HuellaEnvolturaSHA256,
		HuellaSobreSHA256: a.HuellaSobreSHA256,
		VerificadorRef:    a.FirmaAtestacionKMS.VerificadorRef,
		Procedencia:       a.Procedencia, Firma: a.FirmaAtestacionKMS,
		EmitidaEn: a.AtestacionEmitidaEn, ValidaHasta: a.AtestacionValidaHasta,
	}
	solicitud := SolicitudRevalidacionAtestacionKMSBorrador{
		AtestacionKMS: atestacion, IdentidadPrimaria: a.IdentidadPrimaria,
		HuellaAAD: a.HuellaAAD, Revision: a.RevisionReserva, Cercado: a.Cercado,
		HuellaCuerpoReciboSHA256: a.HuellaCuerpoReciboSHA256,
		ArrendamientoVenceEn:     a.ArrendamientoVenceEn,
		ConfirmacionSolicitadaEn: a.ConfirmacionSolicitadaEn,
		SolicitadaEn:             a.RevalidacionSolicitadaEn,
	}
	revalidacion := ResultadoRevalidacionAtestacionKMSBorrador{
		AtestacionRef: a.AtestacionRef, VersionAtestacion: a.VersionAtestacion,
		Estado: estadoRevalidacionKMSAutorizada, HuellaAAD: a.HuellaAAD,
		ComprobacionRef:          a.ComprobacionKMSRef,
		HuellaComprobacionSHA256: a.HuellaComprobacionKMSSHA256,
		ComprobadaEn:             a.RevalidadaEn, Firma: a.FirmaRevalidacionKMS,
	}
	if !atestacion.validaEstructural() || solicitud.Validar() != nil ||
		revalidacion.ValidarPara(solicitud) != nil {
		return AtestacionKMSBorrador{}, SolicitudRevalidacionAtestacionKMSBorrador{},
			ResultadoRevalidacionAtestacionKMSBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return atestacion, solicitud, revalidacion, nil
}
