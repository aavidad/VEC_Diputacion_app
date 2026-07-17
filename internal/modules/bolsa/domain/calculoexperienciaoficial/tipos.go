package calculoexperienciaoficial

const (
	esquemaClaveEfectoV1 = "vec.bolsa.calculo-experiencia-oficial.clave-efecto.v1"
	esquemaIntencionV1   = "vec.bolsa.calculo-experiencia-oficial.intencion-resultado.v1"
	esquemaReciboV1      = "vec.bolsa.calculo-experiencia-oficial.recibo.v1"
	dominioIndiceHMACV1  = "vec.bolsa.calculo-experiencia-oficial.indice-efecto.v1"

	maximoBytesRepresentacionV1 = 32 * 1024
	minimoBytesSecretoHMACV1    = 32
	maximoBytesSecretoHMACV1    = 1024
	maximoVersionV1             = 1_000_000_000
)

// ReferenciaExactaV1 nunca significa «la vigente»: fija identidad, versión y
// contenido. Replica el contrato léxico de ReferenciaVersionada de reglas; la
// capa confiable debe emitir referencias opacas, nunca datos personales.
type ReferenciaExactaV1 struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

// VinculoReglasV1 fija tanto el contenido como el estado gobernado exacto.
type VinculoReglasV1 struct {
	Contenido          ReferenciaExactaV1 `json:"contenido"`
	Revision           uint64             `json:"revision"`
	HuellaEstadoSHA256 string             `json:"huella_estado_sha256"`
}

type VinculoEntradaV1 struct {
	Instantanea           ReferenciaExactaV1 `json:"instantanea"`
	HuellaContenidoSHA256 string             `json:"huella_contenido_sha256"`
}

type VinculoMotorV1 struct {
	Contrato             string `json:"contrato"`
	Version              uint64 `json:"version"`
	HuellaContratoSHA256 string `json:"huella_contrato_sha256"`
}

// CausaGobernadaV1 evita incorporar motivos libres a la identidad semántica.
type CausaGobernadaV1 struct {
	Catalogo ReferenciaExactaV1 `json:"catalogo"`
	Clave    string             `json:"clave"`
}

// VinculoPredecesorV1 identifica un recibo oficial inmutable por referencia y
// huella. Solo puede aparecer en una rectificación.
type VinculoPredecesorV1 struct {
	ReferenciaRecibo   string `json:"referencia_recibo"`
	HuellaReciboSHA256 string `json:"huella_recibo_sha256"`
}

type TipoEfectoV1 string

const (
	EfectoCalculoInicial TipoEfectoV1 = "calculo_inicial"
	EfectoRectificacion  TipoEfectoV1 = "rectificacion"
)

type DatosClaveEfectoV1 struct {
	// SujetoPseudonimizado es la ReferenciaVersionada exacta emitida por la
	// fuente confiable; nunca un DNI, nombre, correo ni pseudónimo sin huella.
	SujetoPseudonimizado ReferenciaExactaV1
	Convocatoria         ReferenciaExactaV1
	Reglas               VinculoReglasV1
	Entrada              VinculoEntradaV1
	Motor                VinculoMotorV1
	HuellaPlanSHA256     string
	Causa                CausaGobernadaV1
	Tipo                 TipoEfectoV1
	Predecesor           *VinculoPredecesorV1
}

type EstadoResultadoV1 string

const (
	ResultadoCompletado EstadoResultadoV1 = "completado"
	ResultadoBloqueado  EstadoResultadoV1 = "bloqueado"
)

type FaseResultadoV1 string

const (
	FaseSeleccion  FaseResultadoV1 = "seleccion"
	FaseIntervalos FaseResultadoV1 = "intervalos"
	FasePuntuacion FaseResultadoV1 = "puntuacion"
	FaseCompletado FaseResultadoV1 = "completado"
)

type materialClaveEfectoV1 struct {
	Esquema              string               `json:"esquema"`
	SujetoPseudonimizado ReferenciaExactaV1   `json:"sujeto_pseudonimizado"`
	Convocatoria         ReferenciaExactaV1   `json:"convocatoria"`
	Reglas               VinculoReglasV1      `json:"reglas"`
	Entrada              VinculoEntradaV1     `json:"entrada"`
	Motor                VinculoMotorV1       `json:"motor"`
	HuellaPlanSHA256     string               `json:"huella_plan_sha256"`
	Causa                CausaGobernadaV1     `json:"causa"`
	Tipo                 TipoEfectoV1         `json:"tipo"`
	Predecesor           *VinculoPredecesorV1 `json:"predecesor,omitempty"`
}

type materialIntencionResultadoV1 struct {
	Esquema               string                `json:"esquema"`
	Clave                 materialClaveEfectoV1 `json:"clave"`
	HuellaResultadoSHA256 string                `json:"huella_resultado_sha256"`
	Estado                EstadoResultadoV1     `json:"estado"`
	Fase                  FaseResultadoV1       `json:"fase"`
}

type materialReciboV1 struct {
	Esquema                 string            `json:"esquema"`
	Referencia              string            `json:"referencia"`
	GeneracionClaveHMAC     uint32            `json:"generacion_clave_hmac"`
	IndiceEfectoHMACSHA256  string            `json:"indice_efecto_hmac_sha256"`
	HuellaClaveEfectoSHA256 string            `json:"huella_clave_efecto_sha256"`
	HuellaIntencionSHA256   string            `json:"huella_intencion_sha256"`
	HuellaResultadoSHA256   string            `json:"huella_resultado_sha256"`
	Estado                  EstadoResultadoV1 `json:"estado"`
	Fase                    FaseResultadoV1   `json:"fase"`
}
