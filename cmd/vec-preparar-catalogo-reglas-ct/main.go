// vec-preparar-catalogo-reglas-ct prepara los bytes que CT190 conservará.
// Solo lee un paquete indicado por el operador; no abre conexiones ni publica.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	maximoFuente          = 4 << 20   // límite del lector de paquetes de catálogos existente
	maximoCanonico        = 90000     // límite del publicador CT190
	maximaVersionCT190    = 9_999_999 // límite decimal del publicador CT190
	maximaAprobacionCT190 = 200       // límite char_length de CT190
)

type fuenteCatalogo struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  *bool     `json:"demostracion"`
	Aviso         string    `json:"aviso"`
	OrigenSHA256  string    `json:"origen_sha256"`
}

var (
	patronRevision        = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	patronSHA256          = regexp.MustCompile(`^[a-f0-9]{64}$`)
	errRutaSalidaInvalida = errors.New("ruta de salida no controlada")
)

type paqueteCatalogo struct {
	VersionEsquema int                         `json:"version_esquema"`
	Fuente         fuenteCatalogo              `json:"fuente"`
	Catalogo       domain.CatalogoConfigurable `json:"catalogo"`
}

// manifiesto ata los dos ficheros exactos que recibe el publicador SQL.
type manifiesto struct {
	CatalogoID     string `json:"catalogo_id"`
	Version        int    `json:"version"`
	FuenteRef      string `json:"fuente_ref"`
	FuenteRevision string `json:"fuente_revision"`
	AprobacionRef  string `json:"aprobacion_ref,omitempty"`
	FuenteSHA256   string `json:"fuente_sha256"`
	CanonicoSHA256 string `json:"canonico_sha256"`
	Ejemplo        bool   `json:"ejemplo"`
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr))
}

