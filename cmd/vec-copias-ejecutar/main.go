// vec-copias-ejecutar compone un único ensayo sintético de copia completa.
// No acepta manifiestos, DSN ni rutas de origen desde stdin.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	captura "vec-diputacion-granada/internal/modules/administracion/adapters/capturafisica"
	contraste "vec-diputacion-granada/internal/modules/administracion/adapters/contrastecopias"
	destino "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	ej "vec-diputacion-granada/internal/modules/administracion/adapters/ejecucioncopias"
	fisica "vec-diputacion-granada/internal/modules/administracion/adapters/ensayofisicopg"
	logica "vec-diputacion-granada/internal/modules/administracion/adapters/ensayologicopg"
	registro "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	appcaptura "vec-diputacion-granada/internal/modules/administracion/application/capturacopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/ejecucioncopias"
	contrastedomain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

type configuracion struct {
	Origen              ej.ConfigOrigenLocal    `json:"origen"`
	Runtime             string                  `json:"runtime"`
	Peticion            puertos.Peticion        `json:"peticion"`
	Fisica              fisica.Configuracion    `json:"fisica"`
	Logica              logica.Configuracion    `json:"logica"`
	Contraste           contraste.Configuracion `json:"contraste"`
	CapturaFisica       captura.Configuracion   `json:"captura_fisica"`
	VentanaSegundos     int                     `json:"ventana_segundos"`
	LiberacionSegundos  int                     `json:"liberacion_segundos"`
	RaizTemporal        string                  `json:"raiz_temporal"`
	LimiteTotalBytes    int64                   `json:"limite_total_bytes"`
	Almacen             string                  `json:"almacen"`
	Catalogo            string                  `json:"catalogo"`
	Registro            string                  `json:"registro"`
	DiarioExterior      string                  `json:"diario_exterior"`
	RaicesRestauradas   []string                `json:"raices_restauradas"`
	ClaveMaestraFichero string                  `json:"clave_maestra_fichero"`
	ClaveRef            string                  `json:"clave_ref"`
	ClaveVersion        string                  `json:"clave_version"`
	PrepararFuente      bool                    `json:"preparar_fuente"`
}

type salida struct {
	Estado     string                `json:"estado"`
	Clave      string                `json:"clave"`
	Alcance    string                `json:"alcance"`
	Produccion bool                  `json:"produccion"`
	Recibo     *app.Recibo           `json:"recibo,omitempty"`
	Hechos     *ej.HechosFuenteLocal `json:"hechos,omitempty"`
	Consulta   int                   `json:"consulta,omitempty"`
	Motivos    []string              `json:"motivos,omitempty"`
}

type sondaLectura struct {
	origen    *ej.OrigenLocal
	consultas int
}

func (s *sondaLectura) EjecutarPostgreSQL(ctx context.Context, herramienta string, args []string, entrada []byte, limite int) ([]byte, error) {
	s.consultas++
	return s.origen.EjecutarPostgreSQL(ctx, herramienta, args, entrada, limite)
}
func (s *sondaLectura) ComprobarExclusion(ctx context.Context) (string, error) {
	return s.origen.ComprobarExclusion(ctx)
}

