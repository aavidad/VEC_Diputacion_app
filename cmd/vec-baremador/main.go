package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

const maximoBytesArchivo = 16 * 1024 * 1024

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, simulacionbaremo.Servicio{})) }

func ejecutar(args []string, salida, diagnostico io.Writer, servicio simulacionbaremo.Simulador) int {
	flags := flag.NewFlagSet("vec-baremador", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	modo := flags.String("modo", "experiencia", "")
	reglas := flags.String("reglas", "", "")
	huellaReglas := flags.String("reglas-sha256", "", "")
	comparar := flags.Bool("comparar-reglas", false, "")
	reglasNuevas := flags.String("reglas-nuevas", "", "")
	huellaReglasNuevas := flags.String("reglas-nuevas-sha256", "", "")
	entrada := flags.String("entrada", "", "")
	huellaEntrada := flags.String("entrada-sha256", "", "")
	ejemplo := flags.String("ejemplo", "", "")
	listar := flags.Bool("listar-ejemplos", false, "")
	limite := flags.Int64("limite-bytes", maximoBytesArchivo, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return diagnosticar(diagnostico, "uso", "modo,reglas,reglas-sha256,entrada,entrada-sha256,limite-bytes", 0)
		}
		return diagnosticar(diagnostico, "argumentos_invalidos", "", 2)
	}
	if *comparar || *reglasNuevas != "" || *huellaReglasNuevas != "" {
		if !*comparar || (*modo != "meritos" && *modo != "experiencia" && *modo != "concursos") || flags.NArg() != 0 || *entrada != "" || *huellaEntrada != "" || *ejemplo != "" || *listar || *reglas == "" || *huellaReglas == "" || *reglasNuevas == "" || *huellaReglasNuevas == "" || *limite <= 0 || *limite > maximoBytesArchivo {
			return diagnosticar(diagnostico, "argumentos_invalidos", "comparacion", 2)
		}
		if *modo == "concursos" {
			return ejecutarDiferenciaConcursos(salida, diagnostico, *reglas, *huellaReglas, *reglasNuevas, *huellaReglasNuevas, *limite)
		}
		if *modo == "experiencia" {
			return ejecutarDiferenciaExperiencia(salida, diagnostico, *reglas, *huellaReglas, *reglasNuevas, *huellaReglasNuevas, *limite)
		}
		return ejecutarDiferencia(salida, diagnostico, *reglas, *huellaReglas, *reglasNuevas, *huellaReglasNuevas, *limite)
	}
	if *modo == "concursos" {
		return ejecutarConcursos(salida, diagnostico, *ejemplo, *listar, *reglas, *huellaReglas, *entrada, *huellaEntrada, *limite, flags.NArg())
	}
	if *ejemplo != "" || *listar {
		return diagnosticar(diagnostico, "argumentos_invalidos", "", 2)
	}
	if (*modo != "experiencia" && *modo != "meritos") || flags.NArg() != 0 || *reglas == "" || *entrada == "" || *huellaReglas == "" || *huellaEntrada == "" || *limite <= 0 || *limite > maximoBytesArchivo {
		return diagnosticar(diagnostico, "argumentos_invalidos", "", 2)
	}
	contenidoReglas, err := leerArchivoLimitado(*reglas, *limite)
	if err != nil {
		return diagnosticar(diagnostico, "archivo_invalido", "reglas", 2)
	}
	contenidoEntrada, err := leerArchivoLimitado(*entrada, *limite)
	if err != nil {
		return diagnosticar(diagnostico, "archivo_invalido", "entrada", 2)
	}
	solicitud := simulacionbaremo.Solicitud{
		ConjuntoCanonico: contenidoReglas, HuellaConjuntoSHA256: *huellaReglas,
		EntradaCanonica: contenidoEntrada, HuellaEntradaSHA256: *huellaEntrada,
	}
	var contenido []byte
	if *modo == "meritos" {
		var simulacion simulacionbaremo.SimulacionMeritos
		simulacion, err = (simulacionbaremo.ServicioMeritos{}).SimularMeritos(solicitud)
		contenido = simulacion.RepresentacionCanonica()
	} else {
		var simulacion simulacionbaremo.Simulacion
		simulacion, err = servicio.Simular(solicitud)
		contenido = simulacion.RepresentacionCanonica()
	}
	if err != nil {
		fase := "simulacion"
		var fallo *simulacionbaremo.Error
		if errors.As(err, &fallo) {
			fase = fallo.Fase
		}
		return diagnosticar(diagnostico, "simulacion_fallida", fase, 2)
	}
	if len(contenido) == 0 {
		return diagnosticar(diagnostico, "simulacion_fallida", "resultado", 2)
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		return diagnosticar(diagnostico, "salida_fallida", "", 2)
	}
	return 0
}

