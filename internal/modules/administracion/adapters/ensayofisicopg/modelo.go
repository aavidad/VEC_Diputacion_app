// Package ensayofisicopg restaura muestras físicas sintéticas en recursos propios.
package ensayofisicopg

import (
	"context"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

type Configuracion struct {
	RaizTemporal        string        `json:"raiz_temporal,omitempty"`
	ImagenSHA256        string        `json:"imagen_sha256"`
	VersionPostgreSQL   string        `json:"version_postgresql"`
	UsuarioBootstrap    string        `json:"usuario_bootstrap"`
	LimiteArchivoBytes  int64         `json:"limite_archivo_bytes"`
	LimiteExtraidoBytes int64         `json:"limite_extraido_bytes"`
	LimiteEntradas      int           `json:"limite_entradas"`
	CPUs                int           `json:"cpus"`
	MemoriaBytes        int64         `json:"memoria_bytes"`
	TiempoLimite        time.Duration `json:"tiempo_limite"`
}

type Archivo struct {
	Ruta   string `json:"ruta"`
	SHA256 string `json:"sha256"`
}
type Componente struct {
	ID   string  `json:"id"`
	Tipo string  `json:"tipo"`
	Tar  Archivo `json:"tar"`
}
type Solicitud struct {
	Sintetica   bool         `json:"sintetica"`
	Componentes []Componente `json:"componentes"`
}
type Resultado struct {
	Estado               string              `json:"estado"`
	Etapa                string              `json:"etapa"`
	VersionPostgreSQL    string              `json:"version_postgresql"`
	HabilitaRestauracion bool                `json:"habilita_restauracion"`
	LimpiezaCompletada   bool                `json:"limpieza_completada"`
	Observacion          puertos.Observacion `json:"observacion"`
	Razones              []copias.Razon      `json:"razones"`
}
type Ensayador struct {
	Configuracion Configuracion
	Observador    puertos.Observador
}

var (
	huellaValida  = regexp.MustCompile(`^[a-f0-9]{64}$`)
	versionValida = regexp.MustCompile(`^18\.[0-9]{1,2}$`)
	usuarioValido = regexp.MustCompile(`^cs06_[a-z][a-z0-9_]{1,40}$`)
	idValido      = regexp.MustCompile(`^fisica:[a-zA-Z0-9][a-zA-Z0-9:_.-]{0,120}$`)
	tipoValido    = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

func (c Configuracion) validar() bool {
	return huellaValida.MatchString(c.ImagenSHA256) && versionValida.MatchString(c.VersionPostgreSQL) && usuarioValido.MatchString(c.UsuarioBootstrap) &&
		c.LimiteArchivoBytes > 0 && c.LimiteArchivoBytes <= 1<<30 && c.LimiteExtraidoBytes > 0 && c.LimiteExtraidoBytes <= 1<<30 &&
		c.LimiteEntradas > 0 && c.LimiteEntradas <= 100000 && c.CPUs > 0 && c.CPUs <= 8 && c.MemoriaBytes >= 64<<20 && c.MemoriaBytes <= 16<<30 && c.TiempoLimite >= time.Second && c.TiempoLimite <= 30*time.Minute
}
func fallo(r *Resultado, etapa, clave string) bool {
	r.Estado, r.Etapa = "restauracion_fisica_fallida", etapa
	r.Razones = append(r.Razones, copias.Razon{Codigo: "ensayo_fisico_fallido", Clave: clave, Esperado: "comprobado", Obtenido: "no_comprobable", Accion: "revisar_ensayo_sintetico"})
	return false
}
func (e Ensayador) Ensayar(ctx context.Context, s Solicitud) Resultado {
	r := Resultado{Estado: "restauracion_fisica_fallida", Etapa: "entrada", LimpiezaCompletada: true, Razones: []copias.Razon{}, Observacion: puertos.Observacion{ContrasteEstado: "no_comprobable", ArranqueEstado: "no_comprobable"}}
	if ctx == nil || !e.Configuracion.validar() || !s.Sintetica || len(s.Componentes) == 0 || len(s.Componentes) > 128 {
		fallo(&r, "entrada", "configuracion")
		return r
	}
	vistos := map[string]bool{}
	for i, c := range s.Componentes {
		if !idValido.MatchString(c.ID) || !tipoValido.MatchString(c.Tipo) || !huellaValida.MatchString(c.Tar.SHA256) || vistos[c.ID] || (i == 0 && (c.ID != "fisica:pgdata" || c.Tipo != "base_fisica")) || (i > 0 && c.Tipo == "base_fisica") {
			fallo(&r, "entrada", "componentes")
			return r
		}
		vistos[c.ID] = true
	}
	ctx, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(e.Configuracion.TiempoLimite))
	defer cancelar()
	if ctx.Err() != nil {
		fallo(&r, "entrada", "plazo_ensayo")
		return r
	}
	return e.ensayar(ctx, s, r)
}
