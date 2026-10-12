package interna

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/documentosbase"
	"vec-diputacion-granada/internal/app/composicion/firmasct"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	pdfvec "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecapp "vec-diputacion-granada/internal/vec/application"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type montajeOriginalR5Gobernado struct {
	manejador http.Handler
	vincular  func(context.Context) (context.Context, error)
	cerrar    func() error
}

// montarOriginalR5Gobernado solo entrega manejador cuando gobierno, Documentos
// D13, catálogo, plantilla y almacén del fichero canónico están acreditados.
func montarOriginalR5Gobernado(ctx context.Context, directorio string, materialCT internactproveedores.Material,
	base *internactproveedores.Proveedores, fuente *internagobierno.FuenteF1, reloj relojGobiernoInterno) (resultado montajeOriginalR5Gobernado, seleccionado bool, err error) {
	var vacio montajeOriginalR5Gobernado
	if _, err := os.Stat(filepath.Join(directorio, "ct_firma_r5.json")); errors.Is(err, os.ErrNotExist) {
		return vacio, false, nil
	} else if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	if ctx == nil || ctx.Err() != nil || base == nil || base.DetalleOriginal == nil || base.OriginalPropuesta == nil || fuente == nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	cfg := config.Load()
	if !cfg.DevelopmentEnabledByDoubleKey() {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	material, err := internactproveedores.CargarMaterialFirmaR5(materialCT)
	if err != nil || cfg.ReglasEjemplo.CTPlantillasSourcePath == "" || material.PlantillasArchivo != cfg.ReglasEjemplo.CTPlantillasSourcePath ||
		cfg.PersonalOrganizacionSourcePath == "" || material.OrganizacionArchivo != cfg.PersonalOrganizacionSourcePath ||
		cfg.RPTCatalogoPath == "" || material.RPTArchivo != cfg.RPTCatalogoPath ||
		!huellaFicheroR5(material.OrganizacionArchivo, material.OrganizacionSHA256, 4<<20) ||
		!huellaFicheroR5(material.RPTArchivo, material.RPTSHA256, 4<<20) {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	plantillas, err := cargarPlantillasOriginalR5(ctx, material, reloj.Ahora())
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	etiquetar, err := firmasct.CargarEtiquetasOriginalR5(ctx, material.OrganizacionArchivo, material.RPTArchivo)
	if err != nil || !huellaFicheroR5(material.OrganizacionArchivo, material.OrganizacionSHA256, 4<<20) ||
		!huellaFicheroR5(material.RPTArchivo, material.RPTSHA256, 4<<20) {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	contenido, err := leerMaterialDocumentosR5(cfg.DevelopmentMaterialDir, material.DocumentoArchivo)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	info, err := internactproveedores.ExtraerDocumentoFirmaR5(contenido, material.DocumentoSHA256)
	clear(contenido)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	politicas, err := conservacion.NuevoCatalogoProvisionalV2(reloj)
	if err != nil || documentosbase.AdmitirCatalogoConservacion(cfg.DevelopmentEnabledByDoubleKey(), politicas.Provisional(), info.Almacen.Tipo, nil) != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	pool, err := internactproveedores.AbrirPoolDocumentosFirmaR5(ctx, info)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	correcto := false
	var cerrarAlmacen func() error
	defer func() {
		if !correcto {
			if cerrarAlmacen != nil {
				if cierreErr := cerrarAlmacen(); cierreErr != nil {
					err = errors.Join(err, ErrDependenciasProductivasNoDisponibles, cierreErr)
				}
			}
			pool.Close()
		}
	}()
	repositorio, err := docpg.NuevoRepositorio(pool)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	objetos, finalizar, err := documentosbase.AbrirAlmacen(ctx, info.Almacen, reloj)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	cerrarAlmacen = finalizar
	if documentosbase.AdmitirCatalogoConservacion(cfg.DevelopmentEnabledByDoubleKey(), politicas.Provisional(), info.Almacen.Tipo, objetos) != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	proveedor, err := internactproveedores.ConstruirProveedorFirmaR5(ctx, base, materialCT, material, fuente, reloj)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	clave, err := internactproveedores.CargarClaveSeudonimosFirmaR5(materialCT, material)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	autoridad, err := internactproveedores.NuevaAutoridadOriginalR5(proveedor, clave, politicas, reloj)
	clear(clave)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	defer func() {
		if !correcto {
			autoridad.Cerrar()
			proveedor.Cerrar()
		}
	}()
	lectura, err := docautorizacion.NuevaFabricaContextoLecturaOriginalV3(autoridad, reloj)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	documentos := &docapp.Servicio{Repositorio: repositorio, Almacen: objetos, Politicas: politicas, Reloj: reloj, ContextosLectura: lectura}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(politicas)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	custodia, err := almacen.NuevaCustodiaDocumentosOriginalCT(documentos, autoridad, almacen.FuncionMapeoExpedienteOriginalCT(ctapp.ReferenciaExpedienteDocumentalFormalizacion), tipos)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	// El PDF usa el mismo adaptador de formalización y la publicación exacta
	// de plantillas. No se registra firma ni resolución CT al prepararlo.
	render := renderizadorEtiquetadoOriginalR5{siguiente: informejuridico.RenderizadorBorradorDesarrollo{
		PDF: pdfvec.Renderizador{}, Plantillas: plantillas, Etiquetas: informejuridico.EtiquetadorReferencias(etiquetar)},
		etiquetar: etiquetar}
	origen, err := ctapp.NuevaFuenteOriginalFirmableRRHH(base.DetalleOriginal, base.OriginalPropuesta, render, tipos)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	servicio, err := vecapp.NuevoServicioOriginalFirmableCT(origen, custodia, tipos)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	manejador, err := httpinterno.NuevoManejadorOriginalFirmableCT(fuente, servicio)
	if err != nil {
		return vacio, true, ErrDependenciasProductivasNoDisponibles
	}
	correcto = true
	return montajeOriginalR5Gobernado{manejador: manejador, vincular: fuente.VincularContextoOriginalCT,
		cerrar: func() error { autoridad.Cerrar(); proveedor.Cerrar(); err := finalizar(); pool.Close(); return err }}, true, nil
}

// La publicación puede no contener todavía una unidad histórica. Un original
// sin sus tres nombres administrativos se deniega antes de generar o custodiar
// bytes; nunca imprime una referencia opaca como si fuera una etiqueta.
type renderizadorEtiquetadoOriginalR5 struct {
	siguiente ctports.RenderizadorBorradorRRHH
	etiquetar func(string) string
}

func (r renderizadorEtiquetadoOriginalR5) RenderizarBorrador(ctx context.Context, tipo ctports.TipoBorradorRRHH,
	detalle ctports.DetalleExpedienteRRHH) ([]byte, error) {
	if r.siguiente == nil || r.etiquetar == nil || detalle.Asignacion == nil ||
		r.etiquetar(detalle.Resumen.CentroRef) == "" || r.etiquetar(detalle.Resumen.CategoriaRef) == "" ||
		r.etiquetar(detalle.Asignacion.UnidadRef) == "" {
		return nil, ctports.ErrConsultaRRHHNoDisponible
	}
	return r.siguiente.RenderizarBorrador(ctx, tipo, detalle)
}

func cargarPlantillasOriginalR5(ctx context.Context, m internactproveedores.MaterialFirmaR5, ahora time.Time) (*informejuridico.PlantillasBorrador, error) {
	if ctx == nil || ctx.Err() != nil || ahora.IsZero() || !huellaFicheroR5(m.PlantillasArchivo, m.PlantillasSHA256, 4<<20) {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	consulta, err := fichero.NuevaConsultaCatalogos(m.PlantillasArchivo)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	versiones, err := consulta.ListarVersionesCatalogo(ctx, informejuridico.CatalogoPlantillasBorradorID)
	if err != nil || !huellaFicheroR5(m.PlantillasArchivo, m.PlantillasSHA256, 4<<20) {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	var publicada *vecdomain.CatalogoConfigurable
	for i := range versiones {
		if versiones[i].Estado == vecdomain.EstadoCatalogoPublicado &&
			(publicada == nil || versiones[i].Version > publicada.Version) {
			publicada = &versiones[i]
		}
	}
	if publicada == nil || publicada.ID != m.PlantillasID || publicada.Version != int(m.PlantillasVersion) {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	huella, err := publicada.HuellaSHA256()
	if err != nil || huella != m.PlantillasHuellaSHA256 {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	plantillas, err := informejuridico.NuevasPlantillasBorrador(*publicada, ahora)
	if err != nil || plantillas.Huella() != m.PlantillasHuellaSHA256 {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	return plantillas, nil
}

func huellaFicheroR5(ruta, esperada string, maximo int64) (valida bool) {
	f, err := os.Open(ruta)
	if err != nil {
		return false
	}
	defer func() {
		if f.Close() != nil {
			valida = false
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximo {
		return false
	}
	b, err := io.ReadAll(io.LimitReader(f, maximo+1))
	if err != nil || len(b) == 0 || int64(len(b)) > maximo {
		return false
	}
	suma := sha256.Sum256(b)
	return hex.EncodeToString(suma[:]) == esperada
}

func leerMaterialDocumentosR5(base, archivo string) (contenido []byte, err error) {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || archivo != "identidad/documentos.json" {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	defer func() {
		if cierreErr := raiz.Close(); cierreErr != nil {
			clear(contenido)
			contenido = nil
			err = errors.Join(err, ErrDependenciasProductivasNoDisponibles, cierreErr)
		}
	}()
	f, err := raiz.Open(archivo)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	defer func() {
		if cierreErr := f.Close(); cierreErr != nil {
			clear(contenido)
			contenido = nil
			err = errors.Join(err, ErrDependenciasProductivasNoDisponibles, cierreErr)
		}
	}()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > 256<<10 {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	b, err := io.ReadAll(io.LimitReader(f, 256<<10+1))
	if err != nil || len(b) == 0 || len(b) > 256<<10 {
		clear(b)
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	return b, nil
}
