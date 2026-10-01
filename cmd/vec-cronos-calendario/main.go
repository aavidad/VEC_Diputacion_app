// vec-cronos-calendario ejecuta únicamente un ensayo sintético por stdin.
package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sort"
	"time"

	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	cronos "vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

//go:embed textos.json
var textosJSON []byte

type fuenteVersion struct {
	VersionID            string                       `json:"version_id"`
	Fuente               ports.FuenteEnsayoCalendario `json:"fuente"`
	HuellaVersionConDias string                       `json:"huella_version_con_dias"`
}

type salida struct {
	ports.ResultadoEnsayoCalendario
	Idioma            string            `json:"idioma"`
	PersonaNombre     string            `json:"persona_nombre"`
	HuellaEntrada     string            `json:"huella_entrada"`
	FuentesCalendario []fuenteVersion   `json:"fuentes_calendario"`
	Textos            map[string]string `json:"textos"`
}

func main() { os.Exit(ejecutar(os.Stdin, os.Stdout, os.Stderr, len(os.Args)-1)) }

func ejecutar(in io.Reader, out, diagnostico io.Writer, argumentos int) int {
	var catalogos map[string]map[string]string
	if json.Unmarshal(textosJSON, &catalogos) != nil {
		return 1
	}
	fallar := func(codigo, idioma string) int {
		mensaje := catalogos[idioma][codigo]
		if mensaje == "" {
			mensaje = catalogos["es"][codigo]
		}
		_ = json.NewEncoder(diagnostico).Encode(map[string]any{"demostracion": true, "error": map[string]string{"codigo": codigo, "mensaje": mensaje}})
		return 1
	}
	if argumentos != 0 {
		return fallar("entrada_invalida", "es")
	}
	e, b, err := leerEntrada(in)
	if err != nil {
		return fallar("entrada_invalida", "es")
	}
	consulta, err := consultaEnsayo(e)
	if err != nil {
		return fallar("entrada_invalida", e.Idioma)
	}
	s, err := cronos.NuevoEnsayoCalendarioHistorico(consulta)
	if err != nil {
		return fallar("consulta_no_disponible", e.Idioma)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	r, err := s.Consultar(ctx, e.Solicitud)
	if err != nil {
		return fallar("consulta_no_disponible", e.Idioma)
	}
	usadas := map[string]bool{}
	for _, t := range r.Tramos {
		if t.Calendario != nil {
			for _, v := range t.Calendario.Versiones {
				usadas[v.ID] = true
			}
		}
	}
	fuentes := []fuenteVersion{}
	for _, v := range e.Calendarios.Versiones {
		if usadas[v.Calendario.Version.ID] {
			canon, _ := json.Marshal(v.Calendario)
			fuentes = append(fuentes, fuenteVersion{VersionID: v.Calendario.Version.ID, Fuente: v.Fuente, HuellaVersionConDias: huella(canon)})
		}
	}
	sort.Slice(fuentes, func(i, j int) bool { return fuentes[i].VersionID < fuentes[j].VersionID })
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if enc.Encode(salida{ResultadoEnsayoCalendario: r, Idioma: e.Idioma, PersonaNombre: e.PersonaNombre, HuellaEntrada: huella(b), FuentesCalendario: fuentes, Textos: catalogos[e.Idioma]}) != nil {
		return fallar("salida_no_disponible", e.Idioma)
	}
	return 0
}

func huella(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func fuenteValida(f ports.FuenteEnsayoCalendario) bool {
	b, err := hex.DecodeString(f.SHA256)
	return err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == f.SHA256 && cal.ReferenciaValida(f.Referencia)
}
