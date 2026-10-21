package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type relojIdentidadDocumentalCTPrueba struct{ ahora time.Time }

func (r relojIdentidadDocumentalCTPrueba) Ahora() time.Time { return r.ahora }

func TestIdentidadDocumentalCTAsignacionOrdinariaExacta(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	c := configuracionIdentidadDocumentalCTPrueba()
	permiso := core.ConcesionRol{Accion: bp.AccionIdentidadDocumentoAceptacionCT, ModuloID: "bolsa",
		TipoRecurso:      bp.TipoIdentidadDocumentoAceptacionCT,
		Finalidades:      []string{bp.FinalidadIdentidadDocumentoAceptacionCT},
		GarantiaMinima:   core.AuthAssuranceHigh,
		CamposPermitidos: []string{"apellidos", "documento_enmascarado", "nombre", "version_fuente"}}
	i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(c.PrincipalID, c.PerfilActivoRef, ahora,
		"identidad_documental_ct_gestion_dev", "Gestión documental CT", "identidad_documental_ct_gestion_dev",
		[]core.ConcesionRol{permiso}, []core.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{c.UnidadRef}}})
	if err != nil {
		t.Fatal(err)
	}
	c.VersionRolRef = i.VersionRol.Referencia()
	a := &proveedorIdentidadDocumentalCT{config: c, reloj: relojIdentidadDocumentalCTPrueba{ahora}}
	base := instantaneaPublicadaDesarrollo{instantanea: i, actoAsignacion: c.ActoAsignacionRef}
	if !a.asignacionAdmitida(base) {
		t.Fatal("asignación documental exacta rechazada")
	}
	for _, caso := range []string{"acto", "perfil", "principal", "unidad", "rol", "concesion", "rol retirado", "revocacion", "version"} {
		otra := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i), actoAsignacion: c.ActoAsignacionRef}
		switch caso {
		case "acto":
			otra.actoAsignacion = "acto:ajeno"
		case "perfil":
			otra.instantanea.AsignacionPerfil.PerfilActivoRef = "prf_otra"
		case "principal":
			otra.instantanea.AsignacionPerfil.PrincipalID = "per_otra"
		case "unidad":
			otra.instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "unidad:ajena"
		case "rol":
			otra.instantanea.VersionRol.RolID = "tecnico_rrhh_desarrollo"
		case "concesion":
			otra.instantanea.VersionRol.Concesiones[0].Finalidades = []string{"preparar_incorporacion_ct"}
		case "rol retirado":
			otra.instantanea.VersionRol.Estado = core.EstadoVersionRolRetirada
		case "revocacion":
			otra.instantanea.ControlVigenciaVersionRol.Estado = core.EstadoControlVigenciaVersionRolRetirada
		case "version":
			otra.instantanea.AsignacionPerfil.VersionRolRef = "rol:ajeno:v1"
		}
		if a.asignacionAdmitida(otra) {
			t.Fatalf("%s usó autorización ajena", caso)
		}
	}
}