func main() {
	if codigo, ok := fisica.ManejarModoInterno(os.Args[1:], os.Stdin, os.Stdout); ok {
		os.Exit(codigo)
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	os.Exit(run(ctx, os.Args[1:], os.Stdout))
}

func run(ctx context.Context, args []string, out io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-ejecutar", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	ruta := flags.String("config", "", "")
	accion := flags.String("accion", "copiar", "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *ruta == "" {
		return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_argumentos"}, 2)
	}
	c, err := leer(*ruta)
	if err != nil {
		return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_configuracion"}, 2)
	}
	switch *accion {
	case "preparar":
		if err := ej.PrepararFuenteSintetica(ctx, c.Origen); err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_origen"}, 1)
		}
		return emitir(out, salida{Estado: "preparada", Clave: "copias_ejecutar_preparada"}, 0)
	case "medir":
		hechos, err := ej.MedirHechosFuenteLocal(ctx, c.Origen)
		if err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_origen"}, 1)
		}
		return emitir(out, salida{Estado: "medida", Clave: "copias_ejecutar_medida", Hechos: &hechos}, 0)
	case "revisar-origen":
		origen, err := ej.NuevoOrigenLocal(c.Origen)
		if err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_inventario"}, 1)
		}
		liberar, err := origen.Adquirir(ctx, c.Origen.OrigenRef)
		if err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_exclusion"}, 1)
		}
		defer liberar()
		if _, err := origen.ComprobarExclusion(ctx); err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_exclusion"}, 1)
		}
		if _, err := origen.Observar(ctx); err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_inventario"}, 1)
		}
		lector, err := contraste.Nuevo(c.Contraste)
		if err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_contraste"}, 1)
		}
		sonda := &sondaLectura{origen: origen}
		snapshot, err := lector.CapturarEjecutor(ctx, sonda, c.Origen.Base, sonda)
		if err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_contraste", Consulta: sonda.consultas}, 1)
		}
		if len(contrastedomain.Validar(snapshot)) != 0 {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_contraste", Consulta: sonda.consultas, Motivos: snapshot.Motivos}, 1)
		}
		return emitir(out, salida{Estado: "observada", Clave: "copias_ejecutar_observada"}, 0)
	case "copiar":
	default:
		return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_argumentos"}, 2)
	}
	if c.PrepararFuente {
		if err = ej.PrepararFuenteSintetica(ctx, c.Origen); err != nil {
			return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_origen"}, 1)
		}
		defer ej.RetirarFuenteSintetica(context.WithoutCancel(ctx), c.Origen)
	}
	s, err := servicio(c)
	if err != nil {
		return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_dependencia"}, 1)
	}
	r, err := s.Copiar(ctx, c.Peticion)
	if err != nil {
		return emitir(out, salida{Estado: "no_comprobable", Clave: "copias_ejecutar_fallida"}, 1)
	}
	return emitir(out, salida{Estado: "completada", Clave: "copias_ejecutar_completada", Recibo: &r}, 0)
}

func servicio(c configuracion) (*app.Servicio, error) {
	if c.Runtime == "" || c.Contraste.DSN != "" || c.Contraste.MaxBytes > 63<<20 || c.VentanaSegundos < 1 || c.LiberacionSegundos < 1 || c.LimiteTotalBytes < 1 || !filepath.IsAbs(c.RaizTemporal) {
		return nil, os.ErrInvalid
	}
	origen, err := ej.NuevoOrigenLocal(c.Origen)
	if err != nil {
		return nil, err
	}
	runtime, err := ej.CargarRuntimeLocal(c.Runtime)
	if err != nil {
		return nil, err
	}
	lector, err := contraste.Nuevo(c.Contraste)
	if err != nil {
		return nil, err
	}
	clave, err := clave(c.ClaveMaestraFichero)
	if err != nil {
		return nil, err
	}
	defer clear(clave[:])
	protector, err := bootstrap.NuevoProtectorCopiasDesarrollo(clave, c.ClaveRef, c.ClaveVersion, c.LimiteTotalBytes)
	if err != nil {
		return nil, err
	}
	fs, err := destino.NuevoFilesystem(destino.ConfiguracionFilesystem{Raiz: c.Almacen, MaximoClaroBytes: c.LimiteTotalBytes}, protector)
	if err != nil {
		return nil, err
	}
	catalogo, err := ej.AbrirCatalogo(c.Catalogo, c.RaicesRestauradas)
	if err != nil {
		_ = fs.Close()
		return nil, err
	}
	material := &ej.MaterialArchivos{RaizLogica: c.Origen.RaizLogica, Bases: []string{c.Origen.Base}, LimiteBytes: c.LimiteTotalBytes}
	control := origen.ControlFisico()
	capturador := &captura.Capturador{Config: c.CapturaFisica, Control: control}
	material.DirectorioFisico = func() string { return capturador.Directorio }
	medidor := ej.MedidorCS06{Lector: lector, Ejecutor: origen, Exclusion: origen, Base: c.Origen.Base, Ficheros: origen}
	proteccion := copias.Proteccion{Formato: "jwe-json", Algoritmo: "A256KW_A256GCM", ClaveRef: c.ClaveRef, ClaveVersion: c.ClaveVersion, AutenticacionRef: "indice:" + c.Peticion.ConjuntoRef, CifradoSHA256: strings.Repeat("0", 64)}
	puente := &ej.CapturaCS04{Servicio: appcaptura.Servicio{Exclusor: origen, Escritores: origen.Escritores(), Inventario: origen, Logico: origen, Componentes: capturador, Ahora: time.Now, TiempoLiberacion: time.Duration(c.LiberacionSegundos) * time.Second}, Medidor: medidor, Material: material, DuracionVentana: time.Duration(c.VentanaSegundos) * time.Second, Proteccion: proteccion, InventarioActual: origen}
	destinoCS03 := ej.DestinoCS03{Destino: fs, Fuente: material, Catalogo: catalogo, ProteccionEsperada: puente.Proteccion}
	r, err := registro.Abrir(registro.Config{Directorio: c.Registro, RaicesRestauradas: c.RaicesRestauradas})
	if err != nil {
		return nil, err
	}
	registroCS07, err := ej.AbrirRegistroCS07(ej.ConfigRegistroCS07{Registro: r, Destino: destinoCS03, DirectorioExterior: c.DiarioExterior, RaicesRestauradas: c.RaicesRestauradas})
	if err != nil {
		return nil, err
	}
	ensayo := ej.EnsayadorCS06{Fuente: destinoCS03, Fisico: c.Fisica, Logico: c.Logica, Lector: lector, Base: c.Origen.Base, Arrancador: runtime, Ficheros: ej.FicherosArchivados{MaxComponenteBytes: c.LimiteTotalBytes}, RaizTemporal: c.RaizTemporal, LimiteTotalBytes: c.LimiteTotalBytes}
	return app.NuevoCopia(app.Dependencias{Inventario: origen, Autorizador: origen, Registro: registroCS07, Ventana: puente, Destino: destinoCS03, Ensayador: ensayo})
}

