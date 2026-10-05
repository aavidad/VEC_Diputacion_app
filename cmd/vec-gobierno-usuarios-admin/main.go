// Command vec-gobierno-usuarios-admin envuelve las tres fases del gobierno
// técnico de usuarios ADMIN V3 (AD188): preparar el material y el plan,
// aplicarlos con el LOGIN técnico del DBA y verificar la cadena de auditoría.
// No crea LOGIN, GRANT ni configuración aprobada: eso pertenece al DBA.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/shared/i18n"
)

const (
	limiteDocumento = 64 << 10
	limiteTimeout   = 10 * time.Minute
)

type datosTextos struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}

// diagnostico es lo único que sale por consola: códigos, textos del catálogo
// y huellas públicas. Nunca DSN, rutas, secretos ni el acuse completo.
type diagnostico struct {
	Codigo        string                                      `json:"codigo"`
	Mensaje       string                                      `json:"mensaje"`
	Limite        string                                      `json:"limite"`
	Estado        string                                      `json:"estado,omitempty"`
	Confirmado    bool                                        `json:"confirmado"`
	AcuseGuardado bool                                        `json:"acuse_guardado"`
	Replay        bool                                        `json:"replay"`
	Preparacion   *bootstrap.PreparacionGobiernoUsuariosAdmin `json:"preparacion,omitempty"`
}

// clavesTextos es la lista cerrada del catálogo: ni una más ni una menos.
var clavesTextos = []string{
	"limite", "uso_invalido", "configuracion_insegura", "configuracion_invalida",
	"salida_insegura", "acuse_inseguro", "preparacion_lista", "preparacion_fallida",
	"operacion_no_confirmada", "commit_no_confirmado", "acuse_no_guardado",
	"gobierno_confirmado", "gobierno_rechazado", "gobierno_no_disponible",
	"cadena_verificada", "cadena_rechazada", "verificacion_no_disponible",
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, operacionesPG(), time.Now))
}

