// Package copias contiene contratos y reglas puras de compatibilidad declarada.
// Sus resultados no acreditan autenticidad, autorización ni restauraciones reales.
package copias

import "time"

const FormatoVersion = 1

type Estado string

const (
	Compatible    Estado = "compatible"
	Incompatible  Estado = "incompatible"
	NoComprobable Estado = "no_comprobable"
)

type Razon struct {
	Codigo   string `json:"codigo"`
	Clave    string `json:"clave"`
	Esperado string `json:"esperado"`
	Obtenido string `json:"obtenido"`
	Accion   string `json:"accion"`
}

type Resultado struct {
	Estado  Estado  `json:"estado"`
	Razones []Razon `json:"razones"`
}

type Artefacto struct {
	ID          string `json:"id"`
	Tipo        string `json:"tipo"`
	SHA256      string `json:"sha256"`
	TamanoBytes int64  `json:"tamano_bytes"`
}

type Migracion struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Modulo struct {
	ID            string      `json:"id"`
	EsquemaSHA256 string      `json:"esquema_sha256"`
	Migraciones   []Migracion `json:"migraciones"`
}

type Release struct {
	ID              string      `json:"id"`
	Commit          string      `json:"commit"`
	Plataforma      string      `json:"plataforma"`
	Binarios        []Artefacto `json:"binarios"`
	Componentes     []Artefacto `json:"componentes"`
	EsquemaEsperado []Modulo    `json:"esquema_esperado"`
}

type Herramienta struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Extension struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Almacen struct {
	ID     string `json:"id"`
	Tipo   string `json:"tipo"`
	SHA256 string `json:"sha256"`
}

type PostgreSQL struct {
	Version        string        `json:"version"`
	RuntimeSHA256  string        `json:"runtime_sha256"`
	Plataforma     string        `json:"plataforma"`
	Herramientas   []Herramienta `json:"herramientas"`
	Extensiones    []Extension   `json:"extensiones"`
	ClusterRef     string        `json:"cluster_ref"`
	Bases          []string      `json:"bases"`
	Almacenes      []Almacen     `json:"almacenes"`
	AmbitoCompleto bool          `json:"ambito_completo"`
}

type Inventario struct {
	FormatoVersion int        `json:"formato_version"`
	Ref            string     `json:"ref"`
	Completo       bool       `json:"completo"`
	PostgreSQL     PostgreSQL `json:"postgresql"`
	Release        Release    `json:"release"`
	Modulos        []Modulo   `json:"modulos"`
}

type Consistencia struct {
	Modo                 string `json:"modo"`
	EscritoresExcluidos  bool   `json:"escritores_excluidos"`
	ParadaLimpia         bool   `json:"parada_limpia"`
	EvidenciaRef         string `json:"evidencia_ref"`
	PuntoRecuperacionRef string `json:"punto_recuperacion_ref"`
}

type Proteccion struct {
	Formato          string `json:"formato"`
	Algoritmo        string `json:"algoritmo"`
	ClaveRef         string `json:"clave_ref"`
	ClaveVersion     string `json:"clave_version"`
	AutenticacionRef string `json:"autenticacion_ref"`
	CifradoSHA256    string `json:"cifrado_sha256"`
}

// Evidencia describe lo registrado por otro componente. No es una atestación.
type Evidencia struct {
	Ref                  string `json:"ref"`
	RecuentosSHA256      string `json:"recuentos_sha256"`
	ContenidoSHA256      string `json:"contenido_sha256"`
	EsquemaSHA256        string `json:"esquema_sha256"`
	RolesSHA256          string `json:"roles_sha256"`
	ACLSHA256            string `json:"acl_sha256"`
	SecuenciasSHA256     string `json:"secuencias_sha256"`
	ObjetosGrandesSHA256 string `json:"objetos_grandes_sha256"`
	FicherosSHA256       string `json:"ficheros_sha256"`
	ArranqueRef          string `json:"arranque_ref"`
}

type Verificacion struct {
	Estado             string    `json:"estado"`
	VerificadorVersion string    `json:"verificador_version"`
	Fecha              time.Time `json:"fecha"`
	Fisica             Evidencia `json:"fisica"`
	Logica             Evidencia `json:"logica"`
}

type Manifiesto struct {
	FormatoVersion   int          `json:"formato_version"`
	ConjuntoRef      string       `json:"conjunto_ref"`
	OperacionRef     string       `json:"operacion_ref"`
	SolicitanteRef   string       `json:"solicitante_ref"`
	MotivoRef        string       `json:"motivo_ref"`
	PoliticaRef      string       `json:"politica_ref"`
	Inicio           time.Time    `json:"inicio"`
	Fin              time.Time    `json:"fin"`
	TamanoBytes      int64        `json:"tamano_bytes"`
	Inventario       Inventario   `json:"inventario"`
	Componentes      []Artefacto  `json:"componentes"`
	InventarioSHA256 string       `json:"inventario_sha256"`
	Consistencia     Consistencia `json:"consistencia"`
	Proteccion       Proteccion   `json:"proteccion"`
	Verificacion     Verificacion `json:"verificacion"`
}

// RuntimeAdmitido procede de la política vigente, nunca de defaults de la CLI.
type RuntimeAdmitido struct {
	Version       string        `json:"version"`
	RuntimeSHA256 string        `json:"runtime_sha256"`
	Plataforma    string        `json:"plataforma"`
	Herramientas  []Herramienta `json:"herramientas"`
	Extensiones   []Extension   `json:"extensiones"`
}

type Politica struct {
	FormatoVersion    int               `json:"formato_version"`
	Ref               string            `json:"ref"`
	Runtimes          []RuntimeAdmitido `json:"runtimes"`
	Releases          []Release         `json:"releases"`
	ReleasesRevocadas []string          `json:"releases_revocadas"`
}

type ModoRestauracion string

const ConjuntoCompleto ModoRestauracion = "conjunto_completo"