func ejecutar(args []string, salida, errores io.Writer) int {
	flags := flag.NewFlagSet("vec-preparar-catalogo-reglas-ct", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var fuente, destino, destinoManifiesto, aprobacionRef string
	var permitirEjemplo bool
	flags.StringVar(&fuente, "fuente", "", "")
	flags.StringVar(&destino, "salida", "", "")
	flags.StringVar(&destinoManifiesto, "manifiesto", "", "")
	flags.StringVar(&aprobacionRef, "aprobacion-ref", "", "")
	flags.BoolVar(&permitirEjemplo, "permitir-ejemplo", false, "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || !rutaFuenteValida(fuente) ||
		fuente == destino || fuente == destinoManifiesto || destino == destinoManifiesto {
		return rechazar(errores, "catalogo_ct190_uso_invalido", 2)
	}
	if err := validarRutaSalida(destino); err != nil {
		return rechazar(errores, "catalogo_ct190_salida_invalida", 2)
	}
	if err := validarRutaSalida(destinoManifiesto); err != nil {
		return rechazar(errores, "catalogo_ct190_salida_invalida", 2)
	}
	contenido, err := leerFuente(fuente)
	if err != nil {
		return rechazar(errores, "catalogo_ct190_fuente_invalida", 2)
	}
	canonico, m, err := preparar(contenido, permitirEjemplo, aprobacionRef)
	if err != nil {
		return rechazar(errores, "catalogo_ct190_catalogo_invalido", 2)
	}
	if err := crearSalida(destino, canonico); err != nil {
		return rechazar(errores, "catalogo_ct190_salida_fallida", 1)
	}
	manifiestoJSON, err := json.Marshal(m)
	if err != nil || crearSalida(destinoManifiesto, append(manifiestoJSON, '\n')) != nil {
		_ = os.Remove(destino)
		return rechazar(errores, "catalogo_ct190_salida_fallida", 1)
	}
	resumen := m
	resumen.AprobacionRef = ""
	if json.NewEncoder(salida).Encode(resumen) != nil {
		return rechazar(errores, "catalogo_ct190_resumen_no_entregado", 1)
	}
	return 0
}

func preparar(contenido []byte, permitirEjemplo bool, aprobacionRef string) ([]byte, manifiesto, error) {
	var vacio manifiesto
	if len(contenido) == 0 || len(contenido) > maximoFuente ||
		validarJSONSinDuplicados(contenido) != nil ||
		validarClavesExactas(contenido, reflect.TypeOf(paqueteCatalogo{})) != nil {
		return nil, vacio, errors.New("fuente no válida")
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var paquete paqueteCatalogo
	if decodificador.Decode(&paquete) != nil || exigirFin(decodificador) != nil ||
		paquete.VersionEsquema != 1 || !patronRevision.MatchString(paquete.Fuente.Revision) ||
		paquete.Fuente.Demostracion == nil || !patronSHA256.MatchString(paquete.Fuente.OrigenSHA256) ||
		paquete.Fuente.ActualizadaEn.IsZero() || paquete.Fuente.ActualizadaEn.Location() != time.UTC ||
		paquete.Fuente.ActualizadaEn.Nanosecond()%1000 != 0 ||
		paquete.Fuente.Aviso != strings.TrimSpace(paquete.Fuente.Aviso) ||
		paquete.Catalogo.ID != reglas.CatalogoContratacionTemporal ||
		paquete.Catalogo.ModuloID != reglas.ModuloContratacionTemporal ||
		paquete.Catalogo.Estado != domain.EstadoCatalogoPublicado ||
		paquete.Catalogo.Version > maximaVersionCT190 ||
		utf8.RuneCountInString(paquete.Catalogo.AprobacionRef) > maximaAprobacionCT190 {
		return nil, vacio, errors.New("catálogo no publicable")
	}
	ejemplo := *paquete.Fuente.Demostracion || paquete.Catalogo.FuenteRef == reglas.MarcaPaqueteEjemplo
	if *paquete.Fuente.Demostracion != (paquete.Catalogo.FuenteRef == reglas.MarcaPaqueteEjemplo) ||
		(ejemplo && !permitirEjemplo) ||
		(!ejemplo && (aprobacionRef == "" || aprobacionRef != strings.TrimSpace(aprobacionRef) ||
			aprobacionRef != paquete.Catalogo.AprobacionRef)) {
		return nil, vacio, errors.New("fuente de ejemplo no admitida")
	}
	if err := reglas.ValidarCatalogoBaseReglas(paquete.Catalogo); err != nil {
		return nil, vacio, err
	}
	canonico, huella, err := reglas.CanonicoCatalogoBaseReglas(paquete.Catalogo)
	if err != nil || len(canonico) > maximoCanonico {
		return nil, vacio, errors.New("catálogo no válido")
	}
	sumaFuente := sha256.Sum256(contenido)
	return canonico, manifiesto{
		CatalogoID:     paquete.Catalogo.ID,
		Version:        paquete.Catalogo.Version,
		FuenteRef:      paquete.Catalogo.FuenteRef,
		FuenteRevision: paquete.Fuente.Revision,
		AprobacionRef:  paquete.Catalogo.AprobacionRef,
		FuenteSHA256:   hex.EncodeToString(sumaFuente[:]),
		CanonicoSHA256: huella,
		Ejemplo:        ejemplo,
	}, nil
}

func leerFuente(ruta string) ([]byte, error) {
	// #nosec G304 -- ruta absoluta facilitada por el operador; lectura acotada y fichero regular.
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximoFuente {
		return nil, errors.New("fuente no regular")
	}
	b, err := io.ReadAll(io.LimitReader(f, maximoFuente+1))
	if err != nil || len(b) == 0 || len(b) > maximoFuente {
		return nil, errors.New("fuente demasiado grande")
	}
	return b, nil
}

func rutaFuenteValida(ruta string) bool { return filepath.IsAbs(ruta) && filepath.Clean(ruta) == ruta }

func validarRutaSalida(ruta string) error {
	if !rutaFuenteValida(ruta) {
		return errRutaSalidaInvalida
	}
	padre, err := filepath.EvalSymlinks(filepath.Dir(ruta))
	if err != nil {
		return fmt.Errorf("%w: %w", errRutaSalidaInvalida, err)
	}
	if padre == "/tmp" || strings.HasPrefix(padre, "/tmp/") {
		return errRutaSalidaInvalida
	}
	info, err := os.Stat(padre)
	if err != nil {
		return fmt.Errorf("%w: %w", errRutaSalidaInvalida, err)
	}
	if !info.IsDir() || info.Mode().Perm()&0022 != 0 {
		return errRutaSalidaInvalida
	}
	return nil
}

func crearSalida(ruta string, contenido []byte) error {
	// #nosec G304 -- ruta absoluta en directorio controlado; O_EXCL impide sobrescritura y enlaces en la hoja.
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(contenido); err != nil {
		_ = f.Close()
		_ = os.Remove(ruta)
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(ruta)
		return err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(ruta)
		return err
	}
	return nil
}

func validarJSONSinDuplicados(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	if err := validarValorJSON(d); err != nil {
		return err
	}
	return exigirFin(d)
}

// encoding/json acepta nombres de campo sin distinguir mayúsculas. Para un
// artefacto probatorio se exige la grafía de los tags JSON; los atributos
// extensibles siguen siendo un mapa y se validan en el dominio.
func validarClavesExactas(contenido []byte, tipo reflect.Type) error {
	for tipo.Kind() == reflect.Pointer {
		tipo = tipo.Elem()
	}
	switch tipo.Kind() {
	case reflect.Struct:
		if tipo == reflect.TypeOf(time.Time{}) {
			return nil
		}
		var objeto map[string]json.RawMessage
		if json.Unmarshal(contenido, &objeto) != nil || objeto == nil {
			return errors.New("objeto JSON inválido")
		}
		campos := make(map[string]reflect.Type, tipo.NumField())
		for i := 0; i < tipo.NumField(); i++ {
			campo := tipo.Field(i)
			nombre := strings.Split(campo.Tag.Get("json"), ",")[0]
			if nombre != "" && nombre != "-" {
				campos[nombre] = campo.Type
			}
		}
		for nombre, valor := range objeto {
			tipoCampo, ok := campos[nombre]
			if !ok {
				return errors.New("nombre de campo JSON inválido")
			}
			if err := validarClavesExactas(valor, tipoCampo); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var valores []json.RawMessage
		if json.Unmarshal(contenido, &valores) != nil || valores == nil {
			return errors.New("lista JSON inválida")
		}
		for _, valor := range valores {
			if err := validarClavesExactas(valor, tipo.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

func validarValorJSON(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		vistas := make(map[string]struct{})
		for d.More() {
			claveToken, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok := claveToken.(string)
			if !ok {
				return errors.New("clave JSON inválida")
			}
			if _, existe := vistas[clave]; existe {
				return errors.New("clave JSON duplicada")
			}
			vistas[clave] = struct{}{}
			if err := validarValorJSON(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := validarValorJSON(d); err != nil {
				return err
			}
		}
	default:
		return errors.New("delimitador JSON inválido")
	}
	_, err = d.Token()
	return err
}

func exigirFin(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("contenido JSON sobrante")
}

func rechazar(w io.Writer, codigo string, estado int) int {
	_, _ = fmt.Fprintln(w, codigo)
	return estado
}
