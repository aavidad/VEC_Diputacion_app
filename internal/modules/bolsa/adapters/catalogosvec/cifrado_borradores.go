// Package catalogosvec adapta catalogos configurables gobernados por el nucleo
// a decisiones de seguridad del modulo Bolsa.
package catalogosvec

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const moduloCatalogoCifradoBorradores = "bolsa"

var ErrCatalogoCifradoBorradoresNoDisponible = errors.New("bolsa catalogos: politica de cifrado no disponible")

// ConfiguracionCatalogoCifradoBorradores fija una version exacta y la
// identidad tecnica que la consulta. La huella completa evita que una version
// con igual numero pero distinta publicacion se acepte por accidente.
type ConfiguracionCatalogoCifradoBorradores struct {
	CatalogoID    string
	Version       int
	HuellaSHA256  string
	ProveedorRef  string
	InstanciaRef  string
	CredencialRef string
	RolRef        string
}

type fuenteCatalogoCifradoBorradores interface {
	puertosvec.ConsultaCatalogosConfigurables
}

type catalogoCifradoBorradores struct {
	fuente    fuenteCatalogoCifradoBorradores
	id        string
	version   int
	huella    string
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador
}

// AutoridadPoliticaCifradoBorradoresCatalogo elige la politica aprobada. Es
// deliberadamente distinta del resolvedor tecnico que materializa el perfil.
type AutoridadPoliticaCifradoBorradoresCatalogo struct{ catalogo *catalogoCifradoBorradores }

// ResolvedorPerfilCifradoBorradoresCatalogo relee la misma version exacta y
// demuestra que el perfil coincide con la decision de politica ya emitida.
type ResolvedorPerfilCifradoBorradoresCatalogo struct{ catalogo *catalogoCifradoBorradores }

func NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
	proveedor fuenteCatalogoCifradoBorradores,
	configuracion ConfiguracionCatalogoCifradoBorradores,
) (*AutoridadPoliticaCifradoBorradoresCatalogo, error) {
	catalogo, err := nuevoCatalogoCifradoBorradores(proveedor, configuracion)
	if err != nil {
		return nil, err
	}
	return &AutoridadPoliticaCifradoBorradoresCatalogo{catalogo: catalogo}, nil
}

func NuevoResolvedorPerfilCifradoBorradoresCatalogo(
	proveedor fuenteCatalogoCifradoBorradores,
	configuracion ConfiguracionCatalogoCifradoBorradores,
) (*ResolvedorPerfilCifradoBorradoresCatalogo, error) {
	catalogo, err := nuevoCatalogoCifradoBorradores(proveedor, configuracion)
	if err != nil {
		return nil, err
	}
	return &ResolvedorPerfilCifradoBorradoresCatalogo{catalogo: catalogo}, nil
}

func nuevoCatalogoCifradoBorradores(
	proveedor fuenteCatalogoCifradoBorradores,
	configuracion ConfiguracionCatalogoCifradoBorradores,
) (*catalogoCifradoBorradores, error) {
	identidad, err := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		configuracion.ProveedorRef, configuracion.InstanciaRef,
		configuracion.CredencialRef, configuracion.RolRef,
	)
	if valorNuloCatalogoCifrado(proveedor) || err != nil || configuracion.CatalogoID == "" ||
		configuracion.Version < 1 || !huellaCatalogoCifradoValida(configuracion.HuellaSHA256) {
		return nil, ErrCatalogoCifradoBorradoresNoDisponible
	}
	return &catalogoCifradoBorradores{
		fuente: proveedor, id: configuracion.CatalogoID, version: configuracion.Version,
		huella: configuracion.HuellaSHA256, identidad: identidad,
	}, nil
}

func (a *AutoridadPoliticaCifradoBorradoresCatalogo) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	if a == nil || a.catalogo == nil {
		return gobiernoconvocatorias.IdentidadAutoridadBorrador{}
	}
	return a.catalogo.identidad
}

func (r *ResolvedorPerfilCifradoBorradoresCatalogo) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	if r == nil || r.catalogo == nil {
		return gobiernoconvocatorias.IdentidadAutoridadBorrador{}
	}
	return r.catalogo.identidad
}

