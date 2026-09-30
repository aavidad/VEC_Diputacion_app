package bootstrap

import (
	"context"
	"testing"
	"time"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestReincorporacionTomaCausasPublicadasConMotivoV3Propio(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos(rutaCausasCeseEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	fuenteCese := fuenteReglasSeguimientoDesarrollo{reglas: resolutorReglasCTEjemploPrueba(t), causas: consulta}
	fuenteRetorno := fuenteReglasReincorporacionTitularDesarrollo{FuenteReglasSeguimiento: fuenteCese}
	instante := relojPresentacionReglasEjemplo.ahora
	causasCese, politicaCese, err := fuenteCese.CausasCese(context.Background(), instante)
	if err != nil {
		t.Fatal(err)
	}
	causasRetorno, politicaRetorno, err := fuenteRetorno.CausasCese(context.Background(), instante)
	if err != nil {
		t.Fatal(err)
	}
	if len(causasRetorno) != len(causasCese) || causasRetorno[0] != causasCese[0] ||
		politicaRetorno.DefinicionRef != politicaCese.DefinicionRef ||
		politicaRetorno.DefinicionVersion != politicaCese.DefinicionVersion ||
		politicaRetorno.DefinicionHuellaSHA256 != politicaCese.DefinicionHuellaSHA256 {
		t.Fatalf("CT130 debe conservar las causas y definición publicada: cese=%+v retorno=%+v", politicaCese, politicaRetorno)
	}
	if politicaCese.MotivoAutorizacion != motivoSeguimientoCeseDesarrollo(cthttp.RutaCesesNombramiento) ||
		politicaRetorno.MotivoAutorizacion != motivoSeguimientoCeseDesarrollo(cthttp.RutaReincorporacionesTitular) {
		t.Fatalf("motivos V3 cruzados: cese=%+v retorno=%+v", politicaCese.MotivoAutorizacion, politicaRetorno.MotivoAutorizacion)
	}
}

func TestReincorporacionExigeLecturaYEscrituraV3Distintas(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos(rutaCausasCeseEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	fuente := fuenteReglasReincorporacionTitularDesarrollo{FuenteReglasSeguimiento: fuenteReglasSeguimientoDesarrollo{
		reglas: resolutorReglasCTEjemploPrueba(t), causas: consulta}}
	_, politica, err := fuente.CausasCese(context.Background(), relojPresentacionReglasEjemplo.ahora)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre                          string
		lectura, antecedente, escritura bool
	}{
		{"tres concesiones", true, true, true},
		{"sin antecedente", true, false, true},
		{"sin consulta", false, true, true},
		{"sin escritura", true, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			contextoReincorporacion, err := nuevoContextoReincorporacionTitularDesarrollo(soporte, soporte.reloj.Ahora())
			if err != nil {
				t.Fatal(err)
			}
			v, err := contextoReincorporacion.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			var concesiones []vecdomain.ConcesionRol
			if caso.lectura {
				concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: accionConsultarSeguimientoCeseDesarrollo,
					ModuloID: ctports.ModuloContratacion, TipoRecurso: "seguimiento_contratacion_temporal",
					Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
			}
			if caso.escritura {
				concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: string(ctdomain.AccionRegistrarReincorporacionTitular),
					ModuloID: ctports.ModuloContratacion, TipoRecurso: ctports.TipoRecursoReincorporacionTitular,
					Finalidades: []string{ctports.FinalidadRegistrarReincorporacionTitular}, GarantiaMinima: vecdomain.AuthAssuranceHigh})
			}
			if caso.antecedente {
				concesiones = append(concesiones, vecdomain.ConcesionRol{Accion: string(ctports.AccionConsultarAntecedenteReincorporacionTitular),
					ModuloID: ctports.ModuloContratacion, TipoRecurso: ctports.TipoRecursoLecturaReincorporacionTitular,
					Finalidades:      []string{ctports.FinalidadLecturaReincorporacionTitular},
					CamposPermitidos: []string{"cese_evento_ref", "cese_recibo_ref", "documento_ref", "documento_sha256", "existe_cese", "fecha_efectiva", "relacion_ref"},
					GarantiaMinima:   vecdomain.AuthAssuranceHigh})
			}
			instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef,
				soporte.reloj.Ahora(), "reincorporacion_titular_ct_prueba", "Retorno titular", "retorno-titular-prueba", concesiones,
				[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
			if err != nil {
				t.Fatal(err)
			}
			soporte.reincorporacionTitular = &soporteSeguimientoCeseDesarrollo{
				instantanea: instantanea, contexto: contextoReincorporacion,
				contextoEsperadoRegistrado: contextoReincorporacion.Resultado,
				sesionOperativa:            proveedorSesionOperativaCTPrueba{contexto: contextoReincorporacion},
			}
			fijo, err := perfilFijoReincorporacionTitular(soporte)
			if err != nil {
				t.Fatal(err)
			}
			if err := soporte.registrarPerfilFijoCTDesarrollo(fijo); err != nil {
				t.Fatal(err)
			}
			central := soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
			central.asignaciones = map[string]instantaneaPublicadaDesarrollo{fijo.perfilRef(): {
				instantanea: instantanea, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo,
				actoControl: actoControlRolReincorporacionTitularDesarrollo}}
			servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(soporte, soporte, soporte, soporte,
				soporte.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
			if err != nil {
				t.Fatal(err)
			}
			autoridad := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte,
				autorizador: &autorizadorAnalisisContratacionTemporalDesarrollo{delegado: servicio, soporte: soporte, instalado: true}},
				lecturaReincorporacion: servicio}
			ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, cthttp.RutaReincorporacionesTitular)
			err = autoridad.AutorizarLecturaSeguimiento(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:1")
			if (err == nil) != caso.lectura {
				t.Fatalf("lectura=%v error=%v", caso.lectura, err)
			}
			recurso := recursoReincorporacionPerfilFijoPrueba("expediente:1")
			solicitud, _, _, _, err := autoridad.exigir(ctx, string(ctdomain.AccionRegistrarReincorporacionTitular),
				ctports.FinalidadRegistrarReincorporacionTitular, recurso)
			if (err == nil) != caso.escritura {
				t.Fatalf("escritura=%v error=%v", caso.escritura, err)
			}
			if caso.escritura {
				datos, err := solicitud.Datos()
				if err != nil || datos.ReferenciaMotivo != politica.MotivoAutorizacion {
					t.Fatalf("CT130 V3 usa motivo ajeno: datos=%+v error=%v", datos, err)
				}
			}
			antecedente := vecdomain.RecursoAutorizable{Referencia: "expediente:1", ModuloID: ctports.ModuloContratacion,
				Tipo:    ctports.TipoRecursoLecturaReincorporacionTitular,
				Ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo},
				Atributos: map[string]string{"version_expediente": "1", "relacion_ref": "relacion:1", "fecha_efectiva": "2026-09-20",
					"documento_ref": "documento:1", "documento_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
			_, _, _, _, err = autoridad.exigirLecturaAntecedenteReincorporacion(ctx, ctports.SolicitudAutorizarOperacionSeguimiento{
				Accion: ctports.AccionConsultarAntecedenteReincorporacionTitular, Finalidad: ctports.FinalidadLecturaReincorporacionTitular,
				Audiencia: ctports.AudienciaLecturaReincorporacionTitularV1,
				Motivo:    motivoSeguimientoCeseDesarrollo(cthttp.RutaReincorporacionesTitular), Recurso: antecedente})
			if (err == nil) != caso.antecedente {
				t.Fatalf("antecedente=%v error=%v", caso.antecedente, err)
			}
			ajena := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, cthttp.RutaCesesNombramiento)
			if err := autoridad.AutorizarLecturaSeguimiento(ajena, organizacionAltaContratacionTemporalDesarrollo, "expediente:1"); err == nil {
				t.Fatal("capacidad de ruta ajena admitida para retorno")
			}
			if central.preparadas != 0 || central.publicadas != 0 {
				t.Fatal("el perfil fijo publicó permisos durante una petición")
			}
		})
	}
}