func TestIdentidadDocumentalCTRutaYMarcaPrivada(t *testing.T) {
	for _, ruta := range []string{httpct.RutaOriginalFirmableCT, httpct.RutaPropuestaFormalizacion, httpct.RutaResolucionFormalizacion} {
		if !rutaIdentidadDocumentalCT(ruta, http.MethodPost) || rutaIdentidadDocumentalCT(ruta, http.MethodGet) {
			t.Fatal("método documental no cerrado")
		}
	}
	if rutaIdentidadDocumentalCT(httpct.RutaConsultaDetalleRRHH, http.MethodPost) {
		t.Fatal("ruta de lectura general admitida")
	}
	for ruta, esperado := range map[string]struct{ clave, capacidad string }{
		httpct.RutaOriginalFirmableCT:      {"ct-original-firmable-preparar", ctports.AccionConsultarDetalleRRHH},
		httpct.RutaPropuestaFormalizacion:  {"ct-formalizacion-proponer", ctpostgres.AccionPropuestaFormalizacion},
		httpct.RutaResolucionFormalizacion: {"ct-formalizacion-resolver", ctpostgres.AccionResolucionFormalizacion},
	} {
		f := fronteraSeguridadComunDesarrollo{ruta: ruta, metodo: http.MethodPost,
			superficie: superficieInternaSeguridadComunDesarrollo,
			descriptor: descriptorFronteraComunDesarrollo{Clave: esperado.clave,
				ClaveCapacidad: esperado.capacidad, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo}}
		if !fronteraIdentidadDocumentalCTValida(f) {
			t.Fatal("frontera original cerrada rechazada")
		}
		f.descriptor.ClaveCapacidad = "contratacion_temporal.otra.consultar"
		if fronteraIdentidadDocumentalCTValida(f) {
			t.Fatal("capacidad ajena admitida")
		}
		f.descriptor.ClaveCapacidad = esperado.capacidad
		f.descriptor.Clave = "ct-ajena"
		if fronteraIdentidadDocumentalCTValida(f) {
			t.Fatal("frontera ajena admitida")
		}
	}
	borrador := fronteraSeguridadComunDesarrollo{ruta: plantillashttp.RutaBorradores, metodo: http.MethodPost,
		superficie: superficieInternaSeguridadComunDesarrollo,
		descriptor: descriptorFronteraComunDesarrollo{Clave: "ct-plantillas-documental-descargar",
			ClaveCapacidad: "contratacion_temporal.plantillas_documentos.documental_descargar",
			ClavePolitica:  "ct-plantillas-documental-v3"}}
	if !fronteraIdentidadDocumentalCTValida(borrador) {
		t.Fatal("borrador documental cerrado rechazado")
	}
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	c := configuracionIdentidadDocumentalCTPrueba()
	c.PrincipalID, c.CertificadoSHA256 = s.principalID, s.certificadoSHA256
	descriptor := descriptorFronteraComunDesarrollo{Clave: "ct-original-firmable-preparar",
		Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost,
		Ruta:               httpct.RutaOriginalFirmableCT,
		PerfilesActivosRef: []string{s.contexto.Resultado.Contexto.PerfilActivoRef},
		ClavePolitica:      clavePoliticaContratacionTemporalDesarrollo,
		ClaveCapacidad:     ctports.AccionConsultarDetalleRRHH}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{descriptor})
	if err != nil {
		t.Fatal(err)
	}
	sesion := &proveedorSesionConsultaRRHHDesarrollo{soporte: s, fronteras: catalogo}
	a := &proveedorIdentidadDocumentalCT{soporte: s, sesion: sesion, config: c}
	sesion.identidadDocumental = a
	marca := &marcaSesionIdentidadDocumentalCT{propietario: a, perfilRef: c.PerfilActivoRef,
		selector: bp.SelectorAceptacionNativaCT{UnidadRef: c.UnidadRef}}
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: s.sello, ruta: httpct.RutaOriginalFirmableCT,
		metodo: http.MethodPost, principal: principal}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx = context.WithValue(ctx, claveSesionIdentidadDocumentalCT{}, marca)
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpct.RutaOriginalFirmableCT,
		superficie: superficieInternaSeguridadComunDesarrollo,
		catalogo:   catalogo, descriptor: descriptor})
	if _, ok := a.marcaValida(ctx); !ok {
		t.Fatal("marca propia de Gestión rechazada")
	}
	marca.version = 1
	if perfil, ok := sesion.perfilActivoSeleccionado(ctx); !ok || perfil != c.PerfilActivoRef {
		t.Fatal("perfil nominal privado no seleccionado")
	}
	for _, caso := range []string{"sin marca", "otro dueño", "otro certificado", "otra unidad", "otra ruta"} {
		otro := ctx
		switch caso {
		case "sin marca":
			otro = context.WithValue(ctx, claveSesionIdentidadDocumentalCT{}, nil)
		case "otro dueño":
			forjada := *marca
			forjada.propietario = &proveedorIdentidadDocumentalCT{}
			otro = context.WithValue(ctx, claveSesionIdentidadDocumentalCT{}, &forjada)
		case "otro certificado":
			copia := capacidad
			copia.principal.Attributes = map[string]string{"certificate_sha256": "ajena"}
			otro = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, copia)
		case "otra unidad":
			forjada := *marca
			forjada.selector.UnidadRef = "unidad:ajena"
			otro = context.WithValue(ctx, claveSesionIdentidadDocumentalCT{}, &forjada)
		case "otra ruta":
			copia := capacidad
			copia.ruta = httpct.RutaConsultaDetalleRRHH
			otro = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, copia)
		}
		if _, ok := a.marcaValida(otro); ok {
			t.Fatalf("%s pasó marca privada", caso)
		}
	}
	if _, _, err := a.ExigirSolicitudLigadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{}); err == nil {
		t.Fatal("PDP documental directo sin marca privada")
	}
	if _, _, _, err := a.EmitirMaterialAutorizacionAtestadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{}); err == nil {
		t.Fatal("emisor documental directo sin marca privada")
	}
}
