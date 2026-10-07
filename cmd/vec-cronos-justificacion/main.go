package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogojustificacion"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func leer(ruta string) (b []byte, err error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0) // #nosec G304 -- ruta explícita, fichero regular y SHA exigido.
	if err != nil {
		return nil, err
	}
	defer func() {
		if fallo := f.Close(); err == nil {
			err = fallo
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > catalogoefectos.LimiteJSON {
		return nil, os.ErrInvalid
	}
	b, err = io.ReadAll(io.LimitReader(f, catalogoefectos.LimiteJSON+1))
	if err != nil || len(b) != int(info.Size()) {
		return nil, os.ErrInvalid
	}
	return b, nil
}

type resultadoEscenario struct {
	Referencia string               `json:"referencia"`
	Anexo      domain.Justificacion `json:"anexo"`
	Revision   domain.Justificacion `json:"revision"`
}

type resultado struct {
	Demostracion bool   `json:"demostracion"`
	Aviso        string `json:"aviso"`
	Limite       string `json:"limite"`
	Etiquetas    struct {
		Anexo    string `json:"anexo"`
		Revision string `json:"revision"`
	} `json:"etiquetas"`
	Escenarios []resultadoEscenario `json:"escenarios"`
}

func ejecutar(args []string, salida io.Writer) error {
	f := flag.NewFlagSet("vec-cronos-justificacion", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	escenario := f.String("escenario", "", "")
	escenarioSHA := f.String("escenario-sha256", "", "")
	politica := f.String("politica", "", "")
	politicaSHA := f.String("politica-sha256", "", "")
	textos := f.String("textos", "", "")
	textosSHA := f.String("textos-sha256", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *escenario == "" || *escenarioSHA == "" || *politica == "" || *politicaSHA == "" || *textos == "" || *textosSHA == "" {
		return os.ErrInvalid
	}
	bPolitica, err := leer(*politica)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bPolitica)
	p, err := catalogojustificacion.CargarPolitica(bPolitica, *politicaSHA)
	if err != nil {
		return os.ErrInvalid
	}
	bTextos, err := leer(*textos)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bTextos)
	t, err := catalogojustificacion.CargarTextos(bTextos, *textosSHA)
	if err != nil {
		return os.ErrInvalid
	}
	bEscenario, err := leer(*escenario)
	if err != nil {
		return os.ErrInvalid
	}
	defer clear(bEscenario)
	e, err := catalogojustificacion.CargarEscenarios(bEscenario, *escenarioSHA)
	if err != nil {
		return os.ErrInvalid
	}
	r := resultado{Demostracion: true, Aviso: t.Aviso, Limite: t.Limite, Escenarios: make([]resultadoEscenario, 0, len(e.Escenarios))}
	r.Etiquetas.Anexo, r.Etiquetas.Revision = t.Anexo, t.Revision
	for _, x := range e.Escenarios {
		anexo, err := domain.PrepararAnexoJustificacion(x.Solicitud, p, nil, x.Vinculo, 0)
		if err != nil {
			return os.ErrInvalid
		}
		vinculoRevision := x.Vinculo
		if x.VinculoRevision != nil {
			vinculoRevision = *x.VinculoRevision
		}
		revision, err := domain.PrepararRevisionJustificacion(x.Solicitud, p, anexo, vinculoRevision, x.VersionEsperada, x.Decision, x.MotivoRef)
		if err != nil {
			return os.ErrInvalid
		}
		r.Escenarios = append(r.Escenarios, resultadoEscenario{Referencia: x.Referencia, Anexo: anexo, Revision: revision})
	}
	var buf bytes.Buffer
	if json.NewEncoder(&buf).Encode(r) != nil {
		return os.ErrInvalid
	}
	_, err = salida.Write(buf.Bytes())
	return err
}

func codigoSalida(err error) int {
	if err == nil {
		return 0
	}
	return 2
}

func run(args []string, salida, errores io.Writer) int {
	err := ejecutar(args, salida)
	if err == nil {
		return codigoSalida(err)
	}
	if fallo := json.NewEncoder(errores).Encode(struct {
		Codigo string `json:"codigo"`
	}{Codigo: "entrada_invalida"}); fallo != nil {
		return codigoSalida(fallo)
	}
	return codigoSalida(err)
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
