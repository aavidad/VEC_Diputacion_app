// Package ensayofisicopg contiene contratos offline de observación del ensayo.
package ensayofisicopg

import "context"

// Ejecutores sólo existen durante el callback. No aceptan DSN, contenedores,
// volúmenes ni ejecutables del host. Los argumentos proceden de adaptadores
// confiables, nunca de una petición web o de contenido SQL de una copia.
type EjecutorPostgreSQL interface {
	EjecutarPostgreSQL(context.Context, string, []string, []byte, int) ([]byte, error)
	ComprobarExclusion(context.Context) (string, error)
}

// EjecutarArchivado ejecuta el binario por ID del inventario ya verificado.
// Su salida no acredita por sí sola salud ni autorización de VEC.
type EjecutorArchivado interface {
	LeerArchivado(context.Context, string, int) ([]byte, error)
	IniciarArchivado(context.Context, string, []string, map[string]string) (ProcesoArchivado, error)
	EjecutarArchivado(context.Context, string, []string, map[string]string, int) ([]byte, error)
}

type Componente struct {
	ID              string
	Tipo            string
	RutaInterna     string
	RutaMaterial    string
	SHA256          string
	ContenidoSHA256 string
	ContenidoBytes  int64
}

type EvidenciaAislamiento struct {
	ImagenSHA256                string
	RedSinSalida                bool
	VolumenesPropios            bool
	ConfiguracionOrigenExcluida bool
	RuntimeInspeccionado        bool
}

type Entorno struct {
	PostgreSQL  EjecutorPostgreSQL
	Archivado   EjecutorArchivado
	Componentes []Componente
	Aislamiento EvidenciaAislamiento
}

type Observacion struct {
	ContrasteEstado              string `json:"contraste_estado"`
	ArranqueEstado               string `json:"arranque_estado"`
	EvidenciaSHA256              string `json:"evidencia_sha256"`
	SaludComprobada              bool   `json:"salud_comprobada"`
	ConsultaAutorizadaComprobada bool   `json:"consulta_autorizada_comprobada"`
	DespachosBloqueados          bool   `json:"despachos_bloqueados"`
}

// Observar se llama antes de destruir el runtime. Un fallo deja el ensayo
// no comprobable. Un observador de pruebas no es evidencia de conjunto válida.
type Observador interface {
	Observar(context.Context, Entorno) (Observacion, error)
}

// SondaHTTP sólo accede al loopback del runtime. Las credenciales y certificados
// sintéticos de una consulta autorizada proceden del perfil archivado protegido.
type SondaHTTP struct {
	Puerto        uint16 `json:"puerto"`
	Ruta          string `json:"ruta"`
	Authorization string `json:"authorization,omitempty"`
	TLS           bool   `json:"tls"`
	CA            string `json:"ca,omitempty"`
	Certificado   string `json:"certificado,omitempty"`
	Clave         string `json:"clave,omitempty"`
	LimiteBytes   int64  `json:"limite_bytes"`
}
type RespuestaHTTP struct {
	EstadoHTTP int    `json:"estado_http"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes"`
	Contenido  []byte `json:"contenido,omitempty"`
}
type ProcesoArchivado interface {
	SondarHTTP(context.Context, SondaHTTP) (RespuestaHTTP, error)
	Detener(context.Context) error
}
