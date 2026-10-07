package bootstrap

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
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
// OpcionesPreparacionPortalExterno reúne lo que decide el operador en el lado
// interno. CuentasAutorizadas es la lista positiva de cuentas del Área
// personal para las que se registran alias del espacio externo: el fichero de
// seudónimos lo produce el proceso externo y no es de confianza por sí solo.
type OpcionesPreparacionPortalExterno struct {
	Destino            string
	Consumidores       []string
	Seudonimos         []byte
	CuentasAutorizadas []string
}

func PrepararMaterialPortalExterno(ctx context.Context, cfg config.Config, opciones OpcionesPreparacionPortalExterno) (ResumenPreparacionPortalExterno, error) {
	var resumen ResumenPreparacionPortalExterno
	destino, consumidores, seudonimos := opciones.Destino, opciones.Consumidores, opciones.Seudonimos
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
	// Con la idempotencia del interno el externo podría derivar la raíz y la
	// clave base: su material debe tener claves propias.
	if err := exigirIdempotenciaPropiaPortalExterno(cfg.DevelopmentMaterialDir, destino); err != nil {
		return resumen, err
	}
	var aliasPedidos seudonimosPortalExterno
	if len(seudonimos) != 0 {
		aliasPedidos, err = seudonimosAutorizadosPortalExterno(cfg, seudonimos, opciones.CuentasAutorizadas)
		if err != nil {
			return resumen, err
		}
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
	ctxConexion, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(2*time.Minute))
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
	if len(aliasPedidos.Cuentas) != 0 {
		var err error
		if portal == separacionportales.PortalInterno {
			// ID7 conserva los alias en su población propia. La herramienta de
			// preparación coteja una provisión ya aprobada, nunca da de alta.
			err = comprobarSeudonimosProvisionadosPortalExterno(ctxConexion, gobierno, aliasPedidos)
		} else {
			err = registrarSeudonimosPortalExterno(ctxConexion, gobierno, aliasPedidos)
		}
		if err != nil {
			return resumen, err
		}
		resumen.Alias = len(aliasPedidos.Cuentas)
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

// seudonimosAutorizadosPortalExterno solo acepta alias de cuentas que el
// operador ha autorizado para el Área personal y que no pertenecen a la
// superficie corporativa de este proceso.
func seudonimosAutorizadosPortalExterno(cfg config.Config, contenido []byte, autorizadas []string) (seudonimosPortalExterno, error) {
	s, err := leerSeudonimosPortalExterno(contenido)
	if err != nil {
		return s, err
	}
	permitidas := map[string]bool{}
	for _, c := range autorizadas {
		if strings.HasPrefix(c, "cta_") {
			permitidas[c] = true
		}
	}
	// Sin configuración corporativa no hay nada que excluir; si existe pero no
	// se puede leer, se falla en lugar de omitir la exclusión en silencio.
	internas := map[string]bool{}
	rutaInterna := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombreConfiguracionPreferencias(core.SuperficieAutenticacionInternaCorporativaV1))
	if _, errFichero := os.Lstat(rutaInterna); errFichero == nil {
		c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
		if err != nil {
			return seudonimosPortalExterno{}, ErrSeudonimosPortalExternoInvalidos
		}
		for _, cuenta := range c.Cuentas {
			internas[cuenta.CuentaRef] = true
		}
	} else if !errors.Is(errFichero, os.ErrNotExist) {
		return seudonimosPortalExterno{}, ErrSeudonimosPortalExternoInvalidos
	}
	for _, c := range s.Cuentas {
		if !permitidas[c.CuentaRef] || internas[c.CuentaRef] {
			return seudonimosPortalExterno{}, ErrSeudonimosPortalExternoInvalidos
		}
	}
	return s, nil
}

// exigirIdempotenciaPropiaPortalExterno rechaza un material externo cuyas
// claves de idempotencia coincidan con las del interno.
func exigirIdempotenciaPropiaPortalExterno(interno, externo string) error {
	leer := func(raiz string) (map[[32]byte]bool, error) {
		huellas := map[[32]byte]bool{}
		entradas, err := os.ReadDir(filepath.Join(raiz, "idempotencia"))
		if err != nil {
			return nil, err
		}
		for _, e := range entradas {
			if !strings.HasSuffix(e.Name(), ".bin") {
				continue
			}
			contenido, err := leerFicheroMaterialSeguro(filepath.Join(raiz, "idempotencia", e.Name()), 1<<10)
			if err != nil {
				return nil, err
			}
			huellas[sha256.Sum256(contenido)] = true
			clear(contenido)
		}
		return huellas, nil
	}
	propias, err := leer(externo)
	if err != nil || len(propias) == 0 {
		return ErrPreparacionPortalExternoInvalida
	}
	internas, err := leer(interno)
	if err != nil {
		return ErrPreparacionPortalExternoInvalida
	}
	for huella := range propias {
		if internas[huella] {
			return ErrPreparacionPortalExternoInvalida
		}
	}
	return nil
}
