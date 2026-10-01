// Package ensayologicopg ejecuta ensayos lógicos sintéticos en un runtime local
// desechable. No concede permisos ni decide la validez de una copia completa.
package ensayologicopg

import (
	"context"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type Configuracion struct {
	ImagenSHA256       string        `json:"imagen_sha256"`
	VersionPostgreSQL  string        `json:"version_postgresql"`
	UsuarioBootstrap   string        `json:"usuario_bootstrap"`
	LimiteArchivoBytes int64         `json:"limite_archivo_bytes"`
	CPUs               int           `json:"cpus"`
	MemoriaBytes       int64         `json:"memoria_bytes"`
	TiempoLimite       time.Duration `json:"tiempo_limite"`
}

type Archivo struct {
	Ruta   string `json:"ruta"`
	SHA256 string `json:"sha256"`
}

type Solicitud struct {
	Sintetica bool    `json:"sintetica"`
	Dump      Archivo `json:"dump"`
	Globals   Archivo `json:"globals"`
}

type Resultado struct {
	Estado               string         `json:"estado"`
	Etapa                string         `json:"etapa"`
	VersionPostgreSQL    string         `json:"version_postgresql"`
	HabilitaRestauracion bool           `json:"habilita_restauracion"`
	LimpiezaCompletada   bool           `json:"limpieza_completada"`
	Razones              []copias.Razon `json:"razones"`
}

type Ensayador struct{ Configuracion Configuracion }

var (
	huellaValida  = regexp.MustCompile(`^[a-f0-9]{64}$`)
	versionValida = regexp.MustCompile(`^18\.[0-9]+$`)
	usuarioValido = regexp.MustCompile(`^cs06_[a-z][a-z0-9_]{1,40}$`)
)

func (c Configuracion) validar() bool {
	return huellaValida.MatchString(c.ImagenSHA256) && versionValida.MatchString(c.VersionPostgreSQL) &&
		usuarioValido.MatchString(c.UsuarioBootstrap) && c.LimiteArchivoBytes > 0 && c.LimiteArchivoBytes <= 1<<30 &&
		c.CPUs > 0 && c.CPUs <= 8 && c.MemoriaBytes >= 64<<20 && c.MemoriaBytes <= 16<<30 &&
		c.TiempoLimite >= time.Second && c.TiempoLimite <= 30*time.Minute
}

func fallo(r *Resultado, etapa, clave, esperado, obtenido string) {
	r.Estado, r.Etapa = "restauracion_logica_fallida", etapa
	r.Razones = append(r.Razones, copias.Razon{Codigo: "ensayo_logico_fallido", Clave: clave,
		Esperado: esperado, Obtenido: obtenido, Accion: "revisar_ensayo_sintetico"})
}

// Ensayar sólo acepta archivos locales explícitamente sintéticos. La marca del
// operador no identifica ni anonimiza datos personales. El destino es siempre
// nuevo, aislado y eliminado; nunca recibe un DSN ni un contenedor de origen.
func (e Ensayador) Ensayar(ctx context.Context, s Solicitud) (r Resultado) {
	r = Resultado{Estado: "restauracion_logica_fallida", Etapa: "entrada", LimpiezaCompletada: true,
		Razones: []copias.Razon{}}
	if ctx == nil || !e.Configuracion.validar() || !s.Sintetica ||
		!huellaValida.MatchString(s.Dump.SHA256) || !huellaValida.MatchString(s.Globals.SHA256) {
		fallo(&r, "entrada", "entrada", "configuracion_y_muestra_sintetica", "no_admitida")
		return r
	}
	ctx, cancelar := context.WithTimeout(ctx, e.Configuracion.TiempoLimite)
	defer cancelar()
	return e.ensayar(ctx, s, r)
}