func (a *AutoridadPoliticaCifradoBorradoresCatalogo) SeleccionarPoliticaCifradoBorrador(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudSeleccionPoliticaCifradoBorrador,
) (gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador, error) {
	if a == nil || a.catalogo == nil || solicitud.Validar() != nil {
		return gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	resolucion, err := a.catalogo.resolver(ctx, solicitud.Material.Accion, solicitud.SolicitadaEn)
	if err != nil {
		return gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador{}, err
	}
	validaHasta := limiteVigenciaCatalogoCifrado(
		solicitud.Control.ArrendamientoVenceEn, resolucion.entrada.VigenteHasta,
	)
	politica, err := gobiernoconvocatorias.NuevaPoliticaGobernadaCifradoBorrador(
		resolucion.perfil, solicitud, resolucion.decisionRef, resolucion.decisionVersion,
		resolucion.catalogo.Referencia(), uint64(resolucion.catalogo.Revision),
		resolucion.huellaCatalogo, resolucion.autoridadRef,
		solicitud.SolicitadaEn, solicitud.SolicitadaEn, validaHasta,
	)
	if err != nil {
		return gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	return politica, nil
}

func (r *ResolvedorPerfilCifradoBorradoresCatalogo) ResolverPerfilCifradoBorrador(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudResolucionPerfilCifradoBorrador,
) (gobiernoconvocatorias.ResolucionPerfilCifradoBorrador, error) {
	if r == nil || r.catalogo == nil || solicitud.Validar() != nil {
		return gobiernoconvocatorias.ResolucionPerfilCifradoBorrador{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	resolucion, err := r.catalogo.resolver(ctx, solicitud.Material.Accion, solicitud.SolicitadaEn)
	if err != nil || !politicaCoincideConCatalogoCifrado(solicitud.PoliticaEsperada, resolucion) {
		return gobiernoconvocatorias.ResolucionPerfilCifradoBorrador{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	validaHasta := limiteVigenciaCatalogoCifrado(
		solicitud.PoliticaEsperada.ValidaHasta, resolucion.entrada.VigenteHasta,
	)
	resultado, err := gobiernoconvocatorias.NuevaResolucionPerfilCifradoBorrador(
		resolucion.perfil, solicitud, resolucion.evidenciaRef, resolucion.evidenciaVersion,
		resolucion.verificadorRef, solicitud.SolicitadaEn, solicitud.SolicitadaEn, validaHasta,
	)
	if err != nil {
		return gobiernoconvocatorias.ResolucionPerfilCifradoBorrador{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	return resultado, nil
}

type resolucionEntradaCifradoBorradores struct {
	catalogo         dominiovec.CatalogoConfigurable
	entrada          dominiovec.EntradaCatalogoConfigurable
	huellaCatalogo   string
	perfil           gobiernoconvocatorias.PerfilCifradoBorrador
	decisionRef      string
	decisionVersion  uint32
	autoridadRef     string
	evidenciaRef     string
	evidenciaVersion uint32
	verificadorRef   string
}

func (c *catalogoCifradoBorradores) resolver(
	ctx context.Context, accion string, instante time.Time,
) (resolucionEntradaCifradoBorradores, error) {
	vacia := resolucionEntradaCifradoBorradores{}
	if c == nil || valorNuloCatalogoCifrado(c.fuente) || ctx == nil {
		return vacia, ErrCatalogoCifradoBorradoresNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	catalogo, err := c.fuente.ObtenerCatalogo(ctx, c.id, c.version)
	if err != nil || ctx.Err() != nil {
		return vacia, errorCatalogoCifrado(ctx)
	}
	catalogo, err = catalogo.ClonarCanonico()
	huella, errHuella := catalogo.HuellaSHA256()
	if err != nil || errHuella != nil || catalogo.ID != c.id || catalogo.Version != c.version ||
		catalogo.ModuloID != moduloCatalogoCifradoBorradores ||
		catalogo.Estado != dominiovec.EstadoCatalogoPublicado || instante.Before(catalogo.PublicadoEn) ||
		!textoCatalogoCifradoIgual(huella, c.huella) {
		return vacia, ErrCatalogoCifradoBorradoresNoDisponible
	}
	entrada, err := entradaCifradoBorradoresParaAccion(catalogo, accion, instante)
	if err != nil {
		return vacia, err
	}
	return proyectarEntradaCifradoBorradores(catalogo, entrada, huella)
}

var atributosEntradaCifradoBorradores = [...]string{
	"accion", "algoritmo_aead", "algoritmo_envoltura_clave", "autoridad_politica_ref",
	"decision_politica_ref", "decision_politica_version", "evidencia_perfil_ref",
	"evidencia_perfil_version", "perfil_referencia", "perfil_version", "verificador_perfil_ref",
}

func entradaCifradoBorradoresParaAccion(
	catalogo dominiovec.CatalogoConfigurable, accion string, instante time.Time,
) (dominiovec.EntradaCatalogoConfigurable, error) {
	var encontrada dominiovec.EntradaCatalogoConfigurable
	coincidencias := 0
	for _, entrada := range catalogo.Entradas {
		if entrada.VigenteEn(instante) && entrada.Atributos["accion"] == accion {
			encontrada = entrada
			coincidencias++
		}
	}
	if coincidencias != 1 || len(encontrada.Atributos) != len(atributosEntradaCifradoBorradores) {
		return dominiovec.EntradaCatalogoConfigurable{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	for _, atributo := range atributosEntradaCifradoBorradores {
		if encontrada.Atributos[atributo] == "" {
			return dominiovec.EntradaCatalogoConfigurable{}, ErrCatalogoCifradoBorradoresNoDisponible
		}
	}
	return encontrada, nil
}

func proyectarEntradaCifradoBorradores(
	catalogo dominiovec.CatalogoConfigurable,
	entrada dominiovec.EntradaCatalogoConfigurable,
	huellaCatalogo string,
) (resolucionEntradaCifradoBorradores, error) {
	perfilVersion, errPerfil := enteroCatalogoCifrado(entrada.Atributos["perfil_version"])
	decisionVersion, errDecision := enteroCatalogoCifrado(entrada.Atributos["decision_politica_version"])
	evidenciaVersion, errEvidencia := enteroCatalogoCifrado(entrada.Atributos["evidencia_perfil_version"])
	huellaPerfil := huellaPerfilCatalogoCifrado(
		entrada.Atributos["perfil_referencia"], perfilVersion,
		entrada.Atributos["algoritmo_aead"], entrada.Atributos["algoritmo_envoltura_clave"],
	)
	perfil, err := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(
		entrada.Atributos["perfil_referencia"], perfilVersion, huellaPerfil,
		entrada.Atributos["algoritmo_aead"], entrada.Atributos["algoritmo_envoltura_clave"],
	)
	if errPerfil != nil || errDecision != nil || errEvidencia != nil || err != nil {
		return resolucionEntradaCifradoBorradores{}, ErrCatalogoCifradoBorradoresNoDisponible
	}
	return resolucionEntradaCifradoBorradores{
		catalogo: catalogo, entrada: entrada, huellaCatalogo: huellaCatalogo, perfil: perfil,
		decisionRef: entrada.Atributos["decision_politica_ref"], decisionVersion: decisionVersion,
		autoridadRef: entrada.Atributos["autoridad_politica_ref"],
		evidenciaRef: entrada.Atributos["evidencia_perfil_ref"], evidenciaVersion: evidenciaVersion,
		verificadorRef: entrada.Atributos["verificador_perfil_ref"],
	}, nil
}

func politicaCoincideConCatalogoCifrado(
	politica gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador,
	resolucion resolucionEntradaCifradoBorradores,
) bool {
	perfil := politica.PerfilEsperado
	esperado := resolucion.perfil
	return politica.CatalogoRef == resolucion.catalogo.Referencia() &&
		politica.RevisionCatalogo == uint64(resolucion.catalogo.Revision) &&
		textoCatalogoCifradoIgual(politica.HuellaCatalogoSHA256, resolucion.huellaCatalogo) &&
		politica.DecisionPoliticaRef == resolucion.decisionRef &&
		politica.VersionDecisionPolitica == resolucion.decisionVersion &&
		politica.AutoridadRef == resolucion.autoridadRef &&
		perfil.Referencia == esperado.Referencia && perfil.Version == esperado.Version &&
		textoCatalogoCifradoIgual(perfil.HuellaContenidoSHA256, esperado.HuellaContenidoSHA256) &&
		perfil.AlgoritmoAEAD == esperado.AlgoritmoAEAD &&
		perfil.AlgoritmoEnvolturaClave == esperado.AlgoritmoEnvolturaClave
}

func huellaPerfilCatalogoCifrado(referencia string, version uint32, aead, envoltura string) string {
	representacion, err := json.Marshal(struct {
		Esquema    string `json:"esquema"`
		Referencia string `json:"referencia"`
		Version    uint32 `json:"version"`
		AEAD       string `json:"algoritmo_aead"`
		Envoltura  string `json:"algoritmo_envoltura_clave"`
	}{"vec.bolsa.perfil-cifrado.v1", referencia, version, aead, envoltura})
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(representacion)
	return hex.EncodeToString(suma[:])
}

func enteroCatalogoCifrado(valor string) (uint32, error) {
	numero, err := strconv.ParseUint(valor, 10, 32)
	if err != nil || numero == 0 {
		return 0, ErrCatalogoCifradoBorradoresNoDisponible
	}
	return uint32(numero), nil
}

func limiteVigenciaCatalogoCifrado(limite time.Time, catalogo time.Time) time.Time {
	if !catalogo.IsZero() && catalogo.Before(limite) {
		return catalogo
	}
	return limite
}

func errorCatalogoCifrado(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrCatalogoCifradoBorradoresNoDisponible
}

func huellaCatalogoCifradoValida(valor string) bool {
	bytes, err := hex.DecodeString(valor)
	return len(valor) == sha256.Size*2 && err == nil && len(bytes) == sha256.Size &&
		hex.EncodeToString(bytes) == valor
}

func textoCatalogoCifradoIgual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func valorNuloCatalogoCifrado(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

var (
	_ gobiernoconvocatorias.ProveedorPoliticaGobernadaCifradoBorrador = (*AutoridadPoliticaCifradoBorradoresCatalogo)(nil)
	_ gobiernoconvocatorias.ResolvedorPerfilCifradoBorrador           = (*ResolvedorPerfilCifradoBorradoresCatalogo)(nil)
)
