package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// ConfiguracionPreparacionIncorporacionB2 conserva el contrato del cargador de
// runtime. Exponerlo no concede permisos ni publica material o gobierno.
type ConfiguracionPreparacionIncorporacionB2 = archivoIncorporacionV2

// LeerConfiguracionPreparacionIncorporacionB2 admite exclusivamente B2 puro.
// El llamante conserva y cierra el Root para leer las rutas relativas privadas.
func LeerConfiguracionPreparacionIncorporacionB2(ruta string) (ConfiguracionPreparacionIncorporacionB2, *os.Root, error) {
	c, r, err := leerConfiguracionIncorporacionV2(ruta)
	if err != nil {
		return c, nil, err
	}
	if c.Planes != "" || c.Personal != "" || c.Continuidad != nil || c.PersonalB2 == nil {
		_ = r.Close()
		return c, nil, ct.ErrComposicionIncorporacionAplicacion
	}
	return c, r, nil
}

// DerivarClavesIncorporacionB2DesdeMaterialDesarrollo reutiliza los descriptores
// y la derivación del publicador. Devuelve material para las 22 operaciones;
// las operaciones de una misma audiencia usan exactamente la misma clave.
// Versiones y revisiones de esta derivación no acreditan gobierno publicado:
// el preparador debe sustituirlas por la fila vigente cotejada antes de escribir.
func DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(dir string, ahora time.Time) ([]ClaveCapacidadPersonalB2V3, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || filepath.Base(dir) != "idempotencia" || dentroDeRepositorioGit(dir) {
		return nil, fmt.Errorf("%w: ruta rechazada", errMaterialPersonalB2V3Desarrollo)
	}
	if p, err := filepath.EvalSymlinks(dir); err != nil || p != dir {
		return nil, fmt.Errorf("%w: enlace rechazada", errMaterialPersonalB2V3Desarrollo)
	}
	idem, err := cargarMaterialIdempotenciaDesarrollo(filepath.Dir(dir), filepath.Join(filepath.Dir(dir), config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return nil, fmt.Errorf("%w: carga rechazada", errMaterialPersonalB2V3Desarrollo)
	}
	defer idem.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idem)
	if err != nil {
		return nil, fmt.Errorf("%w: derivador rechazada", errMaterialPersonalB2V3Desarrollo)
	}
	defer derivador.borrar()
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		return nil, fmt.Errorf("%w: material rechazada", errMaterialPersonalB2V3Desarrollo)
	}
	defer m.borrarCopiasEfimeras()
	descriptores := append(descriptoresMaterialIncorporacionB2(), descriptoresMaterialPersonalB2Desarrollo()...)
	descriptores = append(descriptores, descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()...)
	porAudiencia := map[string]descriptorMaterialConsumidorV3Desarrollo{}
	for _, d := range descriptores {
		porAudiencia[d.Audiencia] = d
	}
	claves := make([]ClaveCapacidadPersonalB2V3, 0, len(operacionesIncorporacionB2()))
	fallo := func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}
	for _, op := range operacionesIncorporacionB2() {
		d, ok := porAudiencia[op.audiencia]
		if !ok {
			fallo()
			return nil, fmt.Errorf("%w: descriptor %s ausente", errMaterialPersonalB2V3Desarrollo, op.clave)
		}
		x, e := derivarMaterialConsumidorV3Desarrollo(m, d)
		if e != nil {
			fallo()
			return nil, fmt.Errorf("%w: derivación %s rechazada", errMaterialPersonalB2V3Desarrollo, op.clave)
		}
		claves = append(claves, ClaveCapacidadPersonalB2V3{CapacidadPublicadaPersonalB2V3: CapacidadPublicadaPersonalB2V3{
			DescriptorCapacidadPersonalB2V3: DescriptorCapacidadPersonalB2V3{op.clave, d.Audiencia, d.Dominio, d.Prefijo},
			ClaveID:                         x.claveHMACID, HuellaGobierno: x.claveHMACHuella, SHA256: x.claveHMACSecreto,
			EmisorID: x.emisorID, Desde: x.validaDesde, Hasta: x.validaHasta,
		}, secreto: x.claveHMAC})
	}
	return claves, nil
}

// ValidarMaterialPreparadoIncorporacionB2 usa los emisores reales de runtime.
func ValidarMaterialPreparadoIncorporacionB2(ruta string, ahora time.Time) error {
	c, r, err := LeerConfiguracionPreparacionIncorporacionB2(ruta)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, op := range operacionesIncorporacionB2() {
		if _, err := cargarEmisorOperacionIncorporacionB2(r, c.PersonalB2.Operaciones[op.clave].Capacidad, op.audiencia, relojPreparacionB2(ahora)); err != nil {
			return err
		}
	}
	return nil
}

type relojPreparacionB2 time.Time

func (r relojPreparacionB2) Ahora() time.Time { return time.Time(r) }

// ErrorMotivoPreparacionIncorporacionB2 contiene únicamente una clave del
// contrato cerrado. No conserva referencia de motivo, DSN ni error SQL.
type ErrorMotivoPreparacionIncorporacionB2 struct{ operacion string }

func (e *ErrorMotivoPreparacionIncorporacionB2) Error() string {
	return "motivo B2 " + e.operacion + ": esperado=motivo publicado vigente observado=no disponible o distinto"
}

// ValidarMotivosPreparacionIncorporacionB2 consulta la autoridad histórica real
// con el LOGIN nominal de motivos; no publica ni escoge referencias.
func ValidarMotivosPreparacionIncorporacionB2(ctx context.Context, c ConfiguracionPreparacionIncorporacionB2, raiz *os.Root, ahora time.Time) error {
	if ctx == nil || raiz == nil || c.PersonalB2 == nil || validarConfiguracionIncorporacionB2(c.PersonalB2) != nil {
		return &ErrorMotivoPreparacionIncorporacionB2{operacion: "motivos_autorizacion"}
	}
	b, err := leerArchivoIncorporacionV2(raiz, c.Pools["motivos_autorizacion"], 16<<10)
	if err != nil {
		return &ErrorMotivoPreparacionIncorporacionB2{operacion: "motivos_autorizacion"}
	}
	defer borrarBytes(b)
	p, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, strings.TrimSpace(string(b)), "vec-preparar-incorporacion-motivos", "vec_autorizacion_motivos_evaluador")
	if err != nil {
		return &ErrorMotivoPreparacionIncorporacionB2{operacion: "motivos_autorizacion"}
	}
	defer p.Close()
	ahora = ahora.UTC().Truncate(time.Microsecond)
	for _, op := range operacionesIncorporacionB2() {
		m := c.PersonalB2.Operaciones[op.clave].Motivo
		v, e := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(p, m.CatalogoID)
		if e != nil || v.ValidarReferenciaMotivoAutorizacionV2(ctx, m, ahora) != nil {
			return &ErrorMotivoPreparacionIncorporacionB2{operacion: op.clave}
		}
	}
	return nil
}
