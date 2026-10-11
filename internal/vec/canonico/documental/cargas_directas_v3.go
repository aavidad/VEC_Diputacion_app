package documental

import (
	"encoding/json"
)

// Las preimágenes de carga contienen exclusivamente valores escalares ya
// validados por aplicación y puertos. El orden de los campos es parte del
// contrato AD240 y se comprueba con vectores de bytes.
type ReservaCargaV3 struct {
	Accion                string `json:"accion"`
	CargaID               string `json:"carga_id"`
	RecursoRef            string `json:"recurso_ref"`
	PersonaRef            string `json:"persona_ref"`
	ClaveReserva          string `json:"clave_reserva"`
	HuellaReservaSHA256   string `json:"huella_reserva_sha256"`
	VersionReserva        uint64 `json:"version_reserva"`
	IndiceCargaHMAC       string `json:"indice_carga_hmac"`
	HuellaCargaHMAC       string `json:"huella_carga_hmac"`
	MIME                  string `json:"mime"`
	Tamano                int64  `json:"tamano"`
	HuellaContenido       string `json:"huella_contenido_sha256"`
	HuellaCargaJSONSHA256 string `json:"huella_carga_json_sha256"`
	HuellaToken           string `json:"huella_token_sha256"`
}

func (d ReservaCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type ConfirmacionPreparacionCargaV3 struct {
	Accion               string `json:"accion"`
	CargaID              string `json:"carga_id"`
	RecursoRef           string `json:"recurso_ref"`
	PersonaRef           string `json:"persona_ref"`
	HuellaToken          string `json:"huella_token_sha256"`
	HuellaManifiesto     string `json:"huella_manifiesto_sha256"`
	HuellaManifiestoJSON string `json:"huella_manifiesto_json_sha256"`
	VersionEsperada      int    `json:"version_esperada"`
	HuellaAnterior       string `json:"huella_anterior_sha256"`
	VersionSiguiente     int    `json:"version_siguiente"`
	HuellaSiguiente      string `json:"huella_siguiente_sha256"`
}

func (d ConfirmacionPreparacionCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

func HuellaJSONCargaV3(valor any) (string, error) {
	bytes, err := json.Marshal(valor)
	if err != nil {
		return "", err
	}
	return HuellaBytesSHA256(bytes), nil
}

type TransicionCargaV3 struct {
	Accion            string `json:"accion"`
	CargaID           string `json:"carga_id"`
	RecursoRef        string `json:"recurso_ref"`
	PersonaRef        string `json:"persona_ref"`
	VersionEsperada   int    `json:"version_esperada"`
	HuellaAnterior    string `json:"huella_anterior_sha256"`
	VersionSiguiente  int    `json:"version_siguiente"`
	HuellaSiguiente   string `json:"huella_siguiente_sha256"`
	AutorizacionV1Ref string `json:"autorizacion_v1_ref"`
	Evento            string `json:"evento"`
}

func (d TransicionCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type ConsultaCargaV3 struct {
	Accion               string `json:"accion"`
	CargaID              string `json:"carga_id"`
	RecursoRef           string `json:"recurso_ref"`
	PersonaRef           string `json:"persona_ref"`
	PerfilActivoRef      string `json:"perfil_activo_ref"`
	RegistroContextoRef  string `json:"registro_contexto_ref"`
	HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
	Finalidad            string `json:"finalidad"`
	CorrelacionRef       string `json:"correlacion_ref"`
	IncluyeManifiesto    bool   `json:"incluye_manifiesto"`
}

func (d ConsultaCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type ConciliacionCargaV3 struct {
	Accion                   string `json:"accion"`
	CargaID                  string `json:"carga_id"`
	RecursoRef               string `json:"recurso_ref"`
	VersionOperativaEsperada uint64 `json:"version_operativa_esperada"`
	EstadoOperativoEsperado  string `json:"estado_operativo_esperado"`
	EstadoResultado          string `json:"estado_resultado"`
	EvidenciaRef             string `json:"evidencia_ref"`
	HuellaEvidenciaSHA256    string `json:"huella_evidencia_sha256"`
	MotivoRef                string `json:"motivo_ref"`
}

func (d ConciliacionCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type EmisionReciboCargaV3 struct {
	Accion                 string `json:"accion"`
	CargaID                string `json:"carga_id"`
	RecursoRef             string `json:"recurso_ref"`
	VersionCarga           int    `json:"version_carga"`
	PersonaRef             string `json:"persona_ref"`
	RegistroContextoRef    string `json:"registro_contexto_ref"`
	HuellaContextoSHA256   string `json:"huella_contexto_sha256"`
	HuellaManifiestoSHA256 string `json:"huella_manifiesto_sha256"`
	IndiceHMAC             string `json:"indice_hmac"`
	GrupoHMAC              string `json:"grupo_hmac"`
	VinculoHMAC            string `json:"vinculo_hmac"`
	EvidenciaAltaRef       string `json:"evidencia_alta_ref"`
	ExpiraEn               string `json:"expira_en"`
}

func (d EmisionReciboCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type IntencionConfirmacionCargaV3 struct {
	Accion               string `json:"accion"`
	CargaID              string `json:"carga_id"`
	RecursoRef           string `json:"recurso_ref"`
	PersonaRef           string `json:"persona_ref"`
	VersionCarga         int    `json:"version_carga"`
	RegistroContextoRef  string `json:"registro_contexto_ref"`
	HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
	IndiceHMAC           string `json:"indice_hmac"`
	GrupoHMAC            string `json:"grupo_hmac"`
	VinculoHMAC          string `json:"vinculo_hmac"`
	EvidenciaConsumoRef  string `json:"evidencia_consumo_ref"`
	IntencionRef         string `json:"intencion_ref"`
	HuellaIntencionHMAC  string `json:"huella_intencion_hmac"`
	RegistradoEn         string `json:"registrado_en"`
	ValidaHasta          string `json:"valida_hasta"`
}

func (d IntencionConfirmacionCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type IntencionPromocionCargaV3 struct {
	Accion                string `json:"accion"`
	CargaID               string `json:"carga_id"`
	RecursoRef            string `json:"recurso_ref"`
	PersonaRef            string `json:"persona_ref"`
	VersionCarga          int    `json:"version_carga"`
	HuellaCargaJSONSHA256 string `json:"huella_carga_json_sha256"`
	IndicePromocionHMAC   string `json:"indice_promocion_hmac"`
	OrigenRef             string `json:"origen_ref"`
	OrigenVersion         string `json:"origen_version"`
	HuellaOrigenSHA256    string `json:"huella_origen_sha256"`
	EvidenciaAnalisisRef  string `json:"evidencia_analisis_ref"`
	RegistroContextoRef   string `json:"registro_contexto_ref"`
	HuellaContextoSHA256  string `json:"huella_contexto_sha256"`
}

func (d IntencionPromocionCargaV3) Preimagen() ([]byte, error) { return json.Marshal(d) }

type ResultadoPromocionCargaV3 struct {
	TransicionCargaV3
	IndicePromocionHMAC string `json:"indice_promocion_hmac"`
}

func (d ResultadoPromocionCargaV3) Preimagen() ([]byte, error) {
	// No incrustar TransicionCargaV3: encoding/json aplanaría el orden de los
	// campos por reglas de promoción. Esta proyección fija el orden explícito.
	return json.Marshal(struct {
		Accion              string `json:"accion"`
		CargaID             string `json:"carga_id"`
		RecursoRef          string `json:"recurso_ref"`
		PersonaRef          string `json:"persona_ref"`
		VersionEsperada     int    `json:"version_esperada"`
		HuellaAnterior      string `json:"huella_anterior_sha256"`
		VersionSiguiente    int    `json:"version_siguiente"`
		HuellaSiguiente     string `json:"huella_siguiente_sha256"`
		AutorizacionV1Ref   string `json:"autorizacion_v1_ref"`
		Evento              string `json:"evento"`
		IndicePromocionHMAC string `json:"indice_promocion_hmac"`
	}{d.Accion, d.CargaID, d.RecursoRef, d.PersonaRef, d.VersionEsperada, d.HuellaAnterior,
		d.VersionSiguiente, d.HuellaSiguiente, d.AutorizacionV1Ref, d.Evento, d.IndicePromocionHMAC})
}

// HuellaEfectoCargaV3 usa la misma forma de contexto que Documentos V3:
// organización fija y atributo preimagen_sha256. AD240 recalcula estos bytes.
func HuellaEfectoCargaV3(preimagen []byte) string {
	if len(preimagen) == 0 || len(preimagen) > 16384 {
		return ""
	}
	contexto := `{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"` +
		HuellaBytesSHA256(preimagen) + `"}}`
	return HuellaBytesSHA256([]byte(contexto))
}