// O_NONBLOCK impide quedar esperando al abrir un FIFO. Solo se admiten
// archivos regulares y la lectura tiene límite incluso si cambian de tamaño.
func leerArchivoLimitado(ruta string, limite int64) (contenido []byte, err error) {
	if limite <= 0 || limite > maximoBytesArchivo {
		return nil, errors.New("limite_invalido")
	}
	// La ruta la elige el operador local y se abre con sus permisos del SO.
	// No procede de HTTP ni hay un directorio de recursos autorizado implícito.
	fichero, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cierre := fichero.Close(); cierre != nil && err == nil {
			contenido = nil
			err = cierre
		}
	}()
	estado, err := fichero.Stat()
	if err != nil {
		return nil, err
	}
	if !estado.Mode().IsRegular() || estado.Size() <= 0 || estado.Size() > limite {
		return nil, errors.New("archivo_invalido")
	}
	contenido, err = io.ReadAll(io.LimitReader(fichero, limite+1))
	if err != nil {
		return nil, err
	}
	if len(contenido) == 0 || int64(len(contenido)) > limite {
		return nil, errors.New("archivo_invalido")
	}
	return contenido, nil
}

func diagnosticar(salida io.Writer, codigo, fase string, retorno int) int {
	contenido, err := json.Marshal(struct {
		Codigo string `json:"codigo"`
		Fase   string `json:"fase,omitempty"`
	}{codigo, fase})
	if err != nil {
		slog.Error("baremador_diagnostico_no_emitido", "fase", "serializacion")
		return 2
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		slog.Error("baremador_diagnostico_no_emitido", "fase", "escritura")
		return 2
	}
	return retorno
}

func ejecutarConcursos(salida, diagnostico io.Writer, ejemplo string, listar bool, rutaReglas, shaReglas, rutaEntrada, shaEntrada string, limite int64, nArg int) int {
	if nArg != 0 || limite <= 0 || limite > maximoBytesArchivo || ((ejemplo != "" || listar) && (rutaReglas != "" || rutaEntrada != "" || shaReglas != "" || shaEntrada != "")) || (ejemplo != "" && listar) {
		return diagnosticar(diagnostico, "argumentos_invalidos", "concursos", 2)
	}
	var c domain.Configuracion
	var e domain.Entrada
	if listar || ejemplo != "" {
		ejemplos, err := simulacion.Ejemplos()
		if err != nil {
			return diagnosticar(diagnostico, "ejemplos_invalidos", "concursos", 2)
		}
		if listar {
			return emitirConcursos(salida, diagnostico, ejemplos)
		}
		encontrado := false
		for _, x := range ejemplos {
			if x.Referencia == ejemplo {
				c, e = x.Configuracion, x.Entrada
				encontrado = true
				break
			}
		}
		if !encontrado {
			return diagnosticar(diagnostico, "ejemplo_inexistente", "concursos", 2)
		}
	} else {
		if rutaReglas == "" || shaReglas == "" || rutaEntrada == "" || shaEntrada == "" {
			return diagnosticar(diagnostico, "argumentos_invalidos", "concursos", 2)
		}
		for _, x := range []struct {
			ruta, sha, campo string
			destino          any
		}{{rutaReglas, shaReglas, "reglas", &c}, {rutaEntrada, shaEntrada, "entrada", &e}} {
			datos, err := leerArchivoLimitado(x.ruta, limite)
			if err != nil {
				return diagnosticar(diagnostico, "archivo_invalido", x.campo, 2)
			}
			suma := sha256.Sum256(datos)
			if x.sha != hex.EncodeToString(suma[:]) {
				return diagnosticar(diagnostico, "huella_invalida", x.campo, 2)
			}
			if err := simulacion.Decodificar(bytes.NewReader(datos), x.destino); err != nil {
				return diagnosticar(diagnostico, "json_contrato_invalido", x.campo, 2)
			}
		}
	}
	resultado, err := application.Simular(c, e)
	if err != nil {
		campo := "calculo"
		var nominal *domain.Error
		if errors.As(err, &nominal) {
			campo = nominal.Codigo + ":" + nominal.Campo
		}
		return diagnosticar(diagnostico, "simulacion_fallida", campo, 2)
	}
	return emitirConcursos(salida, diagnostico, simulacion.Sobre(resultado))
}
func emitirConcursos(salida, diagnostico io.Writer, v any) int {
	contenido, err := json.Marshal(v)
	if err != nil {
		return diagnosticar(diagnostico, "salida_fallida", "concursos", 2)
	}
	n, err := salida.Write(contenido)
	if err != nil || n != len(contenido) {
		return diagnosticar(diagnostico, "salida_fallida", "concursos", 2)
	}
	return 0
}
