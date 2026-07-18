package gobiernoconvocatorias

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var ErrDescifradoBorradorInvalido = errors.New("gobierno convocatorias: descifrado de borrador invalido")

// materialAADCanonicaCifradoBorrador fija tanto el orden JSON como todos los
// campos que 000004 debe persistir literalmente. La AAD no se reconstruye a
// partir de columnas parciales: se releen estos bytes exactos y su SHA-256.
type materialAADCanonicaCifradoBorrador struct {
	Esquema                   string `json:"esquema"`
	VersionRef                string `json:"version_ref"`
	VersionRevision           int    `json:"version_revision"`
	HuellaVersionSHA256       string `json:"huella_version_sha256"`
	EsquemaMaterial           string `json:"esquema_material"`
	HuellaMaterialSHA256      string `json:"huella_material_sha256"`
	PerfilCifradoRef          string `json:"perfil_cifrado_ref"`
	PerfilCifradoVersion      uint32 `json:"perfil_cifrado_version"`
	HuellaPerfilSHA256        string `json:"huella_perfil_cifrado_sha256"`
	AlgoritmoAEAD             string `json:"algoritmo_aead"`
	AlgoritmoEnvolturaClave   string `json:"algoritmo_envoltura_clave"`
	EvidenciaPerfilRef        string `json:"evidencia_perfil_ref"`
	EvidenciaPerfilVersion    uint32 `json:"evidencia_perfil_version"`
	HuellaEvidenciaPerfil     string `json:"huella_evidencia_perfil_sha256"`
	DecisionPoliticaRef       string `json:"decision_politica_ref"`
	DecisionPoliticaVersion   uint32 `json:"decision_politica_version"`
	HuellaDecisionPolitica    string `json:"huella_decision_politica_sha256"`
	LocalizadorEsquema        uint16 `json:"localizador_esquema"`
	LocalizadorDominio        string `json:"localizador_dominio"`
	LocalizadorClaveRef       string `json:"localizador_clave_ref"`
	LocalizadorGeneracion     uint32 `json:"localizador_generacion"`
	LocalizadorHMACSHA256     string `json:"localizador_hmac_sha256"`
	HuellaSolicitudEsquema    uint16 `json:"huella_solicitud_esquema"`
	HuellaSolicitudDominio    string `json:"huella_solicitud_dominio"`
	HuellaSolicitudClaveRef   string `json:"huella_solicitud_clave_ref"`
	HuellaSolicitudGeneracion uint32 `json:"huella_solicitud_generacion"`
	HuellaSolicitudHMACSHA256 string `json:"huella_solicitud_hmac_sha256"`
	RevisionDiario            uint64 `json:"revision_diario"`
	CercadoDiario             uint64 `json:"cercado_diario"`
	ArrendamientoIniciaEn     string `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn      string `json:"arrendamiento_vence_en"`
	AtestacionSelladoRef      string `json:"atestacion_sellado_ref"`
	AtestacionSelladoVersion  uint32 `json:"atestacion_sellado_version"`
	HuellaAtestacionSellado   string `json:"huella_atestacion_sellado_sha256"`
	TokenConsumoSelladoRef    string `json:"token_consumo_sellado_ref"`
	HuellaCorrelacionSHA256   string `json:"huella_correlacion_sha256"`
	ProcedenciaEsquema        string `json:"procedencia_esquema"`
	PerfilEjecucion           string `json:"perfil_ejecucion"`
	AutoridadActo             string `json:"autoridad_acto"`
	ProveedorProcedenciaRef   string `json:"proveedor_procedencia_ref"`
	MigrableProduccion        bool   `json:"migrable_produccion"`
}

func decodificarMaterialAADCanonicaBorrador(
	representacion []byte,
) (materialAADCanonicaCifradoBorrador, bool) {
	if len(representacion) == 0 || len(representacion) > 32<<10 || !utf8.Valid(representacion) {
		return materialAADCanonicaCifradoBorrador{}, false
	}
	var material materialAADCanonicaCifradoBorrador
	decodificador := json.NewDecoder(bytes.NewReader(representacion))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&material); err != nil {
		return materialAADCanonicaCifradoBorrador{}, false
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return materialAADCanonicaCifradoBorrador{}, false
	}
	canonica, err := json.Marshal(material)
	if err != nil || !bytes.Equal(canonica, representacion) || !material.valida() {
		return materialAADCanonicaCifradoBorrador{}, false
	}
	return material, true
}

func (m materialAADCanonicaCifradoBorrador) valida() bool {
	inicia, errInicia := time.Parse(time.RFC3339Nano, m.ArrendamientoIniciaEn)
	vence, errVence := time.Parse(time.RFC3339Nano, m.ArrendamientoVenceEn)
	perfil := PerfilCifradoBorrador{
		Referencia: m.PerfilCifradoRef, Version: m.PerfilCifradoVersion,
		HuellaContenidoSHA256: m.HuellaPerfilSHA256, AlgoritmoAEAD: m.AlgoritmoAEAD,
		AlgoritmoEnvolturaClave: m.AlgoritmoEnvolturaClave,
	}
	procedencia := ProcedenciaActoBorrador{
		Esquema: m.ProcedenciaEsquema, PerfilEjecucion: m.PerfilEjecucion,
		Autoridad: m.AutoridadActo, ProveedorRef: m.ProveedorProcedenciaRef,
		MigrableProduccion: m.MigrableProduccion,
	}
	return errInicia == nil && errVence == nil && m.Esquema == esquemaAADCifradoBorradorV1 &&
		referenciaProyeccionValida(m.VersionRef) && m.VersionRevision > 0 &&
		huellaHexValida(m.HuellaVersionSHA256) &&
		m.EsquemaMaterial == puertosbolsa.EsquemaMaterialIntencionGobiernoConvocatoriaV2 &&
		huellaHexValida(m.HuellaMaterialSHA256) && perfil.valida() &&
		referenciaProyeccionValida(m.EvidenciaPerfilRef) && m.EvidenciaPerfilVersion > 0 &&
		huellaHexValida(m.HuellaEvidenciaPerfil) && referenciaProyeccionValida(m.DecisionPoliticaRef) &&
		m.DecisionPoliticaVersion > 0 && huellaHexValida(m.HuellaDecisionPolitica) &&
		m.LocalizadorEsquema > 0 && m.LocalizadorDominio == "localizador" &&
		referenciaProyeccionValida(m.LocalizadorClaveRef) && m.LocalizadorGeneracion > 0 &&
		huellaHexValida(m.LocalizadorHMACSHA256) && m.HuellaSolicitudEsquema > 0 &&
		m.HuellaSolicitudDominio == "huella_solicitud" &&
		referenciaProyeccionValida(m.HuellaSolicitudClaveRef) && m.HuellaSolicitudGeneracion > 0 &&
		huellaHexValida(m.HuellaSolicitudHMACSHA256) && m.RevisionDiario > 0 && m.CercadoDiario > 0 &&
		instanteOperacionCanonico(inicia) && instanteOperacionCanonico(vence) && vence.After(inicia) &&
		m.ArrendamientoIniciaEn == inicia.Format(time.RFC3339Nano) &&
		m.ArrendamientoVenceEn == vence.Format(time.RFC3339Nano) &&
		referenciaProyeccionValida(m.AtestacionSelladoRef) && m.AtestacionSelladoVersion > 0 &&
		huellaHexValida(m.HuellaAtestacionSellado) && referenciaProyeccionValida(m.TokenConsumoSelladoRef) &&
		huellaHexValida(m.HuellaCorrelacionSHA256) && procedencia.valida()
}

// RestaurarAADCanonicaCifradoBorrador solo acepta la representación exacta
// generada al cifrar. Rechaza JSON equivalente, duplicados, campos extra,
// orden distinto y una huella recalculada sobre una forma no canónica.
func RestaurarAADCanonicaCifradoBorrador(
	representacion []byte,
	huellaSHA256 string,
) (AADCanonicaCifradoBorrador, error) {
	resultado := AADCanonicaCifradoBorrador{
		representacion: append([]byte(nil), representacion...), huellaSHA256: huellaSHA256,
	}
	if !resultado.valida() {
		return AADCanonicaCifradoBorrador{}, ErrDescifradoBorradorInvalido
	}
	return resultado, nil
}

func (e EnvolturaClaveKMSBorrador) EsquemaParaPersistencia() (string, error) {
	if !e.valida() {
		return "", ErrCifradoBorradorInvalido
	}
	return e.esquema, nil
}

func RestaurarEnvolturaClaveKMSBorrador(
	esquema string,
	perfil PerfilCifradoBorrador,
	claveMaestraRef string,
	versionClave uint32,
	materialEnvuelto []byte,
	huellaAAD, huellaSHA256 string,
) (EnvolturaClaveKMSBorrador, error) {
	envoltura, err := NuevaEnvolturaClaveKMSBorrador(
		perfil, claveMaestraRef, versionClave, materialEnvuelto, huellaAAD,
	)
	if err != nil || esquema != esquemaEnvolturaClaveBorradorV1 ||
		!coincideTextoConstante(envoltura.huellaSHA256, huellaSHA256) {
		return EnvolturaClaveKMSBorrador{}, ErrDescifradoBorradorInvalido
	}
	return envoltura, nil
}

func (s SobreCifradoAEADBorrador) EsquemaParaPersistencia() (string, error) {
	if !s.valida() {
		return "", ErrCifradoBorradorInvalido
	}
	return s.esquema, nil
}

func RestaurarSobreCifradoAEADBorrador(
	esquema string,
	perfil PerfilCifradoBorrador,
	nonce, textoCifrado []byte,
	huellaAAD, huellaSHA256 string,
) (SobreCifradoAEADBorrador, error) {
	sobre, err := NuevoSobreCifradoAEADBorrador(perfil, nonce, textoCifrado, huellaAAD)
	if err != nil || esquema != esquemaSobreCifradoBorradorV1 ||
		!coincideTextoConstante(sobre.huellaSHA256, huellaSHA256) {
		return SobreCifradoAEADBorrador{}, ErrDescifradoBorradorInvalido
	}
	return sobre, nil
}

// SolicitudDescifradoBorradorDurable es el límite nominal que construye el
// lector PostgreSQL con una fila cerrada de 000004. No admite un sobre aislado:
// referencia, AAD, perfil, KMS y procedencia deben formar una única prueba.
type SolicitudDescifradoBorradorDurable struct {
	bloqueoSerializacionDiario
	EstadoEsperado puertosbolsa.ReferenciaEstadoVersionConvocatoria
	AAD            AADCanonicaCifradoBorrador
	PerfilEsperado PerfilCifradoBorrador
	EnvolturaClave EnvolturaClaveKMSBorrador
	SobreCifrado   SobreCifradoAEADBorrador
	AtestacionKMS  AtestacionKMSBorrador
	Procedencia    ProcedenciaActoBorrador
}

func NuevaSolicitudDescifradoBorradorDurable(
	estadoEsperado puertosbolsa.ReferenciaEstadoVersionConvocatoria,
	aad AADCanonicaCifradoBorrador,
	perfil PerfilCifradoBorrador,
	envoltura EnvolturaClaveKMSBorrador,
	sobre SobreCifradoAEADBorrador,
	atestacion AtestacionKMSBorrador,
	procedencia ProcedenciaActoBorrador,
) (SolicitudDescifradoBorradorDurable, error) {
	solicitud := SolicitudDescifradoBorradorDurable{
		EstadoEsperado: estadoEsperado, AAD: aad, PerfilEsperado: perfil,
		EnvolturaClave: envoltura, SobreCifrado: sobre,
		AtestacionKMS: atestacion, Procedencia: procedencia,
	}
	if solicitud.Validar() != nil {
		return SolicitudDescifradoBorradorDurable{}, ErrDescifradoBorradorInvalido
	}
	return solicitud, nil
}

func (s SolicitudDescifradoBorradorDurable) Validar() error {
	materialAAD, aadValida := decodificarMaterialAADCanonicaBorrador(s.AAD.representacion)
	huellaAAD, errAAD := s.AAD.HuellaSHA256()
	perfilEnvoltura := s.EnvolturaClave.perfil
	claveRef := s.EnvolturaClave.claveMaestraRef
	versionClave := s.EnvolturaClave.versionClave
	huellaAADEnvoltura := s.EnvolturaClave.huellaAAD
	huellaEnvoltura := s.EnvolturaClave.huellaSHA256
	perfilSobre := s.SobreCifrado.perfil
	huellaAADSobre := s.SobreCifrado.huellaAAD
	huellaSobre := s.SobreCifrado.huellaSHA256
	if !aadValida || errAAD != nil || !s.EnvolturaClave.valida() || !s.SobreCifrado.valida() ||
		s.EstadoEsperado.Validar() != nil || !s.PerfilEsperado.valida() || !s.Procedencia.valida() ||
		!perfilesCifradoCoinciden(s.PerfilEsperado, perfilEnvoltura) ||
		!perfilesCifradoCoinciden(s.PerfilEsperado, perfilSobre) ||
		!s.AtestacionKMS.validaEstructural() ||
		!perfilesCifradoCoinciden(s.PerfilEsperado, s.AtestacionKMS.Perfil) ||
		!procedenciasActoCoinciden(s.Procedencia, s.AtestacionKMS.Procedencia) ||
		!coincideTextoConstante(huellaAAD, huellaAADEnvoltura) ||
		!coincideTextoConstante(huellaAAD, huellaAADSobre) ||
		!coincideTextoConstante(huellaAAD, s.AtestacionKMS.HuellaAAD) ||
		!coincideTextoConstante(huellaEnvoltura, s.AtestacionKMS.HuellaEnvolturaSHA256) ||
		!coincideTextoConstante(huellaSobre, s.AtestacionKMS.HuellaSobreSHA256) ||
		claveRef != s.AtestacionKMS.ClaveMaestraRef || versionClave != s.AtestacionKMS.VersionClave ||
		!materialAAD.coincide(s.EstadoEsperado, s.PerfilEsperado, s.Procedencia) {
		return ErrDescifradoBorradorInvalido
	}
	return nil
}

func (m materialAADCanonicaCifradoBorrador) coincide(
	estado puertosbolsa.ReferenciaEstadoVersionConvocatoria,
	perfil PerfilCifradoBorrador,
	procedencia ProcedenciaActoBorrador,
) bool {
	return m.VersionRef == estado.Referencia && m.VersionRevision == estado.Revision &&
		coincideTextoConstante(m.HuellaVersionSHA256, estado.HuellaEstadoSHA256) &&
		m.PerfilCifradoRef == perfil.Referencia && m.PerfilCifradoVersion == perfil.Version &&
		coincideTextoConstante(m.HuellaPerfilSHA256, perfil.HuellaContenidoSHA256) &&
		m.AlgoritmoAEAD == perfil.AlgoritmoAEAD &&
		m.AlgoritmoEnvolturaClave == perfil.AlgoritmoEnvolturaClave &&
		m.ProcedenciaEsquema == procedencia.Esquema && m.PerfilEjecucion == procedencia.PerfilEjecucion &&
		m.AutoridadActo == procedencia.Autoridad &&
		m.ProveedorProcedenciaRef == procedencia.ProveedorRef &&
		m.MigrableProduccion == procedencia.MigrableProduccion
}

// MaterialParaConectorConfiable devuelve copias que el adaptador debe borrar
// incluso en error. Nunca incluye el agregado descifrado.
func (s SolicitudDescifradoBorradorDurable) MaterialParaConectorConfiable() (
	aad, claveEnvuelta, nonce, textoCifrado []byte,
	err error,
) {
	if s.Validar() != nil {
		return nil, nil, nil, nil, ErrDescifradoBorradorInvalido
	}
	aad, err = s.AAD.RepresentacionCanonica()
	if err != nil {
		return nil, nil, nil, nil, ErrDescifradoBorradorInvalido
	}
	_, _, _, claveEnvuelta, _, _, err = s.EnvolturaClave.DatosParaPersistencia()
	if err != nil {
		borrarBytesDescifradoBorrador(aad)
		return nil, nil, nil, nil, ErrDescifradoBorradorInvalido
	}
	_, nonce, textoCifrado, _, _, err = s.SobreCifrado.DatosParaPersistencia()
	if err != nil {
		borrarBytesDescifradoBorrador(aad, claveEnvuelta)
		return nil, nil, nil, nil, ErrDescifradoBorradorInvalido
	}
	return aad, claveEnvuelta, nonce, textoCifrado, nil
}

type ResultadoDescifradoBorradorDurable struct {
	bloqueoSerializacionDiario
	version dominiobolsa.VersionConvocatoriaGobernada
	estado  puertosbolsa.ReferenciaEstadoVersionConvocatoria
}

// NuevoResultadoDescifradoBorradorDurable es la única salida válida de un
// conector. Decodifica forma canónica y vuelve a fijar referencia+revisión+
// huella antes de conservar el agregado en memoria.
func NuevoResultadoDescifradoBorradorDurable(
	solicitud SolicitudDescifradoBorradorDurable,
	claro []byte,
) (ResultadoDescifradoBorradorDurable, error) {
	if solicitud.Validar() != nil || len(claro) == 0 {
		return ResultadoDescifradoBorradorDurable{}, ErrDescifradoBorradorInvalido
	}
	version, err := dominiobolsa.DecodificarVersionConvocatoriaGobernadaCanonica(claro)
	if err != nil {
		return ResultadoDescifradoBorradorDurable{}, ErrDescifradoBorradorInvalido
	}
	estado, err := puertosbolsa.EstadoVersionConvocatoria(version)
	if err != nil || !estadosVersionBorradorCoinciden(estado, solicitud.EstadoEsperado) {
		return ResultadoDescifradoBorradorDurable{}, ErrDescifradoBorradorInvalido
	}
	resultado := ResultadoDescifradoBorradorDurable{version: version, estado: estado}
	if resultado.validarPara(solicitud) != nil {
		return ResultadoDescifradoBorradorDurable{}, ErrDescifradoBorradorInvalido
	}
	return resultado, nil
}

func (r ResultadoDescifradoBorradorDurable) VersionConvocatoria() (
	dominiobolsa.VersionConvocatoriaGobernada,
	error,
) {
	if r.version.Validar() != nil {
		return dominiobolsa.VersionConvocatoriaGobernada{}, ErrDescifradoBorradorInvalido
	}
	estado, err := puertosbolsa.EstadoVersionConvocatoria(r.version)
	if err != nil || !estadosVersionBorradorCoinciden(estado, r.estado) {
		return dominiobolsa.VersionConvocatoriaGobernada{}, ErrDescifradoBorradorInvalido
	}
	version, err := r.version.ClonarCanonico()
	if err != nil {
		return dominiobolsa.VersionConvocatoriaGobernada{}, ErrDescifradoBorradorInvalido
	}
	return version, nil
}

func (r ResultadoDescifradoBorradorDurable) validarPara(
	solicitud SolicitudDescifradoBorradorDurable,
) error {
	estado, err := puertosbolsa.EstadoVersionConvocatoria(r.version)
	if solicitud.Validar() != nil || err != nil ||
		!estadosVersionBorradorCoinciden(estado, r.estado) ||
		!estadosVersionBorradorCoinciden(estado, solicitud.EstadoEsperado) {
		return ErrDescifradoBorradorInvalido
	}
	return nil
}

func estadosVersionBorradorCoinciden(
	a, b puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) bool {
	return a.Validar() == nil && b.Validar() == nil && a.Referencia == b.Referencia &&
		a.Revision == b.Revision && coincideTextoConstante(a.HuellaEstadoSHA256, b.HuellaEstadoSHA256)
}

type DescifradorBorradorDurable interface {
	DescifrarBorrador(
		context.Context,
		SolicitudDescifradoBorradorDurable,
	) (ResultadoDescifradoBorradorDurable, error)
}

func borrarBytesDescifradoBorrador(conjuntos ...[]byte) {
	for _, datos := range conjuntos {
		for indice := range datos {
			datos[indice] = 0
		}
	}
}
