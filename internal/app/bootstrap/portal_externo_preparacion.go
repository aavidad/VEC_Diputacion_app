package bootstrap

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

// ErrPreparacionPortalExternoInvalida rechaza una preparación pedida fuera
// del lado interno, sin destino externo válido o con consumidores ajenos.
var ErrPreparacionPortalExternoInvalida = errors.New("bootstrap: preparacion del portal externo no valida")

// ResumenPreparacionPortalExterno describe lo preparado sin ningún secreto.
type ResumenPreparacionPortalExterno struct {
	Consumidores  []string
	Claves        int
	Configuracion string
	Alias         int
}

// PrepararMaterialPortalExterno es el paso del lado interno que habilita al
// proceso externo: con el rol de gobierno publica (de forma idempotente) las
// claves de las audiencias externas pedidas y deja en el material del
// proceso externo solo esas claves derivadas y la raíz de atestación. El
// proceso externo nunca recibe la clave base, la clave maestra ni el rol de
// gobierno. Se vuelve a ejecutar si cambia el material interno.
func PrepararMaterialPortalExterno(ctx context.Context, cfg config.Config, destino string, consumidores []string, seudonimos []byte) (ResumenPreparacionPortalExterno, error) {
	var resumen ResumenPreparacionPortalExterno
	cfg = cfg.Normalize()
	portal, err := separacionportales.Parsear(cfg.PortalProceso)
	if ctx == nil || err != nil || portal == separacionportales.PortalExterno || !cfg.DevelopmentEnabledByDoubleKey() ||
		!filepath.IsAbs(cfg.DevelopmentMaterialDir) || len(consumidores) == 0 {
		return resumen, ErrPreparacionPortalExternoInvalida
	}
	vistos := map[string]bool{}
	for _, c := range consumidores {
		if !consumidorPortalExternoValido(c) || vistos[c] {
			return resumen, ErrPreparacionPortalExternoInvalida
		}
		vistos[c] = true
	}
	// El destino tiene que ser un material de proceso externo ya separado y
	// distinto del interno; separacionportales comprueba ambos.
	if err := separacionportales.ComprobarMaterial(separacionportales.PortalExterno, destino); err != nil {
		return resumen, err
	}
	if directorioDentro(cfg.DevelopmentMaterialDir, destino) || directorioDentro(destino, cfg.DevelopmentMaterialDir) {
		return resumen, ErrPreparacionPortalExternoInvalida
	}
	raiz := cfg.DevelopmentMaterialDir
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return resumen, err
	}
	defer idempotencia.borrar()
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	if err != nil {
		return resumen, err
	}
	defer derivador.borrar()
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, time.Now())
	if err != nil {
		return resumen, err
	}
	defer base.borrarCopiasEfimeras()
	_, dsnGobierno, err := cfg.ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return resumen, err
	}
	ctxConexion, cancelar := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelar()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctxConexion, dsnGobierno,
		"vec-preparar-portal-externo", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		return resumen, err
	}
	defer gobierno.Close()
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterialPortalExternoV3())
	if err != nil {
		return resumen, err
	}
	publicados := map[string][]materialAtestacionContratacionTemporalDesarrollo{}
	defer func() {
		for _, lista := range publicados {
			borrarMaterialesV3PortalExterno(lista)
		}
	}()
	for _, consumidor := range consumidoresPortalExternoV3 {
		if !slices.Contains(consumidores, consumidor) {
			continue
		}
		for _, audiencia := range audienciasConsumidorPortalExternoV3(consumidor) {
			descriptor, ok := catalogo.descriptorPara(audiencia)
			if !ok {
				return resumen, ErrPreparacionPortalExternoInvalida
			}
			derivado, err := derivarMaterialConsumidorV3Desarrollo(base, descriptor)
			if err != nil {
				return resumen, err
			}
			// derivar comparte la clave privada con la base: se duplica para
			// que borrar cada material no deje a los demás sin ella.
			derivado.privada = append(derivado.privada[:0:0], base.privada...)
			derivado.spki = append([]byte(nil), base.spki...)
			if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctxConexion, gobierno, &derivado); err != nil {
				derivado.borrarCopiasEfimeras()
				return resumen, err
			}
			publicados[consumidor] = append(publicados[consumidor], derivado)
			resumen.Claves++
		}
		resumen.Consumidores = append(resumen.Consumidores, consumidor)
	}
	// Alias de las cuentas del Área personal, calculados por el propio
	// proceso externo con su clave (ExportarSeudonimosPortalExterno).
	if len(seudonimos) != 0 {
		s, err := leerSeudonimosPortalExterno(seudonimos)
		if err != nil {
			return resumen, err
		}
		if err := registrarSeudonimosPortalExterno(ctxConexion, gobierno, s); err != nil {
			return resumen, err
		}
		resumen.Alias = len(s.Cuentas)
	}
	if err := escribirMaterialV3PortalExterno(destino, publicados); err != nil {
		return resumen, err
	}
	// Relectura: lo escrito debe poder cargarse tal cual.
	inv, err := leerInventarioV3PortalExterno(destino)
	if err != nil {
		return resumen, err
	}
	resumen.Configuracion = inv.Configuracion.Revision
	return resumen, nil
}

// directorioDentro indica si hijo es padre o está debajo de él.
func directorioDentro(padre, hijo string) bool {
	rel, err := filepath.Rel(filepath.Clean(padre), filepath.Clean(hijo))
	return err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