func clave(ruta string) ([32]byte, error) {
	var k [32]byte
	if !filepath.IsAbs(ruta) {
		return k, os.ErrInvalid
	}
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- fichero local explícito; propietario, modo y tamaño se verifican antes de usarlo.
	if err != nil {
		return k, os.ErrInvalid
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() != 32 {
		return k, os.ErrInvalid
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != uint32(os.Geteuid()) || sys.Nlink != 1 { // #nosec G115 -- UID Linux corresponde a uid_t de 32 bits.
		return k, os.ErrInvalid
	}
	b := make([]byte, 32)
	defer clear(b)
	if _, err := io.ReadFull(f, b); err != nil {
		return k, os.ErrInvalid
	}
	var extra [1]byte
	if n, _ := f.Read(extra[:]); n != 0 {
		return k, os.ErrInvalid
	}
	post, err := f.Stat()
	if err != nil || !os.SameFile(st, post) || post.Size() != st.Size() {
		return k, os.ErrInvalid
	}
	copy(k[:], b)
	if k == [32]byte{} {
		return k, os.ErrInvalid
	}
	return k, nil
}
func leer(ruta string) (configuracion, error) {
	var c configuracion
	if !filepath.IsAbs(ruta) {
		return c, os.ErrInvalid
	}
	f, e := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- ruta local explícita del operador; O_NOFOLLOW, propietario, modo y tamaño se validan antes de decodificar.
	if e != nil {
		return c, e
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Mode().Perm() != 0600 || s.Size() < 2 || s.Size() > 4<<20 {
		return c, os.ErrInvalid
	}
	st, ok := s.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Geteuid()) || st.Nlink != 1 { // #nosec G115 -- UID Linux corresponde a uid_t de 32 bits.
		return c, os.ErrInvalid
	}
	d := json.NewDecoder(io.LimitReader(f, 4<<20))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF {
		return c, os.ErrInvalid
	}
	return c, nil
}
func emitir(w io.Writer, s salida, codigo int) int {
	s.Alcance = "ensemble_sintetico_local"
	s.Produccion = false
	if json.NewEncoder(w).Encode(s) != nil {
		return 2
	}
	return codigo
}
