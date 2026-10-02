package bootstrap

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
	"time"
)

func perfilGobiernoReglasBaremoPrueba(t *testing.T) *PerfilGobiernoReglasBaremoV3 {
	t.Helper()
	base, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	p, err := NuevoPerfilGobiernoReglasBaremoV3(base, "convocatoria:baremo-prueba", "expediente:baremo-prueba", time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPerfilGobiernoReglasBaremoSoloConcedeAltaYLecturaExactas(t *testing.T) {
	p := perfilGobiernoReglasBaremoPrueba(t)
	if p.PerfilRef() == "" || p.PerfilRef() == p.soporte.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		p.plantilla.Validar() != nil || len(p.plantilla.VersionRol.Concesiones) != 3 ||
		len(p.plantilla.AsignacionPerfil.Ambitos) != 2 {
		t.Fatal("perfil fijo no segregado o plantilla inválida")
	}
	alta, lectura, recibo := p.plantilla.VersionRol.Concesiones[0], p.plantilla.VersionRol.Concesiones[1], p.plantilla.VersionRol.Concesiones[2]
	if alta.Accion != "bolsa.reglas_baremo.borrador.crear" || alta.TipoRecurso != "intencion_gobierno_reglas_baremo" ||
		!slices.Equal(alta.CamposPermitidos, []string{"auditoria", "estado_reglas_baremo", "salida_eventos"}) ||
		lectura.Accion != "bolsa.reglas_baremo.version.consultar" || lectura.TipoRecurso != "version_reglas_baremo_gobernada" ||
		!slices.Equal(lectura.CamposPermitidos, []string{"estado_reglas_baremo"}) ||
		recibo.Accion != "bolsa.reglas_baremo.recibo.consultar" || recibo.TipoRecurso != lectura.TipoRecurso ||
		!slices.Equal(recibo.CamposPermitidos, []string{"estado_reglas_baremo", "recibo"}) {
		t.Fatal("concesión distinta del contrato V3 mínimo")
	}
	for _, c := range p.plantilla.VersionRol.Concesiones {
		if c.Accion == "bolsa.llamamientos.crear" || c.Accion == "bolsa.reglas_baremo.publicar" {
			t.Fatal("el perfil de borrador prestó otro permiso")
		}
	}
	if _, err := NuevoPerfilGobiernoReglasBaremoV3(p.soporte, "", "expediente:baremo-prueba", time.Now()); err == nil {
		t.Fatal("ámbito vacío admitido")
	}
}

func TestDescriptorGobiernoReglasBaremoAudienciaNominal(t *testing.T) {
	d := DescriptorMaterialGobiernoReglasBaremoV3()
	if d.Audiencia != "vec_bolsa_reglas_baremo.gobierno_borrador.v3" || d.Dominio == "" || d.Prefijo == "" || d.ProveedorNominal == "" {
		t.Fatal("descriptor de material V3 incompleto")
	}
}

func TestGobiernoBaremoAudienciaPublicableConRaizComunYClavePropia(t *testing.T) {
	d := DescriptorMaterialGobiernoReglasBaremoV3()
	if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		t.Fatal("composición no permite publicar la audiencia nominal de baremo")
	}
	for _, audiencia := range []string{"", d.Audiencia + ".otra", "vec_bolsa_reglas_baremo.gobierno_borrador.v2", "vec_bolsa_reglas_baremo.*"} {
		cruzado := d
		cruzado.Audiencia = audiencia
		if audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(cruzado.Audiencia) {
			t.Fatalf("admitió audiencia fuera del descriptor: %q", audiencia)
		}
	}
	base := materialRenovableCTPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	propio, err := derivarMaterialConsumidorV3Desarrollo(base, d)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(propio.claveHMAC)
	ajeno, err := derivarMaterialConsumidorV3Desarrollo(base, descriptorMaterialMiBolsaDesarrollo())
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(ajeno.claveHMAC)
	if propio.audienciaConsumo != d.Audiencia || bytes.Equal(propio.claveHMAC, base.claveHMAC) ||
		bytes.Equal(propio.claveHMAC, ajeno.claveHMAC) || propio.claveHMACID == ajeno.claveHMACID ||
		!bytes.Equal(propio.spki, base.spki) || !reflect.DeepEqual(propio.raiz, base.raiz) ||
		!reflect.DeepEqual(propio.configuracion, base.configuracion) ||
		propio.configuracionRef != base.configuracionRef || propio.configuracionHuella != base.configuracionHuella {
		t.Fatal("descriptor prestó clave de otro consumidor o sustituyó la raíz común")
	}
}

func TestProvisionGobiernoReglasBaremoNoRestauraRevocacionesNiRecortes(t *testing.T) {
	p := perfilGobiernoReglasBaremoPrueba(t)
	actual := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.plantilla),
		actoAsignacion: actoAsignacionGobiernoReglasBaremoV3,
		actoControl:    actoControlGobiernoReglasBaremoV3,
		actualizadaPor: p.plantilla.AsignacionPerfil.EmitidaPor}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	huella, err := actual.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil || !preimagenProvisionableGobiernoReglasBaremoV3(actual, p, "aprobacion:sintetica", huella, ahora) {
		t.Fatal("preimagen propia aprobada fue rechazada")
	}
	for nombre, mutar := range map[string]func(*instantaneaPublicadaDesarrollo){
		"sin_aprobacion": func(a *instantaneaPublicadaDesarrollo) {},
		"otro_acto":      func(a *instantaneaPublicadaDesarrollo) { a.actoAsignacion = "acto:ajeno" },
		"rol_retirado":   func(a *instantaneaPublicadaDesarrollo) { a.instantanea.ControlVigenciaVersionRol.Estado = "retirada" },
		"campos_recortados": func(a *instantaneaPublicadaDesarrollo) {
			a.instantanea.VersionRol.Concesiones[2].CamposPermitidos = []string{"estado_reglas_baremo"}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := actual
			copia.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(actual.instantanea)
			mutar(&copia)
			aprobacion := "aprobacion:sintetica"
			if nombre == "sin_aprobacion" {
				aprobacion = ""
			}
			if preimagenProvisionableGobiernoReglasBaremoV3(copia, p, aprobacion, huella, ahora) {
				t.Fatal("preimagen restringida admitida")
			}
		})
	}
}