func ejecutar(args []string, salida, errores io.Writer, ops operaciones, ahora func() time.Time) int {
	f := flag.NewFlagSet("vec-gobierno-usuarios-admin", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var fase, rutaConfig, rutaTextos, timeout, acuse string
	f.StringVar(&fase, "fase", "", "")
	f.StringVar(&rutaConfig, "config", "", "")
	f.StringVar(&rutaTextos, "textos", "", "")
	f.StringVar(&timeout, "timeout", "", "")
	f.StringVar(&acuse, "acuse", "", "")
	parseErr := f.Parse(args)
	textos, idioma, err := cargarTextos(rutaTextos)
	if err != nil {
		return informarFalloCatalogo(errores, err)
	}
	emitir := func(w io.Writer, d diagnostico, exit int) int {
		d.Mensaje = textos.T(idioma, d.Codigo)
		d.Limite = textos.T(idioma, "limite")
		if json.NewEncoder(w).Encode(d) != nil {
			return 2
		}
		return exit
	}
	fallo := func(codigo string) int { return emitir(errores, diagnostico{Codigo: codigo}, 1) }

	duracion, err := time.ParseDuration(timeout)
	faseValida := fase == "preparar" || fase == "aplicar" || fase == "verificar"
	acuseCoherente := (fase == "aplicar") == (acuse != "")
	if parseErr != nil || err != nil || duracion <= 0 || duracion > limiteTimeout || f.NArg() != 0 || !faseValida || !acuseCoherente || rutaConfig == "" || ops.incompletas() || ahora == nil {
		return fallo("uso_invalido")
	}
	if fase == "aplicar" && !nombreAcuseValido(acuse) {
		return fallo("acuse_inseguro")
	}
	b, err := bootstrap.LeerArchivoPrivadoDenominacionPersona(rutaConfig, limiteDocumento)
	if err != nil {
		return fallo("configuracion_insegura")
	}
	var cfg configuracionPrivada
	err = decodificarEstricto(b, &cfg)
	clear(b)
	if err != nil || cfg.validar(fase) != nil {
		return fallo("configuracion_invalida")
	}
	raiz, err := bootstrap.AbrirRaizPrivadaDenominacionPersona(filepath.Join(cfg.Salida, bootstrap.ArchivoConfiguracionGobiernoUsuarios))
	if err != nil {
		return fallo("salida_insegura")
	}
	defer raiz.Close()
	if fase == "aplicar" {
		// Comprobación temprana para no conectar si el destino ya existe; la
		// reserva exclusiva definitiva la hace la propia operación.
		if _, err := raiz.Lstat(acuse); !errors.Is(err, os.ErrNotExist) {
			return fallo("acuse_inseguro")
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), duracion)
	defer cancelar()

	switch fase {
	case "preparar":
		origen := bootstrap.MaterialOrigenGobiernoUsuariosAdmin{DirectorioMaterial: cfg.DirectorioMaterial, RutaConfiguracionHMAC: cfg.RutaConfiguracionHMAC, ArchivoSemillaRaiz: cfg.ArchivoSemillaRaiz, ValidezClaves: time.Duration(cfg.HorasValidezClaves) * time.Hour, ConjuntoVersion: cfg.ConjuntoCapacidades}
		r, err := ops.preparar(ctx, cfg.DSNLectura, duracion, origen, raiz)
		if err != nil {
			return fallo("preparacion_fallida")
		}
		return emitir(salida, diagnostico{Codigo: "preparacion_lista", Preparacion: &r}, 0)
	case "aplicar":
		a, err := ops.aplicar(ctx, cfg.DSNOperador, duracion, raiz, acuse)
		switch {
		case errors.Is(err, bootstrap.ErrCommitGobiernoUsuariosIndeterminado):
			return emitir(errores, diagnostico{Codigo: "commit_no_confirmado", Estado: "indeterminado"}, 2)
		case errors.Is(err, bootstrap.ErrAcuseGobiernoUsuariosNoGuardado):
			return emitir(errores, diagnostico{Codigo: "acuse_no_guardado", Estado: a.Estado, Confirmado: true, Replay: a.Codigo == "gobierno_usuarios_replay"}, 2)
		case err != nil:
			return fallo("operacion_no_confirmada")
		}
		d := diagnostico{Estado: a.Estado, Confirmado: true, AcuseGuardado: true, Replay: a.Codigo == "gobierno_usuarios_replay"}
		switch a.Estado {
		case "permitido":
			d.Codigo = "gobierno_confirmado"
			return emitir(salida, d, 0)
		case "denegado":
			d.Codigo = "gobierno_rechazado"
		default:
			d.Codigo = "gobierno_no_disponible"
		}
		return emitir(errores, d, 1)
	default:
		nombre := "verificacion-cadena-" + ahora().UTC().Format("20060102T150405Z") + ".json"
		informe, err := ops.verificar(ctx, cfg.DSNLectura, duracion, raiz, nombre)
		if err != nil {
			return fallo("verificacion_no_disponible")
		}
		if informe.Estado != "verificada" {
			return fallo("cadena_rechazada")
		}
		return emitir(salida, diagnostico{Codigo: "cadena_verificada"}, 0)
	}
}

// nombreAcuseValido exige un nombre base; una ruta se rechaza antes de leer
// nada. La operación vuelve a comprobarlo con su propia lista.
func nombreAcuseValido(nombre string) bool {
	return nombre != "" && len(nombre) <= 128 && nombre != "." && nombre != ".." && filepath.Base(nombre) == nombre
}

func cargarTextos(ruta string) (*i18n.Catalog, string, error) {
	if ruta == "" {
		return nil, "", os.ErrInvalid
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, "", err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || len(b) > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	var datos datosTextos
	if decodificarEstricto(b, &datos) != nil || datos.Idioma == "" || len(datos.Mensajes) != len(clavesTextos) {
		return nil, "", os.ErrInvalid
	}
	for _, k := range clavesTextos {
		if datos.Mensajes[k] == "" {
			return nil, "", os.ErrInvalid
		}
	}
	c, err := i18n.New(datos.Idioma, map[string]map[string]string{datos.Idioma: datos.Mensajes})
	return c, datos.Idioma, err
}

// Sin catálogo sólo se emite un código de protocolo, sin rutas ni mensajes raw.
func informarFalloCatalogo(w io.Writer, _ error) int {
	_ = json.NewEncoder(w).Encode(struct {
		Codigo string `json:"codigo"`
	}{Codigo: "catalogo_no_disponible"})
	return 2
}
