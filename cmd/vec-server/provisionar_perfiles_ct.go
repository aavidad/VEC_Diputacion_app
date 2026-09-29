package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

const maximoManifiestoPerfilesCT = 32 * 1024

var (
	huellaPerfilesCT     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	referenciaPerfilesCT = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)
)

type ejecutorProvisionPerfilesCT func(context.Context, config.Config, bootstrap.SolicitudProvisionPerfilesCT) (bootstrap.ResultadoProvisionPerfilesCT, error)

// ejecutarProvisionPerfilesCT no abre el servidor. Los errores tienen códigos
// fijos: ni los argumentos ni los errores del proveedor llegan a stdout/stderr.
func ejecutarProvisionPerfilesCT(ctx context.Context, args []string, salida, errores io.Writer, cfg config.Config, ejecutar ejecutorProvisionPerfilesCT) int {
	if ctx == nil || salida == nil || errores == nil || ejecutar == nil {
		return 2
	}
	solicitud, err := leerArgumentosProvisionPerfilesCT(args)
	if err != nil {
		fmt.Fprintln(errores, `{"error":"argumentos_invalidos"}`)
		return 2
	}
	if solicitud.Aplicar {
		if err := comprobarManifiestoPerfilesCT(solicitud.ManifiestoRuta, solicitud.ManifiestoSHA256); err != nil {
			fmt.Fprintln(errores, `{"error":"manifiesto_invalido"}`)
			return 2
		}
	}
	resultado, err := ejecutar(ctx, cfg, solicitud)
	if err != nil {
		fmt.Fprintln(errores, `{"error":"provision_rechazada"}`)
		return 1
	}
	if err := json.NewEncoder(salida).Encode(resultado); err != nil {
		fmt.Fprintln(errores, `{"error":"salida_no_disponible"}`)
		return 1
	}
	return 0
}

func leerArgumentosProvisionPerfilesCT(args []string) (bootstrap.SolicitudProvisionPerfilesCT, error) {
	f := flag.NewFlagSet("provisionar-perfiles-ct", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var s bootstrap.SolicitudProvisionPerfilesCT
	f.BoolVar(&s.Preparar, "preparar", false, "")
	f.BoolVar(&s.Aplicar, "aplicar", false, "")
	f.StringVar(&s.ManifiestoRuta, "manifiesto", "", "")
	f.StringVar(&s.ManifiestoSHA256, "sha256", "", "")
	f.StringVar(&s.AprobacionRef, "aprobacion-ref", "", "")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return bootstrap.SolicitudProvisionPerfilesCT{}, fmt.Errorf("argumentos_invalidos")
	}
	if s.Preparar && !s.Aplicar && s.ManifiestoRuta == "" && s.ManifiestoSHA256 == "" && s.AprobacionRef == "" {
		return s, nil
	}
	if s.Aplicar && !s.Preparar && s.ManifiestoRuta != "" && huellaPerfilesCT.MatchString(s.ManifiestoSHA256) && referenciaPerfilesCT.MatchString(s.AprobacionRef) {
		return s, nil
	}
	return bootstrap.SolicitudProvisionPerfilesCT{}, fmt.Errorf("argumentos_invalidos")
}

func comprobarManifiestoPerfilesCT(ruta, huellaEsperada string) error {
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("manifiesto_invalido")
	}
	f, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("manifiesto_invalido")
	}
	defer f.Close()
	abierto, err := f.Stat()
	if err != nil || !os.SameFile(info, abierto) {
		return fmt.Errorf("manifiesto_invalido")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maximoManifiestoPerfilesCT+1))
	if err != nil || n == 0 || n > maximoManifiestoPerfilesCT || hex.EncodeToString(h.Sum(nil)) != huellaEsperada {
		return fmt.Errorf("manifiesto_invalido")
	}
	return nil
}
