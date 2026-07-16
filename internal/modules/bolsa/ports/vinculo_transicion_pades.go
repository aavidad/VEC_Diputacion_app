package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

const (
	esquemaVinculoTransicionPAdES = "vec.bolsa.vinculo_transicion_pades"
	versionVinculoTransicionPAdES = 1
	tipoVinculoRevisionSellada    = "revision_sellada"
	tipoVinculoRevisionLongeva    = "revision_longeva"
	prefijoVinculoRevisionSellada = "vinculo_revision_sellada:"
	prefijoVinculoRevisionLongeva = "vinculo_revision_longeva:"
)

// VinculoTransicionPAdES identifica un recibo canónico calculado por el
// núcleo. Su huella se incorpora al manifiesto HMAC para impedir que una
// evidencia válida se combine con una revisión PDF distinta.
type VinculoTransicionPAdES struct {
	Referencia   string
	HuellaSHA256 string
}

func (v VinculoTransicionPAdES) ValidarParaTipo(tipo string) error {
	prefijo := prefijoVinculoRevisionSellada
	if tipo == tipoVinculoRevisionLongeva {
		prefijo = prefijoVinculoRevisionLongeva
	} else if tipo != tipoVinculoRevisionSellada {
		return ErrSolicitudBaremacionInvalida
	}
	if !huellaSHA256Valida(v.HuellaSHA256) || v.Referencia != prefijo+v.HuellaSHA256 {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type artefactoCanonicoVinculoPAdES struct {
	ProcesoRef                       string `json:"proceso_ref"`
	SolicitudRef                     string `json:"solicitud_ref"`
	SujetoRef                        string `json:"sujeto_ref"`
	BaremacionMeritoRef              string `json:"baremacion_merito_ref"`
	DecisionRef                      string `json:"decision_ref"`
	VersionBaremacion                uint64 `json:"version_baremacion"`
	SesionFirmaRef                   string `json:"sesion_firma_ref"`
	EvidenciaFirmaInteractivaRef     string `json:"evidencia_firma_interactiva_ref"`
	HuellaEvidenciaInteractivaSHA256 string `json:"huella_evidencia_interactiva_sha256"`
	DocumentoFirmableRef             string `json:"documento_firmable_ref"`
	VersionDocumentoFirmable         string `json:"version_documento_firmable"`
	HuellaDocumentoFirmableSHA256    string `json:"huella_documento_firmable_sha256"`
	EvidenciaCustodiaRef             string `json:"evidencia_custodia_ref"`
	FirmaRef                         string `json:"firma_ref"`
	HuellaFirmaSHA256                string `json:"huella_firma_sha256"`
	DocumentoFirmadoRef              string `json:"documento_firmado_ref"`
	HuellaDocumentoSHA256            string `json:"huella_documento_sha256"`
	HuellaContenidoSHA256            string `json:"huella_contenido_sha256"`
	PoliticaFirmaRef                 string `json:"politica_firma_ref"`
	PoliticaFirmaVersion             int    `json:"politica_firma_version"`
	HuellaPoliticaFirmaSHA256        string `json:"huella_politica_firma_sha256"`
	FirmanteRef                      string `json:"firmante_ref"`
	PerfilFirmanteClave              string `json:"perfil_firmante_clave"`
	FirmadaEn                        string `json:"firmada_en"`
}

type atestacionCanonicaVinculoPAdES struct {
	Estado                                  string                             `json:"estado"`
	ValidacionRef                           string                             `json:"validacion_ref"`
	HuellaValidacionSHA256                  string                             `json:"huella_validacion_sha256"`
	DocumentoFirmadoRef                     string                             `json:"documento_firmado_ref"`
	HuellaDocumentoSHA256                   string                             `json:"huella_documento_sha256"`
	FirmanteVerificadoRef                   string                             `json:"firmante_verificado_ref"`
	PerfilVerificadoClave                   string                             `json:"perfil_verificado_clave"`
	PerfilFirmaVerificadoClave              string                             `json:"perfil_firma_verificado_clave"`
	SelloTiempoVerificadoRef                string                             `json:"sello_tiempo_verificado_ref"`
	HuellaSelloTiempoVerificadaSHA256       string                             `json:"huella_sello_tiempo_verificada_sha256"`
	AumentoLongevidadVerificadoRef          string                             `json:"aumento_longevidad_verificado_ref"`
	HuellaAumentoLongevidadVerificadaSHA256 string                             `json:"huella_aumento_longevidad_verificada_sha256"`
	Comprobaciones                          []comprobacionCanonicaVinculoPAdES `json:"comprobaciones"`
	ValidadaEn                              string                             `json:"validada_en"`
}

type comprobacionCanonicaVinculoPAdES struct {
	Clave                 string `json:"clave"`
	Estado                string `json:"estado"`
	EvidenciaRef          string `json:"evidencia_ref"`
	HuellaEvidenciaSHA256 string `json:"huella_evidencia_sha256"`
}

type materialVinculoRevisionSelladaPAdES struct {
	Esquema                          string                         `json:"esquema"`
	Version                          int                            `json:"version"`
	Tipo                             string                         `json:"tipo"`
	ArtefactoOrigen                  artefactoCanonicoVinculoPAdES  `json:"artefacto_origen"`
	ArtefactoDestino                 artefactoCanonicoVinculoPAdES  `json:"artefacto_destino"`
	SelloTiempoRef                   string                         `json:"sello_tiempo_ref"`
	HuellaSelloTiempoSHA256          string                         `json:"huella_sello_tiempo_sha256"`
	PoliticaSelloTiempoRef           string                         `json:"politica_sello_tiempo_ref"`
	PoliticaSelloTiempoVersion       int                            `json:"politica_sello_tiempo_version"`
	HuellaPoliticaSelloTiempoSHA256  string                         `json:"huella_politica_sello_tiempo_sha256"`
	ValidacionSelloRef               string                         `json:"validacion_sello_ref"`
	HuellaValidacionSelloSHA256      string                         `json:"huella_validacion_sello_sha256"`
	SelladoEn                        string                         `json:"sellado_en"`
	AtestacionRevisionPAdESBaselineT atestacionCanonicaVinculoPAdES `json:"atestacion_revision_pades_baseline_t"`
}

type materialVinculoRevisionLongevaPAdES struct {
	Esquema                            string                         `json:"esquema"`
	Version                            int                            `json:"version"`
	Tipo                               string                         `json:"tipo"`
	ArtefactoOrigen                    artefactoCanonicoVinculoPAdES  `json:"artefacto_origen"`
	ArtefactoDestino                   artefactoCanonicoVinculoPAdES  `json:"artefacto_destino"`
	EvidenciaAumentoRef                string                         `json:"evidencia_aumento_ref"`
	HuellaEvidenciaAumentoSHA256       string                         `json:"huella_evidencia_aumento_sha256"`
	NivelLongevidadClave               string                         `json:"nivel_longevidad_clave"`
	PoliticaLongevidadRef              string                         `json:"politica_longevidad_ref"`
	PoliticaLongevidadVersion          int                            `json:"politica_longevidad_version"`
	HuellaPoliticaLongevidadSHA256     string                         `json:"huella_politica_longevidad_sha256"`
	AumentadaEn                        string                         `json:"aumentada_en"`
	VinculoRevisionSelladaRef          string                         `json:"vinculo_revision_sellada_ref"`
	HuellaVinculoRevisionSelladaSHA256 string                         `json:"huella_vinculo_revision_sellada_sha256"`
	AtestacionRevisionPAdESBaselineLTA atestacionCanonicaVinculoPAdES `json:"atestacion_revision_pades_baseline_lta"`
}

// NuevoVinculoRevisionSelladaPAdES vincula la transición B→T con la
// atestación que prueba el token TSA embebido en esa misma revisión T.
func NuevoVinculoRevisionSelladaPAdES(
	sello SelloTiempoFirma,
	atestacion ValidacionFirmaServidor,
) (VinculoTransicionPAdES, error) {
	if sello.Validar() != nil || !atestacion.AptaParaDecision() ||
		atestacion.Artefacto != sello.ArtefactoSellado ||
		atestacion.PerfilFirmaVerificadoClave != PerfilFirmaPAdESBaselineT ||
		atestacion.SelloTiempoVerificadoRef != sello.SelloTiempoRef ||
		atestacion.HuellaSelloTiempoVerificadaSHA256 != sello.HuellaSelloTiempoSHA256 ||
		atestacion.AumentoLongevidadVerificadoRef != "" ||
		atestacion.HuellaAumentoLongevidadVerificadaSHA256 != "" ||
		atestacion.ValidadaEn.Before(sello.SelladoEn) {
		return VinculoTransicionPAdES{}, ErrSelloTiempoNoDisponible
	}
	material := materialVinculoRevisionSelladaPAdES{
		Esquema: esquemaVinculoTransicionPAdES, Version: versionVinculoTransicionPAdES,
		Tipo:             tipoVinculoRevisionSellada,
		ArtefactoOrigen:  artefactoCanonicoVinculo(sello.ArtefactoOrigen),
		ArtefactoDestino: artefactoCanonicoVinculo(sello.ArtefactoSellado),
		SelloTiempoRef:   sello.SelloTiempoRef, HuellaSelloTiempoSHA256: sello.HuellaSelloTiempoSHA256,
		PoliticaSelloTiempoRef:           sello.PoliticaSelloTiempoRef,
		PoliticaSelloTiempoVersion:       sello.PoliticaSelloTiempoVersion,
		HuellaPoliticaSelloTiempoSHA256:  sello.HuellaPoliticaSelloTiempoSHA256,
		ValidacionSelloRef:               sello.ValidacionSelloRef,
		HuellaValidacionSelloSHA256:      sello.HuellaValidacionSHA256,
		SelladoEn:                        instanteCanonicoVinculo(sello.SelladoEn),
		AtestacionRevisionPAdESBaselineT: atestacionCanonicaVinculo(atestacion),
	}
	return nuevoVinculoTransicionPAdES(prefijoVinculoRevisionSellada, material)
}

// ValidarRevisionSelladaPara recompone el recibo desde el material original;
// no acepta una pareja referencia/huella válida pero perteneciente a otra
// transición.
func (v VinculoTransicionPAdES) ValidarRevisionSelladaPara(
	sello SelloTiempoFirma,
	atestacion ValidacionFirmaServidor,
) error {
	esperado, err := NuevoVinculoRevisionSelladaPAdES(sello, atestacion)
	if err != nil || v != esperado {
		return ErrSelloTiempoNoDisponible
	}
	return nil
}

// NuevoVinculoRevisionLongevaPAdES vincula la transición T→LTA con la
// atestación que prueba tanto el sello como el aumento embebidos en LTA.
func NuevoVinculoRevisionLongevaPAdES(
	sello SelloTiempoFirma,
	atestacionT ValidacionFirmaServidor,
	aumento ResultadoAumentoFirma,
	atestacionLTA ValidacionFirmaServidor,
) (VinculoTransicionPAdES, error) {
	vinculoT, err := NuevoVinculoRevisionSelladaPAdES(sello, atestacionT)
	if err != nil || aumento.Validar() != nil || !atestacionLTA.AptaParaDecision() ||
		aumento.ArtefactoOrigen != sello.ArtefactoSellado || atestacionLTA.Artefacto != aumento.Artefacto ||
		atestacionLTA.PerfilFirmaVerificadoClave != PerfilFirmaPAdESBaselineLTA ||
		atestacionLTA.SelloTiempoVerificadoRef != sello.SelloTiempoRef ||
		atestacionLTA.HuellaSelloTiempoVerificadaSHA256 != sello.HuellaSelloTiempoSHA256 ||
		atestacionLTA.AumentoLongevidadVerificadoRef != aumento.EvidenciaAumentoRef ||
		atestacionLTA.HuellaAumentoLongevidadVerificadaSHA256 != aumento.HuellaEvidenciaSHA256 ||
		aumento.AumentadaEn.Before(atestacionT.ValidadaEn) ||
		atestacionLTA.ValidadaEn.Before(aumento.AumentadaEn) {
		return VinculoTransicionPAdES{}, ErrAumentoFirmaNoDisponible
	}
	material := materialVinculoRevisionLongevaPAdES{
		Esquema: esquemaVinculoTransicionPAdES, Version: versionVinculoTransicionPAdES,
		Tipo:                               tipoVinculoRevisionLongeva,
		ArtefactoOrigen:                    artefactoCanonicoVinculo(aumento.ArtefactoOrigen),
		ArtefactoDestino:                   artefactoCanonicoVinculo(aumento.Artefacto),
		EvidenciaAumentoRef:                aumento.EvidenciaAumentoRef,
		HuellaEvidenciaAumentoSHA256:       aumento.HuellaEvidenciaSHA256,
		NivelLongevidadClave:               aumento.NivelAlcanzadoClave,
		PoliticaLongevidadRef:              aumento.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:          aumento.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256:     aumento.HuellaPoliticaLongevidadSHA256,
		AumentadaEn:                        instanteCanonicoVinculo(aumento.AumentadaEn),
		VinculoRevisionSelladaRef:          vinculoT.Referencia,
		HuellaVinculoRevisionSelladaSHA256: vinculoT.HuellaSHA256,
		AtestacionRevisionPAdESBaselineLTA: atestacionCanonicaVinculo(atestacionLTA),
	}
	return nuevoVinculoTransicionPAdES(prefijoVinculoRevisionLongeva, material)
}

// ValidarRevisionLongevaPara recompone el recibo desde el material original
// y falla cerrado ante cualquier sustitución de revisión o evidencia.
func (v VinculoTransicionPAdES) ValidarRevisionLongevaPara(
	sello SelloTiempoFirma,
	atestacionT ValidacionFirmaServidor,
	aumento ResultadoAumentoFirma,
	atestacionLTA ValidacionFirmaServidor,
) error {
	esperado, err := NuevoVinculoRevisionLongevaPAdES(sello, atestacionT, aumento, atestacionLTA)
	if err != nil || v != esperado {
		return ErrAumentoFirmaNoDisponible
	}
	return nil
}

func nuevoVinculoTransicionPAdES(prefijo string, material any) (VinculoTransicionPAdES, error) {
	canonico, err := json.Marshal(material)
	if err != nil {
		return VinculoTransicionPAdES{}, ErrSolicitudBaremacionInvalida
	}
	suma := sha256.Sum256(canonico)
	huella := hex.EncodeToString(suma[:])
	vinculo := VinculoTransicionPAdES{Referencia: prefijo + huella, HuellaSHA256: huella}
	tipo := tipoVinculoRevisionSellada
	if prefijo == prefijoVinculoRevisionLongeva {
		tipo = tipoVinculoRevisionLongeva
	}
	return vinculo, vinculo.ValidarParaTipo(tipo)
}

func artefactoCanonicoVinculo(a ArtefactoFirma) artefactoCanonicoVinculoPAdES {
	return artefactoCanonicoVinculoPAdES{
		ProcesoRef: a.ProcesoRef, SolicitudRef: a.SolicitudRef, SujetoRef: a.SujetoRef,
		BaremacionMeritoRef: a.BaremacionMeritoRef, DecisionRef: a.DecisionRef,
		VersionBaremacion: a.VersionBaremacion, SesionFirmaRef: a.SesionFirmaRef,
		EvidenciaFirmaInteractivaRef:     a.EvidenciaFirmaInteractivaRef,
		HuellaEvidenciaInteractivaSHA256: a.HuellaEvidenciaInteractivaSHA256,
		DocumentoFirmableRef:             a.DocumentoFirmable.Referencia,
		VersionDocumentoFirmable:         a.DocumentoFirmable.Version,
		HuellaDocumentoFirmableSHA256:    a.HuellaDocumentoFirmableSHA256,
		EvidenciaCustodiaRef:             a.EvidenciaCustodiaRef, FirmaRef: a.FirmaRef,
		HuellaFirmaSHA256: a.HuellaFirmaSHA256, DocumentoFirmadoRef: a.DocumentoFirmadoRef,
		HuellaDocumentoSHA256: a.HuellaDocumentoSHA256, HuellaContenidoSHA256: a.HuellaContenidoSHA256,
		PoliticaFirmaRef: a.PoliticaFirmaRef, PoliticaFirmaVersion: a.PoliticaFirmaVersion,
		HuellaPoliticaFirmaSHA256: a.HuellaPoliticaFirmaSHA256, FirmanteRef: a.FirmanteRef,
		PerfilFirmanteClave: a.PerfilFirmanteClave, FirmadaEn: instanteCanonicoVinculo(a.FirmadaEn),
	}
}

func atestacionCanonicaVinculo(v ValidacionFirmaServidor) atestacionCanonicaVinculoPAdES {
	comprobaciones := make([]comprobacionCanonicaVinculoPAdES, len(v.Comprobaciones))
	for indice, comprobacion := range v.Comprobaciones {
		comprobaciones[indice] = comprobacionCanonicaVinculoPAdES{
			Clave: comprobacion.Clave, Estado: string(comprobacion.Estado),
			EvidenciaRef:          comprobacion.EvidenciaRef,
			HuellaEvidenciaSHA256: comprobacion.HuellaEvidenciaSHA256,
		}
	}
	sort.Slice(comprobaciones, func(i, j int) bool { return comprobaciones[i].Clave < comprobaciones[j].Clave })
	return atestacionCanonicaVinculoPAdES{
		Estado:        string(v.Estado),
		ValidacionRef: v.ValidacionRef, HuellaValidacionSHA256: v.HuellaValidacionSHA256,
		DocumentoFirmadoRef:   v.Artefacto.DocumentoFirmadoRef,
		HuellaDocumentoSHA256: v.Artefacto.HuellaDocumentoSHA256,
		FirmanteVerificadoRef: v.FirmanteVerificadoRef, PerfilVerificadoClave: v.PerfilVerificadoClave,
		PerfilFirmaVerificadoClave:              v.PerfilFirmaVerificadoClave,
		SelloTiempoVerificadoRef:                v.SelloTiempoVerificadoRef,
		HuellaSelloTiempoVerificadaSHA256:       v.HuellaSelloTiempoVerificadaSHA256,
		AumentoLongevidadVerificadoRef:          v.AumentoLongevidadVerificadoRef,
		HuellaAumentoLongevidadVerificadaSHA256: v.HuellaAumentoLongevidadVerificadaSHA256,
		Comprobaciones:                          comprobaciones,
		ValidadaEn:                              instanteCanonicoVinculo(v.ValidadaEn),
	}
}

func instanteCanonicoVinculo(instante time.Time) string {
	return instante.UTC().Format(time.RFC3339Nano)
}
